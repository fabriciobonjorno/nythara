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
	"veurubro/backend/internal/config"
	"veurubro/backend/internal/engine"
	"veurubro/backend/internal/httpapi"
	"veurubro/backend/internal/mailer"
	"veurubro/backend/internal/storage"
	"veurubro/backend/internal/telemetry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	databaseURL, err := config.RequiredSecret("DATABASE_URL")
	if err != nil {
		logger.Error("configuração do banco ausente", "error", err)
		os.Exit(1)
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
	shutdownTelemetry, err := telemetry.Setup(ctx, "nythara-api", engine.CompetitiveRulesetVersion)
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
	if resendAPIKey := os.Getenv("RESEND_API_KEY"); resendAPIKey != "" {
		sender, err := mailer.NewResend(resendAPIKey, os.Getenv("RESEND_FROM_EMAIL"))
		if err != nil {
			logger.Error("configuração do Resend inválida", "error", err)
			os.Exit(1)
		}
		if err := service.ConfigurePasswordRecovery(sender, os.Getenv("PUBLIC_APP_URL")); err != nil {
			logger.Error("configuração da recuperação de senha inválida", "error", err)
			os.Exit(1)
		}
		logger.Info("recuperação de senha configurada")
	} else {
		logger.Warn("recuperação de senha desativada; RESEND_API_KEY ausente")
	}
	googleClientID, googleClientSecret := os.Getenv("GOOGLE_OAUTH_CLIENT_ID"), os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")
	if googleClientID != "" || googleClientSecret != "" {
		if err := service.ConfigureGoogleOAuth(googleClientID, googleClientSecret, os.Getenv("PUBLIC_APP_URL")); err != nil {
			logger.Error("configuração do Google OAuth inválida", "error", err)
			os.Exit(1)
		}
		logger.Info("Google OAuth configurado")
	} else {
		logger.Warn("Google OAuth desativado; credenciais ausentes")
	}
	if os.Getenv("RESEND_WEBHOOK_SECRET") == "" {
		logger.Warn("webhook do Resend desativado; RESEND_WEBHOOK_SECRET ausente")
	} else {
		logger.Info("webhook do Resend configurado")
	}
	battleManager := battle.NewManager(db)

	// Fase 7: versões publicadas ficam executáveis (replays históricos) e o
	// matchmaking segue o ponteiro ativo do banco; ativações via admin
	// repontam o manager em tempo real.
	if registered, err := service.RegisterStoredRulesets(ctx); err != nil {
		logger.Error("falha ao registrar rulesets persistidos", "error", err)
		os.Exit(1)
	} else if len(registered) > 0 {
		logger.Info("rulesets registrados", "versions", registered)
	}
	if infos, err := db.ListRulesets(ctx); err == nil {
		for _, info := range infos {
			if info.Active {
				battleManager.SetActiveRuleset(info.Version)
			}
		}
	}
	service.SetRulesetActivationHook(battleManager.SetActiveRuleset)
	battleManager.SetProgressRecorder(func(ctx context.Context, finished battle.FinishedMatch, events []engine.Event) {
		players := [2]app.FinishedPlayer{}
		for slot := 0; slot < 2; slot++ {
			players[slot] = app.FinishedPlayer{UserID: finished.Players[slot].UserID,
				ChampionID: finished.Players[slot].ChampionID}
		}
		if err := service.RecordFinishedMatch(ctx, finished.MatchID, finished.Mode,
			finished.RulesetVersion, players, finished.Winner, events); err != nil {
			logger.Error("progressão da partida falhou", "match", finished.MatchID, "error", err)
		}
	})

	handler, err := httpapi.NewWithOptions(service, battleManager, logger, db.Ping, httpapi.Options{
		TrustedProxyCIDRs:   os.Getenv("TRUSTED_PROXY_CIDRS"),
		ResendWebhookSecret: os.Getenv("RESEND_WEBHOOK_SECRET"),
	})
	if err != nil {
		logger.Error("configuração HTTP inválida", "error", err)
		os.Exit(1)
	}
	handler = otelhttp.NewHandler(handler, "http.server")
	port := os.Getenv("PORT")
	if port == "" {
		port = "18080"
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
