package main

import (
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/api"
	"github.com/ireuven89/routewise/internal/api/middleware"
	"github.com/ireuven89/routewise/internal/config"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

func main() {

	// Load environment variables
	if err := godotenv.Load(); err != nil {
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

	// Setup routes
	api.SetupRoutes(router, db)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	router.Run(":" + port)
}
