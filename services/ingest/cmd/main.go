package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cascade-analytics/cascade/services/ingest/internal/handler"
	"github.com/cascade-analytics/cascade/services/ingest/internal/middleware"
	"github.com/cascade-analytics/cascade/services/ingest/internal/queue"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

func main() {
	redisURL := getenv("REDIS_URL", "redis://localhost:6379")
	stream := getenv("REDIS_STREAM", "ingest:stream")
	port := getenv("PORT", "8080")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		fatalf("parse Redis URL: %v", err)
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		fatalf("connect to Redis: %v", err)
	}

	enqueuer := queue.NewEnqueuer(rdb, stream)
	captureH := handler.NewCaptureHandler(enqueuer)

	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		redisStatus := "ok"
		if err := enqueuer.Ping(r.Context()); err != nil {
			redisStatus = "error"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"redis":  redisStatus,
		})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.RequireTenantID)
		r.Post("/capture", captureH.Capture)
		r.Post("/batch", captureH.Batch)
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Printf("ingest service listening on :%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		fatalf("shutdown: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}
