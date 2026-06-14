package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m42-labs/saiph-worker/internal/config"
	workerhttp "github.com/m42-labs/saiph-worker/internal/http"
	"github.com/m42-labs/saiph-worker/internal/worker"
)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "saiph-worker")

	runner, err := worker.New(cfg, log)
	if err != nil {
		log.Error("worker_init_failed", "error", err.Error())
		os.Exit(1)
	}
	defer runner.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           workerhttp.Router("saiph-worker"),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("http_started", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http_failed", "error", err.Error())
			os.Exit(1)
		}
	}()

	go runner.Run(ctx)
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
