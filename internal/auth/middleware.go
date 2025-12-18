package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Middleware 认证中间件
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		xApiKey := c.GetHeader("x-api-key")
		xGoogApiKey := c.GetHeader("X-Goog-Api-Key")
		authHeader := c.GetHeader("Authorization")
		keyParam := c.Query("key")

		token := xApiKey
		if token == "" {
			token = xGoogApiKey
		}
		if token == "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		if token == "" {
			token = keyParam
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": map[string]any{
					"message": "Missing authentication. Provide Authorization header, x-api-key, or ?key= parameter",
					"type":    "authentication_error",
				},
			})
			c.Abort()
			return
		}

		cache, err := GetOrRefreshToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": map[string]any{
					"message": fmt.Sprintf("Token refresh failed: %v", err),
					"type":    "authentication_error",
				},
			})
			c.Abort()
			return
		}

		c.Set("accessToken", cache.AccessToken)
		c.Set("projectID", cache.ProjectID)
		c.Set("email", cache.Email)
		c.Next()
	}
}
