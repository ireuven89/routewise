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
