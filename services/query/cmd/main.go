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

	"github.com/cascade-analytics/cascade/services/query/internal/handler"
	"github.com/cascade-analytics/cascade/services/query/internal/middleware"
	"github.com/cascade-analytics/cascade/services/query/internal/store"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

func main() {
	chDSN := getenv("CLICKHOUSE_DSN", "clickhouse://cascade:cascade@localhost:9000/cascade")
	redisURL := getenv("REDIS_URL", "redis://localhost:6379")
	port := getenv("PORT", "8081")

	chStore, err := store.NewClickHouseStore(chDSN)
	if err != nil {
		fatalf("connect ClickHouse: %v", err)
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		fatalf("parse Redis URL: %v", err)
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := chStore.Ping(pingCtx); err != nil {
		fatalf("ClickHouse ping: %v", err)
	}
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		fatalf("Redis ping: %v", err)
	}

	queryH := handler.NewQueryHandler(chStore)

	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		chStatus := "ok"
		if err := chStore.Ping(r.Context()); err != nil {
			chStatus = "error"
		}
		redisStatus := "ok"
		if err := rdb.Ping(r.Context()).Err(); err != nil {
			redisStatus = "error"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":     "ok",
			"clickhouse": chStatus,
			"redis":      redisStatus,
		})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.RequireTenantID)
		r.Get("/count", queryH.Count)
		r.Get("/timeseries", queryH.TimeSeries)
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Printf("query service listening on :%s\n", port)
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
