package router

import (
	"net/http" // Partials için eklendi

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
	// FuncMap burada tanımlanabilir, ancak dict fonksiyonuna ihtiyacımız kalmadı.
	// r.engine.SetFuncMap(template.FuncMap{ ... })

	r.engine.LoadHTMLGlob("./web/templates/**/*")
	r.engine.Use(middleware.CORSMiddleware())
	r.engine.Use(middleware.OptionalAuthMiddleware(r.config.JWTSecret)) // JWTSecret config'den alınmalı
}

func (r *Router) setupRoutes() {
	// --- Static Routes ---
	r.engine.Static("/static", "./web/static")
	r.engine.Static("/generated", "./generated") // Üretilen müzik dosyaları için

	// --- Initialize Handlers ---
	authHandler := handlers.NewAuthHandler(r.repository, r.config)
	// musicHandler'a config parametresi eklenmeli (eğer JWTSecret vs. kullanıyorsa)
	musicHandler := handlers.NewMusicHandler(r.repository, r.config)
	frontendHandler := handlers.NewFrontendHandler(r.repository, r.config, r.rabbitmqClient)

	// --- Custom 404 Handler ---
	r.engine.NoRoute(frontendHandler.NotFoundPage)

	// --- Frontend Routes (Page Loads) ---
	r.engine.GET("/", frontendHandler.HomePage)
	r.engine.GET("/login", frontendHandler.Login)
	r.engine.GET("/register", frontendHandler.Register)
	// AuthMiddleware JWTSecret almalı
	r.engine.GET("/library", middleware.AuthMiddleware(r.config.JWTSecret), frontendHandler.Library)
	r.engine.GET("/explore", frontendHandler.Explore)
	r.engine.GET("/musics/:id", musicHandler.GetMusicPage)

	// --- Partials Routes (for HTMX swaps) ---
	partials := r.engine.Group("/partials")
	{
		partials.GET("/play-button", func(c *gin.Context) {
			_, _, auth := middleware.GetUserInfoFromContext(c)
			// Varsayılan olarak auth durumunu gönderiyoruz, diğer gerekli veriler (örn: MusicType, ModelType)
			// ana sayfa handler'ından zaten geliyor ve JS ile yönetiliyor.
			c.HTML(http.StatusOK, "partials/play_button.html", gin.H{"auth": auth})
		})
		partials.GET("/edit-title-form", frontendHandler.GetEditTitleFormPartial)
		partials.GET("/title-text", frontendHandler.GetTitleTextPartial)
	}

	// --- API v1 Routes ---
	apiv1 := r.engine.Group("/api/v1")
	{
		// Auth API
		apiv1.POST("/register", authHandler.RegisterHandler)
		apiv1.POST("/login", authHandler.LoginHandler)
		apiv1.POST("/logout", authHandler.LogoutHandler)

		// Music Generation & Listing API (HTMX için)
		apiv1.POST("/generate-music", frontendHandler.GenerateMusicHandler)
		// AuthMiddleware JWTSecret almalı
		apiv1.GET("/music", middleware.AuthMiddleware(r.config.JWTSecret), frontendHandler.GetMusicsLibrary)
		apiv1.GET("/explore-music-data", frontendHandler.GetExploreMusicData) // Yeni eklendi

		// Music Management API
		// AuthMiddleware JWTSecret almalı
		apiv1.POST("/save-music", middleware.AuthMiddleware(r.config.JWTSecret), musicHandler.SaveGeneratedMusicHandler)
		apiv1.PUT("/music/:id/title", middleware.AuthMiddleware(r.config.JWTSecret), musicHandler.UpdateMusicTitleHandler)
	}

	// --- Swagger ---
	// Projenizde swagger kullanıyorsanız:
	// swaggerURL := ginSwagger.URL("/swagger/doc.json") // Swagger JSON dosyanızın yolu
	// r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerURL))
	// Eğer kullanmıyorsanız bu satırları kaldırabilirsiniz.
	// Şimdilik varsayılan olarak bırakıyorum:
	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
