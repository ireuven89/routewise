package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/api/handlers"
	"github.com/ireuven89/routewise/internal/api/middleware"
)

func SetupRoutes(router *gin.Engine, h handlers.Handlers) {
	// Health check
	router.GET("/health", h.Health.Check) // Detailed (replaces /health + /metrics)
	router.GET("/ready", h.Health.Ready)  // Kubernetes readiness probe
	router.GET("/live", h.Health.Live)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public auth routes
		v1.POST("/register", h.Auth.Register)
		v1.POST("/login", h.Auth.Login)
		v1.POST("/workers/request-otp", h.Auth.RequestWorkerOTP)
		v1.POST("/worker/verify-otp", h.Auth.VerifyWorkerOTP)

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/me", h.Auth.GetProfile)

			// Google Maps configuration
			protected.GET("/config/google-maps", h.Geocoding.GetFrontendConfig)

			//service calls
			protected.POST("service_calls", h.Job.CreateServiceCall)

			// Jobs
			protected.POST("/jobs", h.Job.Create)
			protected.GET("/jobs", h.Job.GetAll)
			protected.GET("/jobs/:id", h.Job.GetByID)
			protected.PUT("/jobs/:id", h.Job.Update)
			protected.DELETE("/jobs/:id", h.Job.Delete)
			protected.PATCH("/jobs/:id/assign", h.Job.AssignTechnician)
			protected.PATCH("/jobs/:id/status", h.Job.UpdateStatus)

			// Customers
			protected.POST("/customers", h.Customer.Create)
			protected.GET("/customers", h.Customer.GetAll)
			protected.GET("/customers/:id", h.Customer.GetByID)
			protected.PUT("/customers/:id", h.Customer.Update)
			protected.DELETE("/customers/:id", h.Customer.Delete)

			// Technicians
			protected.POST("/workers", h.Technician.Create)
			protected.GET("/workers", h.Technician.GetAll)
			protected.GET("/workers/:id", h.Technician.GetByID)
			protected.PUT("/workers/:id", h.Technician.Update)
			protected.DELETE("/workers/:id", h.Technician.Delete)

			//files
			protected.POST("/projects/:id/files", h.Files.Upload)
			protected.GET("projects/:id/files", h.Files.ListFiles)
			protected.GET("/files/:id", h.Files.GetFile)
			protected.DELETE("/files/:id", h.Files.DeleteFile)
		}
	}
}
