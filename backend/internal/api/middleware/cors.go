package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
	allowedOrigins := strings.Split(allowedOriginsStr, ",")

	//every request should get here
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if strings.TrimSpace(allowedOrigin) == origin {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}

}
