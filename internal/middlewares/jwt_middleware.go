package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/morgarakt/aurify/internal/utils"
)

func OptionalAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := utils.VerifyToken(c, secret)
		if err == nil && claims != nil {
			c.Set("userID", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("authenticated", true)
		} else {
			c.Set("authenticated", false)
		}
		c.Next()
	}
}

func IsAuthenticated(c *gin.Context) bool {
	authenticated, exists := c.Get("authenticated")
	if !exists {
		return false
	}
	return authenticated.(bool)
}

func GetUserInfoFromContext(c *gin.Context) (string, bool) {
	username := ""
	auth := false
	if IsAuthenticated(c) {
		auth = true
		username = utils.GetUsername(c)
	}
	return username, auth
}
