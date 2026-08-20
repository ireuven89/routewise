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
	orgRepo := repository.NewOrganizationRepository(db)
	paymentNotificationRepo := repository.NewPaymentNotificationRepository(db)

	//initialize services
	s3Service, err := services.NewS3Service()
	notificationService := service.NewTwilioNotificationService()
	authService := service.NewAuthService(workerRepo, otpRepo, organizationUser, orgRepo, notificationService)

	// Initialize Google Maps geocoding service
	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	geocodingService := service.NewGeocodingService(googleMapsAPIKey)

	workerService := service.NewWorkerService(workerRepo, geocodingService)
	paymentService := service.NewPaymentService(jobRepo, customerRepo, orgRepo, paymentNotificationRepo, notificationService)
	jobService := service.NewJobService(jobRepo, paymentService)
	customerService := service.NewCustomerService(customerRepo, geocodingService)
	providerService := service.NewProviderService(orgRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	jobHandler := handlers.NewJobHandler(jobService)
	customerHandler := handlers.NewCustomerHandler(customerService)
	technicianHandler := handlers.NewWorkerHandler(workerService)
	filesHandler := handlers.NewFileHandler(fileRepo, jobRepo, s3Service)
	healthHandler := handlers.NewHealthHandler(db)

	// Initialize geocoding handler with frontend API key
	googleMapsFrontendAPIKey := os.Getenv("GOOGLE_MAPS_FRONTEND_API_KEY")
	geocodingHandler := handlers.NewGeocodingHandler(googleMapsFrontendAPIKey)
	providerHandler := handlers.NewProviderHandler(providerService, googleMapsFrontendAPIKey)
	dashboardHandler := handlers.NewDashboardHandler(jobService)
	paymentHandler := handlers.NewPaymentHandler(paymentService)

	h := handlers.NewHandlers(
		authHandler,
		jobHandler,
		customerHandler,
		technicianHandler,
		filesHandler,
		healthHandler,
		geocodingHandler,
		providerHandler,
		dashboardHandler,
		paymentHandler,
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
