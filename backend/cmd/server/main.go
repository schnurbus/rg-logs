package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"rg-logs/internal/api"
	"rg-logs/internal/db"
	"rg-logs/internal/ingest"
)

func main() {
	databaseURL := env("DATABASE_URL", "postgres://rglogs:rglogs@localhost:5432/rglogs?sslmode=disable")
	uploadDir := env("UPLOAD_DIR", "./uploads")
	addr := env("HTTP_ADDR", ":3000")

	if !filepath.IsAbs(uploadDir) {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("getcwd: %v", err)
		}
		if uploadDir == "./uploads" || uploadDir == "uploads" {
			if filepath.Base(cwd) == "backend" {
				uploadDir = filepath.Clean(filepath.Join(cwd, "..", "uploads"))
			} else {
				uploadDir = filepath.Join(cwd, "uploads")
			}
		} else {
			uploadDir = filepath.Join(cwd, uploadDir)
		}
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		log.Fatalf("upload dir: %v", err)
	}

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
	worker := ingest.NewWorker(store, uploadDir, 16)
	worker.Start(2)

	h := &api.Handler{Store: store, Worker: worker, UploadDir: uploadDir}
	app := api.NewRouter(h)

	go func() {
		log.Printf("listening on %s (UPLOAD_DIR=%s)", addr, uploadDir)
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
