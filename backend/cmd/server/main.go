package main

import (
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/api"
	"github.com/ireuven89/routewise/internal/api/handlers"
	"github.com/ireuven89/routewise/internal/api/middleware"
	"github.com/ireuven89/routewise/internal/config"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/internal/service"
	"github.com/ireuven89/routewise/services"
	"github.com/joho/godotenv"
)

func main() {

	// Load environment variables
	if err := godotenv.Overload(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	db, err := config.InitDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Setup Gin router
	err = sentry.Init(sentry.ClientOptions{
		Dsn:         os.Getenv("SENTRY_DSN"),
		Environment: "production",
	})
	if err != nil {
		log.Printf("Sentry initialization failed: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	router := gin.Default()
	router.Use(sentrygin.New(sentrygin.Options{}))
	router.Use(middleware.Cors())

	//init repositories
	fileRepo := repository.NewFileRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	workerRepo := repository.NewWorkerRepository(db)
	organizationUser := repository.NewUserRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	jobRepo := repository.NewJobRepository(db, customerRepo)

	//initialize services
	s3Service, err := services.NewS3Service()
	authService := service.NewAuthService(workerRepo, otpRepo, organizationUser)
	workerService := service.NewWorkerService(workerRepo)
	jobService := service.NewJobService(jobRepo)
	customerService := service.NewCustomerService(customerRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	jobHandler := handlers.NewJobHandler(jobService)
	customerHandler := handlers.NewCustomerHandler(customerService)
	technicianHandler := handlers.NewWorkerHandler(workerService)
	filesHandler := handlers.NewFileHandler(fileRepo, jobRepo, s3Service)
	healthHandler := handlers.NewHealthHandler(db)

	h := handlers.NewHandlers(
		authHandler,
		jobHandler,
		customerHandler,
		technicianHandler,
		filesHandler,
		healthHandler,
	)

	// Setup routes
	api.SetupRoutes(router, *h)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err = router.Run(":" + port); err != nil {
		log.Printf("Failed to start server: %v", err)
		log.Fatal(err)
	}
}
