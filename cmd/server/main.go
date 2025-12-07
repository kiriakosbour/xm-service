package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xm-company-service/internal/config"
	"xm-company-service/internal/core"
	"xm-company-service/internal/handler"
	"xm-company-service/internal/middleware"
	"xm-company-service/internal/platform/kafka"
	"xm-company-service/internal/platform/postgres"
	"xm-company-service/internal/service"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg := config.Load()
	log.Printf("Starting server with config: port=%s, db=%s", cfg.Server.Port, maskDSN(cfg.Database.URL))

	// Initialize database
	db, err := initDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	repo := postgres.NewRepository(db)
	if err := repo.Migrate(context.Background()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Initialize Kafka producer
	var producer core.EventProducer
	if cfg.Kafka.Enabled {
		producer = kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.Enabled)
	} else {
		producer = kafka.NewNoOpProducer()
	}
	defer producer.Close()

	// Initialize service and handlers
	companySvc := service.NewCompanyService(repo, producer)
	companyHandler := handler.NewHandler(companySvc)
	healthHandler := handler.NewHealthHandler(db)

	// Setup router
	r := setupRouter(companyHandler, healthHandler)

	// Create server
	srv := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}

func initDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func setupRouter(h *handler.Handler, health *handler.HealthHandler) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// Health check endpoints (no auth required)
	r.Get("/health/live", health.Live)
	r.Get("/health/ready", health.Ready)

	// Public routes
	r.Get("/companies/{id}", h.Get)

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth)
		r.Post("/companies", h.Create)
		r.Patch("/companies/{id}", h.Patch)
		r.Delete("/companies/{id}", h.Delete)
	})

	return r
}

// maskDSN hides password in DSN for logging
func maskDSN(dsn string) string {
	// Simple masking - in production use a proper DSN parser
	return "****"
}
