package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/zixflow/messaging-simulator/internal/admin"
	"github.com/zixflow/messaging-simulator/internal/config"
	"github.com/zixflow/messaging-simulator/internal/core"
	"github.com/zixflow/messaging-simulator/internal/migrate"
	"github.com/zixflow/messaging-simulator/internal/rcs"
	"github.com/zixflow/messaging-simulator/internal/store"
	"github.com/zixflow/messaging-simulator/internal/whatsapp"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	if err := migrate.Up(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	db, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	svc := core.NewServices(db, cfg.MediaStoragePath)
	if err := svc.EnsureMediaDir(); err != nil {
		log.Fatalf("media dir: %v", err)
	}
	defer svc.Dispatcher.Stop()

	rcsHandler := rcs.NewHandler(svc)
	waHandler := whatsapp.NewHandler(svc)

	adminHandler, err := admin.NewHandler(svc, cfg)
	if err != nil {
		log.Fatalf("admin handler: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	origins := append(cfg.CORSOrigins, "http://localhost:3000", "http://localhost:5173")
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Admin UI API — all dashboard routes under /api for same-domain nginx routing
	r.Mount("/api", adminHandler.Routes())

	// Vendor APIs — production paths on separate domain (no /api prefix)
	r.Mount("/", rcsHandler.Routes())

	// Meta Graph API — production paths (graph.facebook.com/v19.0/...)
	r.Mount("/", waHandler.Routes())

	// Dev aliases for single-host domain swap (backward compatible)
	r.Mount("/rcs", rcsHandler.Routes())
	r.Mount("/whatsapp", waHandler.Routes())

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	fmt.Println("shutdown complete")
}
