package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"rg-logs/internal/api"
	"rg-logs/internal/auth"
	"rg-logs/internal/db"
	"rg-logs/internal/ingest"
	"rg-logs/internal/storage"
)

func main() {
	loadDotEnv("../.env", ".env")

	databaseURL := mustEnv("DATABASE_URL")
	supabaseURL := mustEnv("SUPABASE_URL")
	anonKey := mustEnv("SUPABASE_ANON_KEY")
	serviceRoleKey := mustEnv("SUPABASE_SERVICE_ROLE_KEY")
	addr := env("HTTP_ADDR", ":3000")

	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migrations applied")

	store := db.NewStore(pool)
	authClient := auth.NewClient(supabaseURL, anonKey)
	storageClient := storage.NewClient(supabaseURL, serviceRoleKey, storage.DefaultBucket)
	worker := ingest.NewWorker(store, storageClient, 16)
	worker.Start(2)

	h := &api.Handler{
		Store:   store,
		Worker:  worker,
		Auth:    authClient,
		Storage: storageClient,
	}
	app := api.NewRouter(h)

	go func() {
		log.Printf("listening on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(shutdownCtx)
	worker.Stop()
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env %s (set it or add it to .env / ../.env)", key)
	}
	return v
}

// loadDotEnv loads KEY=VALUE pairs from the given files if present.
// Existing process env vars are never overwritten.
func loadDotEnv(paths ...string) {
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, val, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			if len(val) >= 2 {
				if (val[0] == '"' && val[len(val)-1] == '"') ||
					(val[0] == '\'' && val[len(val)-1] == '\'') {
					val = val[1 : len(val)-1]
				}
			}
			if key == "" {
				continue
			}
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
		_ = f.Close()
	}
}
