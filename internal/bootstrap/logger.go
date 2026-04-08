package bootstrap

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func NewLogger(cfg Config) zerolog.Logger {
	zerolog.SetGlobalLevel(cfg.LogLevel)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	return zerolog.New(os.Stdout).With().Timestamp().Str("service", "otp-service").Logger()
}
