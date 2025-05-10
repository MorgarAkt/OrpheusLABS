package router

import (
	// Partials için eklendi

	"github.com/gin-gonic/gin"
	"github.com/morgarakt/aurify/internal/config"                 // Projenizin config yolu
	"github.com/morgarakt/aurify/internal/handlers"               // Projenizin handlers yolu
	middleware "github.com/morgarakt/aurify/internal/middlewares" // Projenizin middlewares yolu
	"github.com/morgarakt/aurify/internal/repository"             // Projenizin repository yolu
	"github.com/morgarakt/aurify/internal/services"               // Projenizin services yolu

	swaggerFiles "github.com/swaggo/files"     // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
)

type Router struct {
	engine         *gin.Engine
	repository     *repository.Repository
	config         *config.Config
	rabbitmqClient *services.RabbitMQClient
}

func NewRouter(repo *repository.Repository, cfg *config.Config, rmqClient *services.RabbitMQClient) *Router {
	r := &Router{
		engine:         gin.Default(),
		repository:     repo,
		config:         cfg,
		rabbitmqClient: rmqClient,
	}
	r.setupMiddlewares() // Önce middleware'ler ve template ayarları
	r.setupRoutes()      // Sonra route'lar
	return r
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) setupMiddlewares() {
	r.engine.LoadHTMLGlob("./web/templates/**/*")
	r.engine.Use(middleware.CORSMiddleware())
	r.engine.Use(middleware.OptionalAuthMiddleware(r.config.JWTSecret))
}

func (r *Router) setupRoutes() {
	authHandler := handlers.NewAuthHandler(r.repository, r.config)
	musicHandler := handlers.NewMusicHandler(r.repository, r.config)
	frontendHandler := handlers.NewFrontendHandler(r.repository, r.config, r.rabbitmqClient)

	r.engine.Static("/static", "./web/static")
	r.engine.Static("/generated", "./generated")
	r.engine.NoRoute(frontendHandler.NotFoundPage)

	r.engine.GET("/", frontendHandler.HomePage)
	r.engine.GET("/login", frontendHandler.Login)
	r.engine.GET("/register", frontendHandler.Register)
	r.engine.GET("/library", middleware.AuthMiddleware(r.config.JWTSecret), frontendHandler.Library)
	r.engine.GET("/explore", frontendHandler.Explore)

	// Müzik Detay Sayfası Route'u
	r.engine.GET("/musics/:id", musicHandler.GetMusicPage) // musicHandler'a yönlendirildi

	partials := r.engine.Group("/partials")
	{
		partials.GET("/edit-title-form", frontendHandler.GetEditTitleFormPartial)
		partials.GET("/title-text", frontendHandler.GetTitleTextPartial)
		// Yeni partial'lar için (like butonu, visibility toggle) buraya eklenebilir veya handler'lar direkt HTML dönebilir
		// partials.GET("/like-button/:id", musicHandler.GetLikeButtonPartial) // Örnek
	}

	admin := r.engine.Group("/admin")
	{
		admin.GET("/", func(c *gin.Context) { /* ... */ })
	}

	apiv1 := r.engine.Group("/api/v1")
	{
		apiv1.POST("/register", authHandler.RegisterHandler)
		apiv1.POST("/login", authHandler.LoginHandler)
		apiv1.POST("/logout", authHandler.LogoutHandler)

		apiv1.POST("/generate-music", frontendHandler.GenerateMusicHandler)
		apiv1.GET("/music", middleware.AuthMiddleware(r.config.JWTSecret), frontendHandler.GetMusicsLibrary)
		apiv1.GET("/explore-music-data", frontendHandler.GetExploreMusicData)

		// Music Management API (frontend_handler.go'daki SaveGeneratedMusicHandler'a ek olarak veya yerine)
		// Eğer SaveGeneratedMusicHandler genel bir "müziği güncelle" ise, onun PUT olması daha uygun olabilir.
		// Şimdilik frontend_handler.go'daki POST /save-music kalıyor.
		// apiv1.PUT("/music/:id/details", middleware.AuthMiddleware(r.config.JWTSecret), musicHandler.UpdateMusicDetailsHandler)

		apiv1.PUT("/music/:id/title", middleware.AuthMiddleware(r.config.JWTSecret), musicHandler.UpdateMusicTitleHandler)

		apiv1.POST("/music/:id/toggle-like", middleware.AuthMiddleware(r.config.JWTSecret), musicHandler.ToggleLikeMusic)
		apiv1.PUT("/music/:id/visibility", middleware.AuthMiddleware(r.config.JWTSecret), musicHandler.ToggleMusicVisibility)
	}

	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
