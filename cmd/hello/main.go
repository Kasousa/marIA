// Command hello is the Phase 1 service used to prove the pipeline.
//
// Run with no arguments to serve HTTP. Run "hello healthcheck" to probe the
// local /healthz endpoint; the distroless image has no curl, so the ECS
// container health check calls the binary itself.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kasousa/maria/internal/hello"
	"github.com/Kasousa/maria/pkg/logging"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := healthcheck(hello.Port()); err != nil {
			os.Exit(1)
		}
		return
	}

	cfg, err := hello.Load()
	if err != nil {
		logging.New(os.Stderr, "hello", slog.LevelInfo).Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	logger := logging.New(os.Stdout, "hello", cfg.LogLevel).With("env", cfg.Env, "version", cfg.Version)
	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

func run(cfg hello.Config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           hello.NewRouter(cfg, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "port", cfg.Port)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	// ECS sends SIGTERM on deploys and Spot interruptions: finish in-flight requests.
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func healthcheck(port string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+hello.HealthPath, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
