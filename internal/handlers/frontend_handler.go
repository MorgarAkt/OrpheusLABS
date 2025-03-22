package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/morgarakt/aurify/internal/config"
	middleware "github.com/morgarakt/aurify/internal/middlewares"
	"github.com/morgarakt/aurify/internal/repository"
	"github.com/morgarakt/aurify/internal/utils"
)

type FrontendHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewFrontendHandler(repo *repository.Repository, cfg *config.Config) *FrontendHandler {
	return &FrontendHandler{
		repo: repo,
		cfg:  cfg,
	}
}

func (h *FrontendHandler) HomePage(c *gin.Context) {
	username, auth := middleware.GetUserInfoFromContext(c)

	musicTypes, err := h.repo.MusicType.GetAll()
	if err != nil {
		println(err)
	}
	modelTypes, err := h.repo.ModelType.GetAll()
	if err != nil {
		println(err)
	}

	data := gin.H{
		"title":     "Aurify - Create an aura",
		"auth":      auth,
		"username":  username,
		"MusicType": musicTypes,
		"ModelType": modelTypes,
	}
	c.HTML(http.StatusOK, "home.html", data)
}

func (h *FrontendHandler) Login(c *gin.Context) {
	_, auth := middleware.GetUserInfoFromContext(c)
	if auth {
		c.Redirect(http.StatusOK, "/")
	}

	data := gin.H{
		"title": "Login to Aurify",
	}
	c.HTML(http.StatusOK, "login.html", data)
}

func (h *FrontendHandler) Register(c *gin.Context) {
	_, auth := middleware.GetUserInfoFromContext(c)
	if auth {
		c.Redirect(http.StatusOK, "/")
	}

	data := gin.H{
		"title": "Register to Aurify",
	}
	c.HTML(http.StatusOK, "register.html", data)
}

func (h *FrontendHandler) Library(c *gin.Context) {
	username, auth := middleware.GetUserInfoFromContext(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	if page < 1 {
		page = 1
	}

	musicTypes, err := h.repo.MusicType.GetAll()
	if err != nil {
		println(err)
	}

	musicList, err := h.repo.Music.GetAll()
	if err != nil {
		println(err)
	}

	perPage := 4

	searchQuery := c.Query("q")
	musicType := c.Query("musictype")

	musics, pagination := utils.GetLibrary(musicList, page, perPage, searchQuery, musicType)

	data := gin.H{
		"auth":       auth,
		"username":   username,
		"title":      "Your Music Library - Aurify",
		"MusicType":  musicTypes,
		"Music":      musics,
		"Pagination": pagination,
	}

	c.HTML(http.StatusOK, "library.html", data)
}

// TODO: Implement User Filter
func (h *FrontendHandler) GetMusicsLibrary(c *gin.Context) {
	// username, auth := middleware.GetUserInfoFromContext(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	if page < 1 {
		page = 1
	}

	musicList, err := h.repo.Music.GetAll()
	if err != nil {
		println(err)
	}

	perPage := 4

	searchQuery := c.Query("q")
	musicType := c.Query("musictype")

	musics, pagination := utils.GetLibrary(musicList, page, perPage, searchQuery, musicType)

	data := gin.H{
		"Music":      musics,
		"Pagination": pagination,
	}

	c.HTML(http.StatusOK, "musics-pagination", data)
}

func (h *FrontendHandler) Explore(c *gin.Context) {
	data := gin.H{
		"title": "Register to Aurify",
	}
	c.HTML(http.StatusOK, "explore.html", data)
}
