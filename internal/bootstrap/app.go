package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"

	httpadapter "otp-pair-code-interview/internal/adapters/http"
	"otp-pair-code-interview/internal/adapters/persistence/postgres"
	otpapp "otp-pair-code-interview/internal/application/otp"
)

func Run(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	logger := NewLogger(cfg)
	logger.Info().Str("event", "startup_started").Msg("application starting")

	pool, err := openPoolWithRetry(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer pool.Close()

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	if err := runMigrations(ctx, sqlDB); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	repo := postgres.NewOTPRepository(pool)
	service := otpapp.NewService(
		repo,
		systemClock{},
		secureOTPGenerator{},
		otpapp.Config{
			AllowReissue:      cfg.OTPAllowReissue,
			MaxFailedAttempts: cfg.OTPMaxFailedAttempts,
		},
	)
	handler := httpadapter.NewHandler(service, logger)
	router := httpadapter.NewRouter(handler, logger, cfg.RequestTimeout)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info().Str("event", "server_started").Str("addr", server.Addr).Msg("server listening")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-ctx.Done():
	}

	logger.Info().Str("event", "shutdown_started").Msg("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	logger.Info().Str("event", "shutdown_complete").Msg("shutdown complete")
	return nil
}

func openPoolWithRetry(ctx context.Context, cfg Config, logger zerolog.Logger) (*pgxpool.Pool, error) {
	for attempt := 1; ; attempt++ {
		pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			pingErr := pool.Ping(pingCtx)
			cancel()
			if pingErr == nil {
				return pool, nil
			}
			pool.Close()
			err = pingErr
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
			logger.Warn().Int("attempt", attempt).Err(err).Msg("database not ready, retrying")
		}
	}
}

func runMigrations(ctx context.Context, db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, db, "migrations")
}
