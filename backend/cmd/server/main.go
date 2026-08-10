package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"veurubro/backend/internal/app"
	"veurubro/backend/internal/battle"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/httpapi"
	"veurubro/backend/internal/storage"
	"veurubro/backend/internal/telemetry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://veurubro:veurubro_dev@localhost:5432/veurubro?sslmode=disable"
	}
	db, err := storage.Open(ctx, databaseURL)
	if err != nil {
		logger.Error("falha ao conectar ao banco", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.SyncCatalog(ctx); err != nil {
		logger.Error("falha ao sincronizar catálogo", "error", err)
		os.Exit(1)
	}
	shutdownTelemetry, err := telemetry.Setup(ctx, "veu-rubro-api", engine.RulesetVersion)
	if err != nil {
		logger.Error("falha ao iniciar telemetria", "error", err)
		os.Exit(1)
	}
	defer func() {
		telemetryCtx, telemetryCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer telemetryCancel()
		if err := shutdownTelemetry(telemetryCtx); err != nil {
			logger.Error("falha ao encerrar telemetria", "error", err)
		}
	}()

	service := app.New(db)
	battleManager := battle.NewManager(db)
	handler := otelhttp.NewHandler(httpapi.New(service, battleManager, logger, db.Ping), "http.server")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("backend iniciado", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("servidor interrompido", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown incompleto", "error", err)
	}
}
