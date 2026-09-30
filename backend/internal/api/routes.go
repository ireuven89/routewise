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

		// Public customer discovery routes (no auth)
		public := v1.Group("/public")
		{
			public.GET("/providers", h.Provider.SearchProviders)
			public.GET("/config/google-maps", h.Provider.GetPublicGoogleMapsConfig)

			// service requests (broadcast bidding)
			public.POST("/service-requests", h.ServiceRequest.Create)
			public.GET("/service-requests/:token", h.ServiceRequest.GetByToken)
			public.POST("/service-requests/:token/award", h.ServiceRequest.Award)
		}

		// Protected routes (org users and technicians)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/me", h.Auth.GetProfile)

			// Jobs a technician needs in the mobile app. Handlers limit technicians to the
			// jobs assigned to them; owners see the whole organization.
			protected.GET("/jobs", h.Job.GetAll)
			protected.GET("/jobs/:id", h.Job.GetByID)
			protected.PATCH("/jobs/:id/status", h.Job.UpdateStatus)

			//files
			protected.POST("/projects/:id/files", h.Files.Upload)
			protected.GET("projects/:id/files", h.Files.ListFiles)
			protected.GET("/files/:id", h.Files.GetFile)
			protected.DELETE("/files/:id", h.Files.DeleteFile)
		}

		// Technician-only: respond to an assigned job
		worker := v1.Group("")
		worker.Use(middleware.AuthMiddleware(), middleware.RequireWorker())
		{
			worker.POST("/jobs/:id/accept", h.Job.Accept)
			worker.POST("/jobs/:id/decline", h.Job.Decline)
		}

		// Organization users only (technician tokens get 403)
		orgOnly := v1.Group("")
		orgOnly.Use(middleware.AuthMiddleware(), middleware.RequireOrgUser())
		{
			// Google Maps configuration
			orgOnly.GET("/config/google-maps", h.Geocoding.GetFrontendConfig)

			//service calls
			orgOnly.POST("service_calls", h.Job.CreateServiceCall)

			// Jobs (create / edit / dispatch)
			orgOnly.POST("/jobs", h.Job.Create)
			orgOnly.PUT("/jobs/:id", h.Job.Update)
			orgOnly.DELETE("/jobs/:id", h.Job.Delete)
			orgOnly.PATCH("/jobs/:id/assign", h.Job.AssignTechnician)

			// Customers
			orgOnly.POST("/customers", h.Customer.Create)
			orgOnly.GET("/customers", h.Customer.GetAll)
			orgOnly.GET("/customers/:id", h.Customer.GetByID)
			orgOnly.PUT("/customers/:id", h.Customer.Update)
			orgOnly.DELETE("/customers/:id", h.Customer.Delete)

			// Technicians
			orgOnly.POST("/workers", h.Technician.Create)
			orgOnly.GET("/workers", h.Technician.GetAll)
			orgOnly.GET("/workers/:id", h.Technician.GetByID)
			orgOnly.PUT("/workers/:id", h.Technician.Update)
			orgOnly.DELETE("/workers/:id", h.Technician.Delete)

			// Organization settings (service area + pricing)
			orgOnly.PUT("/organization/service-area", h.Provider.UpdateServiceArea)
			orgOnly.PUT("/organization/service-offer", h.Provider.UpdateServiceOffer)

			// Dashboard stats
			orgOnly.GET("/dashboard/stats", h.Dashboard.GetStats)

			// Find-service bidding: org-side leads/bids
			orgOnly.GET("/leads", h.ServiceRequest.ListLeads)
			orgOnly.PUT("/leads/:id/bid", h.ServiceRequest.UpsertBid)
		}
	}
}
