package api

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/api/handlers"
	"github.com/ireuven89/routewise/internal/api/middleware"
	"time"
)

func SetupRoutes(router *gin.Engine, db *sql.DB) {
	// Health check

	router.GET("/health", func(c *gin.Context) {
		err := db.Ping()
		if err != nil {
			c.JSON(503, gin.H{
				"status": "unhealthy",
				"error":  "database unreachable",
			})
			return
		}

		c.JSON(200, gin.H{
			"status":    "ok",
			"message":   "RouteWise API is running",
			"database":  "connected",
			"timestamp": time.Now(),
		})
	})

	router.GET("/metrics", func(c *gin.Context) {
		stats := db.Stats()
		c.JSON(200, gin.H{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
			"wait_count":       stats.WaitCount,
			"max_open":         stats.MaxOpenConnections,
		})
	})

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	jobHandler := handlers.NewJobHandler(db)
	customerHandler := handlers.NewCustomerHandler(db)
	technicianHandler := handlers.NewTechnicianHandler(db)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public auth routes
		v1.POST("/register", authHandler.Register)
		v1.POST("/login", authHandler.Login)

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/me", authHandler.GetProfile)

			// Jobs
			protected.POST("/jobs", jobHandler.Create)
			protected.GET("/jobs", jobHandler.GetAll)
			protected.GET("/jobs/:id", jobHandler.GetByID)
			protected.PUT("/jobs/:id", jobHandler.Update)
			protected.DELETE("/jobs/:id", jobHandler.Delete)
			protected.PATCH("/jobs/:id/assign", jobHandler.AssignTechnician)
			protected.PATCH("/jobs/:id/status", jobHandler.UpdateStatus)

			// Customers
			protected.POST("/customers", customerHandler.Create)
			protected.GET("/customers", customerHandler.GetAll)
			protected.GET("/customers/:id", customerHandler.GetByID)
			protected.PUT("/customers/:id", customerHandler.Update)
			protected.DELETE("/customers/:id", customerHandler.Delete)

			// Technicians
			protected.POST("/technicians", technicianHandler.Create)
			protected.GET("/technicians", technicianHandler.GetAll)
			protected.GET("/technicians/:id", technicianHandler.GetByID)
			protected.PUT("/technicians/:id", technicianHandler.Update)
			protected.DELETE("/technicians/:id", technicianHandler.Delete)

		}
	}
}
