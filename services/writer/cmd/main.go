package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/cascade-analytics/cascade/services/writer/internal/consumer"
	"github.com/cascade-analytics/cascade/services/writer/internal/writer"
	"github.com/redis/go-redis/v9"
)

func main() {
	redisURL := getenv("REDIS_URL", "redis://localhost:6379")
	stream := getenv("REDIS_STREAM", "ingest:stream")
	group := getenv("REDIS_CONSUMER_GROUP", "writer-group")
	chDSN := getenv("CLICKHOUSE_DSN", "clickhouse://cascade:cascade@localhost:9000/cascade")
	batchSize := getenvInt("BATCH_SIZE", 1000)
	flushMS := getenvInt("FLUSH_INTERVAL_MS", 500)

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		fatalf("parse Redis URL: %v", err)
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	chWriter, err := writer.NewClickHouseWriter(chDSN)
	if err != nil {
		fatalf("connect ClickHouse: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Verify connectivity before entering the consumer loop.
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		fatalf("Redis ping: %v", err)
	}
	if err := chWriter.Ping(pingCtx); err != nil {
		fatalf("ClickHouse ping: %v", err)
	}

	c := consumer.New(
		rdb, chWriter,
		stream, group, "writer-1",
		batchSize,
		time.Duration(flushMS)*time.Millisecond,
	)

	if err := c.EnsureGroup(ctx); err != nil {
		fatalf("ensure consumer group: %v", err)
	}

	fmt.Println("writer: consuming from", stream)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		cancel()
	}()

	if err := c.Run(ctx); err != nil {
		fatalf("consumer: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}
