//go:build integration

// Package integration runs the real HTTP router, handlers, services and repositories
// against a throwaway Postgres database. Only external providers are faked: Twilio is
// replaced by a recorder, and no Google/AWS calls are made by the flows tested here.
//
// Run with:
//
//	TEST_DATABASE_URL=postgres://user:pass@localhost:5432/routewise_test?sslmode=disable \
//	JWT_SECRET=test go test -tags integration ./internal/integration/...
//
// The database is wiped on every run, so its name must end in "_test".
package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/api"
	"github.com/ireuven89/routewise/internal/api/handlers"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/internal/service"
	_ "github.com/lib/pq"
)

var (
	testDB     *sql.DB
	testServer *httptest.Server
	notifier   = &recordingNotifier{}
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Println("TEST_DATABASE_URL not set; skipping integration tests")
		os.Exit(0)
	}
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "integration-test-secret")
	}

	if err := mustBeTestDatabase(dsn); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Println("open db:", err)
		os.Exit(1)
	}
	if err := resetSchema(db); err != nil {
		fmt.Println("reset schema:", err)
		os.Exit(1)
	}
	if err := applyMigrations(db, "../../migrations"); err != nil {
		fmt.Println("apply migrations:", err)
		os.Exit(1)
	}
	testDB = db

	gin.SetMode(gin.TestMode)
	testServer = httptest.NewServer(newRouter(db))

	code := m.Run()

	testServer.Close()
	db.Close()
	os.Exit(code)
}

// mustBeTestDatabase refuses to run against anything that isn't clearly a test database,
// since resetSchema drops everything in it.
func mustBeTestDatabase(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("parse TEST_DATABASE_URL: %w", err)
	}
	name := strings.TrimPrefix(u.Path, "/")
	if !strings.HasSuffix(name, "_test") {
		return fmt.Errorf("refusing to wipe database %q: TEST_DATABASE_URL must point at a database whose name ends in _test", name)
	}
	return nil
}

func resetSchema(db *sql.DB) error {
	_, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`)
	return err
}

var migrationNumber = regexp.MustCompile(`^(\d+)_`)

// applyMigrations runs every migration in numeric order (001, 002, ..., 0010, 0011, ...).
// Plain string sorting would put 0010-0014 before 001_init_schema.sql.
func applyMigrations(db *sql.DB, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	number := func(path string) int {
		m := migrationNumber.FindStringSubmatch(filepath.Base(path))
		if m == nil {
			return 1 << 30
		}
		n, _ := strconv.Atoi(m[1])
		return n
	}
	sort.SliceStable(files, func(i, j int) bool {
		if number(files[i]) != number(files[j]) {
			return number(files[i]) < number(files[j])
		}
		return files[i] < files[j]
	})

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// newRouter wires the app the same way cmd/server/main.go does, minus Sentry and the
// providers these tests don't exercise (S3 files, Google geocoding).
func newRouter(db *sql.DB) *gin.Engine {
	workerRepo := repository.NewWorkerRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	userRepo := repository.NewUserRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	jobRepo := repository.NewJobRepository(db, customerRepo)
	orgRepo := repository.NewOrganizationRepository(db)
	serviceRequestRepo := repository.NewServiceRequestRepository(db, customerRepo, jobRepo)
	serviceRequestNotificationRepo := repository.NewServiceRequestNotificationRepository(db)

	authService := service.NewAuthService(workerRepo, otpRepo, userRepo, orgRepo)
	jobService := service.NewJobService(jobRepo)
	customerService := service.NewCustomerService(customerRepo, service.NewGeocodingService(""))
	providerService := service.NewProviderService(orgRepo)
	serviceRequestService := service.NewServiceRequestService(
		serviceRequestRepo, serviceRequestNotificationRepo, notifier, "http://frontend.test")

	h := handlers.NewHandlers(
		handlers.NewAuthHandler(authService),
		handlers.NewJobHandler(jobService),
		handlers.NewCustomerHandler(customerService),
		nil, // workers
		nil, // files (S3)
		handlers.NewHealthHandler(db),
		nil, // geocoding config
		handlers.NewProviderHandler(providerService, ""),
		handlers.NewDashboardHandler(jobService),
		handlers.NewServiceRequestHandler(serviceRequestService, "http://frontend.test"),
	)

	router := gin.New()
	api.SetupRoutes(router, *h)
	return router
}

// -----------------------------------------------------------------------
// Fake Twilio
// -----------------------------------------------------------------------

type sentMessage struct {
	Channel string // "sms" or "whatsapp"
	To      string
	Body    string
}

type recordingNotifier struct {
	mu   sync.Mutex
	sent []sentMessage
}

func (n *recordingNotifier) SendSMS(phone, message string) error {
	n.record("sms", phone, message)
	return nil
}

func (n *recordingNotifier) SendWhatsApp(phone, message string) error {
	n.record("whatsapp", phone, message)
	return nil
}

func (n *recordingNotifier) record(channel, to, body string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, sentMessage{Channel: channel, To: to, Body: body})
}

// messagesTo returns every message sent to phone so far.
func (n *recordingNotifier) messagesTo(phone string) []sentMessage {
	n.mu.Lock()
	defer n.mu.Unlock()
	var out []sentMessage
	for _, m := range n.sent {
		if m.To == phone {
			out = append(out, m)
		}
	}
	return out
}

// waitForMessage polls (notifications are sent from background goroutines) until a
// message to phone on channel containing substr arrives.
func waitForMessage(t *testing.T, phone, channel, substr string) sentMessage {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, m := range notifier.messagesTo(phone) {
			if m.Channel == channel && strings.Contains(m.Body, substr) {
				return m
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no %s to %s containing %q; got %+v", channel, phone, substr, notifier.messagesTo(phone))
	return sentMessage{}
}

// -----------------------------------------------------------------------
// HTTP helpers
// -----------------------------------------------------------------------

func doJSON(t *testing.T, method, path, token string, body interface{}, out interface{}) int {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, testServer.URL+"/api/v1"+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		t.Logf("%s %s -> %d: %s", method, path, resp.StatusCode, raw)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, raw, err)
		}
	}
	return resp.StatusCode
}

func expectStatus(t *testing.T, got, want int, what string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: expected HTTP %d, got %d", what, want, got)
	}
}
