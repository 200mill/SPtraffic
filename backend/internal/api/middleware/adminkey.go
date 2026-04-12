package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminKey returns a middleware that requires the X-Admin-Key header to match key.
// If key is empty, all requests are rejected (admin routes disabled).
func AdminKey(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if key == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin endpoints are disabled"})
			return
		}
		if c.GetHeader("X-Admin-Key") != key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin key"})
			return
		}
		c.Next()
	}
}
