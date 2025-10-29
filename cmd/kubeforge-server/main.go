package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"kubeforge/internal/api"
	"kubeforge/internal/config"
	"kubeforge/internal/db"
)

func main() {
	log.Println("Starting KubeForge server...")

	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := db.Init(db.Config{
		Driver: cfg.Database.Driver,
		DSN:    cfg.Database.DSN,
	}); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(api.Logger)
	router.Use(api.Recovery)
	router.Use(api.CORS)

	// Health check endpoint
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		api.WriteSuccess(w, map[string]string{
			"status":  "ok",
			"version": "1.0.0",
		})
	}).Methods("GET")

	// API routes
	clusterHandler := api.NewClusterHandler()
	clusterHandler.RegisterRoutes(router)

	// Create HTTP server
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
