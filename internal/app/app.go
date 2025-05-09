package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/morgarakt/aurify/internal/config"
	"github.com/morgarakt/aurify/internal/repository"
	"github.com/morgarakt/aurify/internal/router"
	"github.com/morgarakt/aurify/internal/services"
	"github.com/morgarakt/aurify/internal/utils"
	"gorm.io/gorm"
)

type Application struct {
	cfg            *config.Config
	db             *gorm.DB
	repository     *repository.Repository
	rabbitmqClient *services.RabbitMQClient
	router         *router.Router
	server         *http.Server
}

func NewApplication() (*Application, error) {
	app := &Application{}
	var err error

	if err = app.loadConfig(); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	if err = app.initializeDatabase(); err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if err = app.initializeRabbitMQ(); err != nil {
		if app.db != nil {
			_ = utils.CloseDB(app.db)
		}
		return nil, fmt.Errorf("rabbitmq error: %w", err)
	}

	app.initializeRepositories()
	app.initializeRouter()

	return app, nil
}

func (a *Application) loadConfig() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.RabbitMQURL == "" {
		return fmt.Errorf("RABBITMQ_URL is not set in config")
	}

	a.cfg = cfg
	log.Println("Configuration loaded successfully.")
	return nil
}

func (a *Application) initializeDatabase() error {
	db, err := utils.ConnectDB(a.cfg)
	if err != nil {
		return err
	}

	log.Println("Attempting database migration...")
	if err := utils.MigrateDB(db); err != nil {
		_ = utils.CloseDB(db)
		return fmt.Errorf("migration failed: %w", err)
	}

	a.db = db
	log.Println("Database initialized and migrated successfully.")
	return nil
}

func (a *Application) initializeRabbitMQ() error {
	log.Println("Initializing RabbitMQ client...")
	rmqClient, err := services.NewRabbitMQClient(a.cfg.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("failed to initialize RabbitMQ client: %w", err)
	}
	a.rabbitmqClient = rmqClient
	log.Println("RabbitMQ client initialized successfully.")
	return nil
}

func (a *Application) initializeRepositories() {
	a.repository = repository.NewRepository(a.db)
	log.Println("Repositories initialized.")
}

func (a *Application) initializeRouter() {
	a.router = router.NewRouter(a.repository, a.cfg, a.rabbitmqClient)
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", a.cfg.Port),
		Handler: a.router.Engine(),
	}
	log.Println("Router initialized.")
}

func (a *Application) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		log.Printf("Server starting on port %s", a.cfg.Port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown failed: %v", err)
	} else {
		log.Println("Server gracefully stopped.")
	}

	if a.rabbitmqClient != nil {
		log.Println("Closing RabbitMQ connection...")
		a.rabbitmqClient.Close()
		log.Println("RabbitMQ connection closed.")
	} else {
		log.Println("RabbitMQ client was not initialized, skipping close.")
	}

	if a.db != nil {
		log.Println("Closing database connection...")
		if err := utils.CloseDB(a.db); err != nil {
			log.Printf("Error closing database connection: %v", err)
			return fmt.Errorf("error closing database: %w", err)
		}
		log.Println("Database connection closed.")
	} else {
		log.Println("Database connection was not initialized, skipping close.")
	}

	log.Println("Server exited properly")
	return nil
}
