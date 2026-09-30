package middleware

import "github.com/gin-gonic/gin"

func RequireOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("user_role")
		if role != "owner" {
			c.JSON(403, gin.H{"error": "Only owners can perform this action"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// UserTypeWorker is the user_type claim on tokens issued to technicians (mobile OTP login).
const UserTypeWorker = "worker"

// RequireOrgUser blocks technician (worker) tokens. Technicians only get the job endpoints
// the mobile app needs; everything else (customers, settings, creating/assigning jobs, …)
// is for organization users.
func RequireOrgUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("user_type") == UserTypeWorker {
			c.JSON(403, gin.H{"error": "Not available for technicians"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireWorker allows only technician tokens (e.g. accepting/declining an assigned job).
func RequireWorker() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("user_type") != UserTypeWorker {
			c.JSON(403, gin.H{"error": "Only technicians can perform this action"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireOwnerOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("user_role")
		if role != "owner" && role != "admin" {
			c.JSON(403, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}
		c.Next()
	}
}
