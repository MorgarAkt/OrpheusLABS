package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morgarakt/aurify/internal/config"
	"github.com/morgarakt/aurify/internal/handlers"
	middleware "github.com/morgarakt/aurify/internal/middlewares"
	"github.com/morgarakt/aurify/internal/repository"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Router struct {
	engine     *gin.Engine
	repository *repository.Repository
	config     *config.Config
}

func NewRouter(repo *repository.Repository, cfg *config.Config) *Router {
	r := &Router{
		engine:     gin.New(),
		repository: repo,
		config:     cfg,
	}
	r.setupMiddlewares()
	r.setupRoutes()
	return r
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) setupMiddlewares() {
	r.engine.Use(
		gin.Logger(),
		gin.Recovery(),
		middleware.CORSMiddleware(),
		middleware.OptionalAuthMiddleware(r.config.JWTSecret),
	)
}

func (r *Router) setupRoutes() {
	r.engine.LoadHTMLGlob("./web/templates/**/*")
	r.engine.Static("/static", "./web/static")

	authHandler := handlers.NewAuthHandler(r.repository, r.config)
	frontendHandler := handlers.NewFrontendHandler(r.repository, r.config)

	frontend := r.engine.Group("/")
	{
		frontend.GET("/", frontendHandler.HomePage)
		frontend.GET("/login", frontendHandler.Login)
		frontend.GET("/register", frontendHandler.Register)
		frontend.GET("/library", frontendHandler.Library)
		frontend.GET("/explore", frontendHandler.Library)
	}

	apiv1 := r.engine.Group("/api/v1")
	{
		apiv1.GET("/music", frontendHandler.GetMusicsLibrary)
		apiv1.POST("/register", authHandler.RegisterHandler)
		apiv1.POST("/login", authHandler.LoginHandler)
		apiv1.POST("/logout", authHandler.LogoutHandler)
	}

	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
