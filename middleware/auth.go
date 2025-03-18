package middleware

import (
	"net/http"
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/open-cmuq/passport-backend/utils"
  "strconv"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			if err.Error() == "token has expired" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			}
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// AdminOnlyMiddleware restricts access to admin users
func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// OwnershipMiddleware ensures that a user can only update their own data (unless they are an admin)
func OwnershipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user ID from the token
		userID := c.GetUint("user_id")
		role := c.GetString("role")

		// Get the ID from the URL parameter
		idParam := c.Param("id")
		resourceID, err := strconv.Atoi(idParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			c.Abort()
			return
		}

		// Allow access if the user is an admin or the owner of the resource
		if role == "admin" || userID == uint(resourceID) {
			c.Next()
			return
		}

		// Deny access
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to update this resource"})
		c.Abort()
	}
}
