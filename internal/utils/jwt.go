package utils

import (
	"errors"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var revokedTokens = make(map[string]bool)
var mu sync.Mutex

func GenerateToken(c *gin.Context, userID, username, secret string, expiry time.Duration) error {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return err
	}

	c.SetCookie("token", tokenString, int(expiry.Seconds()), "/", "", false, true)
	return nil
}

func VerifyToken(c *gin.Context, secret string) (*Claims, error) {
	tokenString, err := c.Cookie("token")
	if err != nil {
		return nil, errors.New("missing token cookie")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	mu.Lock()
	defer mu.Unlock()
	if revokedTokens[tokenString] {
		return nil, errors.New("token has been revoked")
	}
	return claims, nil
}

func RefreshToken(c *gin.Context, secret string, newExpiry time.Duration) error {
	_, err := c.Cookie("token")
	if err != nil {
		return errors.New("missing token cookie")
	}

	claims, err := VerifyToken(c, secret)
	if err != nil {
		return err
	}

	newClaims := Claims{
		UserID:   claims.UserID,
		Username: claims.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(newExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	tokenString, err := newToken.SignedString([]byte(secret))
	if err != nil {
		return err
	}

	c.SetCookie("token", tokenString, int(newExpiry.Seconds()), "/", "", false, true)
	return nil
}

func RevokeToken(c *gin.Context) error {
	tokenString, err := c.Cookie("token")
	if err != nil {
		return errors.New("missing token cookie")
	}

	mu.Lock()
	defer mu.Unlock()
	revokedTokens[tokenString] = true

	c.SetCookie("token", "", -1, "/", "", false, true)
	return nil
}

func ExtractToken(c *gin.Context) string {
	token, err := c.Cookie("token")
	if err != nil {
		return ""
	}
	return token
}

func GetUsername(c *gin.Context) string {
	username, _ := c.Get("username")
	return username.(string)
}

func GetUserID(c *gin.Context) string {
	userID, _ := c.Get("userID")
	return userID.(string)
}
