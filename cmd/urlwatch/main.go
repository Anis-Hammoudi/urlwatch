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

	"urlwatch/internal/api"
	"urlwatch/internal/checker"
	"urlwatch/internal/store"
)

func main() {
	// 1. Initialize Structured Logging
	logLevel := slog.LevelInfo
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		var l slog.Level
		if err := l.UnmarshalText([]byte(levelStr)); err == nil {
			logLevel = l
		}
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// 2. Initialize Core Dependencies
	memStore := store.NewMemoryStore()
	httpChecker := checker.NewHTTPChecker(nil) // Uses http.DefaultClient internally

	// 3. Assemble the API Router
	router := api.NewServer(logger, memStore, httpChecker)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Configure the HTTP Server with best-practice timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 4. Graceful Shutdown Setup (Bonus Requirement)
	quit := make(chan os.Signal, 1)
	// Listen for CTRL+C (SIGINT) and Docker/K8s stop signals (SIGTERM)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start the server in a separate goroutine so it doesn't block
	go func() {
		logger.Info("Starting server", slog.String("port", port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed to start", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Block the main thread until an OS signal is received in the quit channel
	<-quit
	logger.Info("Shutting down server gracefully...")

	// Create a context with a 10-second timeout to allow ongoing batches to finish
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tell the server to stop accepting new requests and finish the current ones
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("Server exited properly")
}
