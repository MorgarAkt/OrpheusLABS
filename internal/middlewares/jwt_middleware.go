// Modify: OrpheusLABS/internal/middlewares/jwt_middleware.go
package middleware

import (
	"net/http"
	"net/url" // URL işlemleri için eklendi

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morgarakt/aurify/internal/utils"
)

// OptionalAuthMiddleware sets context if token is valid, but doesn't block
func OptionalAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := utils.VerifyToken(c, secret)
		if err == nil && claims != nil {
			c.Set("userID", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("authenticated", true)
			// Admin rolü kontrolü gelecekte burada eklenebilir
			// Örneğin: c.Set("isAdmin", claims.IsAdmin)
		} else {
			c.Set("authenticated", false)
			c.Set("userID", "")   // Ensure userID is empty if not authenticated
			c.Set("username", "") // Ensure username is empty if not authenticated
			// c.Set("isAdmin", false)
		}
		c.Next()
	}
}

// AuthMiddleware requires a valid token and blocks if invalid/missing
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := utils.VerifyToken(c, secret)
		if err != nil || claims == nil {
			// Kullanıcı giriş yapmamışsa veya token geçersizse
			// Yönlendirme URL'sini al
			redirectURL := c.Request.URL.Path
			if c.Request.URL.RawQuery != "" {
				redirectURL += "?" + c.Request.URL.RawQuery
			}

			// Standart isimlendirme kullanıldı
			c.HTML(http.StatusUnauthorized, "error/unauthorized.html", gin.H{
				"title":           "Yetkisiz Erişim",
				"IsLoginRequired": true,                         // Bu sayfa için giriş gerektiğini belirt
				"IsAdminRequired": false,                        // Bu genel yetkilendirme, admin özel değil
				"RedirectURL":     url.QueryEscape(redirectURL), // Giriş sonrası yönlendirme için URL'yi encode et
				// Navbar için auth bilgisini de gönderelim (opsiyonel, base layout'a bağlı)
				"auth": false,
			})
			c.Abort() // Handler zincirini durdur
			return
		}

		// Admin kontrolü (gelecekte eklenecek)
		// if claims.Role != "admin" { // Örneğin claims içinde Role alanı varsa
		// 	c.HTML(http.StatusForbidden, "error/unauthorized.html", gin.H{
		// 		"title":           "Yönetici Yetkisi Gerekli",
		// 		"IsLoginRequired": false,
		// 		"IsAdminRequired": true,
		// 		"auth":            true, // Kullanıcı giriş yapmış ama admin değil
		// 		"username":        claims.Username,
		// 	})
		// 	c.Abort()
		// 	return
		// }

		// Set user info in context for downstream handlers
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("authenticated", true)
		// c.Set("isAdmin", claims.IsAdmin) // Admin rolü eklendiğinde
		c.Next()
	}
}

func IsAuthenticated(c *gin.Context) bool {
	authenticated, exists := c.Get("authenticated")
	if !exists {
		return false
	}
	isAuthenticated, ok := authenticated.(bool)
	return ok && isAuthenticated
}

// GetUserInfoFromContext retrieves user details and authentication status.
// It now also includes a placeholder for isAdmin status.
func GetUserInfoFromContext(c *gin.Context) (userID uuid.UUID, username string, auth bool /*, isAdmin bool */) {
	auth = IsAuthenticated(c)
	// isAdmin = c.GetBool("isAdmin") // Admin rolü eklendiğinde

	if auth {
		username = c.GetString("username")
		userIDStr := c.GetString("userID")
		var err error
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			auth = false
			username = ""
			// isAdmin = false
		}
	}
	return userID, username, auth //, isAdmin
}
