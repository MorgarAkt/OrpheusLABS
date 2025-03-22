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
	router "github.com/morgarakt/aurify/internal/router"
	"github.com/morgarakt/aurify/internal/utils"
	"gorm.io/gorm"
)

type Application struct {
	cfg        *config.Config
	db         *gorm.DB
	repository *repository.Repository
	router     *router.Router
	server     *http.Server
}

func NewApplication() (*Application, error) {
	app := &Application{}

	if err := app.loadConfig(); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	if err := app.initializeDatabase(); err != nil {
		return nil, fmt.Errorf("database error: %w", err)
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
	a.cfg = cfg
	return nil
}

func (a *Application) initializeDatabase() error {
	db, err := utils.ConnectDB(a.cfg)
	if err != nil {
		return err
	}

	if err := utils.MigrateDB(db); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	a.db = db
	return nil
}

func (a *Application) initializeRepositories() {
	a.repository = repository.NewRepository(a.db)
}

func (a *Application) initializeRouter() {
	a.router = router.NewRouter(a.repository, a.cfg)
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", a.cfg.Port),
		Handler: a.router.Engine(),
	}
}

func (a *Application) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		log.Printf("Server running on port %s", a.cfg.Port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("forced shutdown: %w", err)
	}

	if err := utils.CloseDB(a.db); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}

	log.Println("Server exited properly")
	return nil
}
