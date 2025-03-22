package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/morgarakt/aurify/internal/config"
	"github.com/morgarakt/aurify/internal/models"
	"github.com/morgarakt/aurify/internal/repository"
	"github.com/morgarakt/aurify/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewAuthHandler(repo *repository.Repository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		repo: repo,
		cfg:  cfg,
	}
}

type LoginRequest struct {
	Email    string `json:"email" form:"email" binding:"required,email"`
	Password string `json:"password" form:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Email    string `json:"email" form:"email" binding:"required,email"`
	Password string `json:"password" form:"password" binding:"required,min=8"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type RegisterResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *AuthHandler) LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request data"})
		return
	}

	user, err := h.repo.User.GetByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid credentials"})
		return
	}

	err = utils.GenerateToken(c, user.ID.String(), user.Username, h.cfg.JWTSecret, time.Hour*24)

	c.HTML(http.StatusCreated, "login-successful.html", err)
}

func (h *AuthHandler) RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBind(&req); err != nil {
		fmt.Printf("%v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request data"})
		return
	}

	_, err := h.repo.User.GetByEmail(req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, ErrorResponse{Error: "Email already in use"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to hash password"})
		return
	}

	user := &models.User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	if err := h.repo.User.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create user"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
		return
	}

	c.SetCookie("Token", tokenString, 3600, "/", "localhost", false, true)
	c.HTML(http.StatusCreated, "register-successful.html", err)
}

func (h *AuthHandler) LogoutHandler(c *gin.Context) {
	utils.RevokeToken(c)
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}
