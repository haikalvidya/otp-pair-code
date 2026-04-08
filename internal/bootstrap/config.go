package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type Config struct {
	Port                 string
	DatabaseURL          string
	LogLevel             zerolog.Level
	RequestTimeout       time.Duration
	ShutdownTimeout      time.Duration
	OTPAllowReissue      bool
	OTPMaxFailedAttempts int
}

func LoadConfig() (Config, error) {
	levelText := getEnv("LOG_LEVEL", "info")
	level, err := zerolog.ParseLevel(strings.ToLower(levelText))
	if err != nil {
		return Config{}, fmt.Errorf("parse LOG_LEVEL: %w", err)
	}

	requestTimeout, err := getDuration("REQUEST_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("parse REQUEST_TIMEOUT: %w", err)
	}

	shutdownTimeout, err := getDuration("SHUTDOWN_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("parse SHUTDOWN_TIMEOUT: %w", err)
	}

	maxAttempts, err := getInt("OTP_MAX_FAILED_ATTEMPTS", 5)
	if err != nil {
		return Config{}, fmt.Errorf("parse OTP_MAX_FAILED_ATTEMPTS: %w", err)
	}

	allowReissue, err := getBool("OTP_ALLOW_REISSUE", false)
	if err != nil {
		return Config{}, fmt.Errorf("parse OTP_ALLOW_REISSUE: %w", err)
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          databaseURL,
		LogLevel:             level,
		RequestTimeout:       requestTimeout,
		ShutdownTimeout:      shutdownTimeout,
		OTPAllowReissue:      allowReissue,
		OTPMaxFailedAttempts: maxAttempts,
	}, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	return time.ParseDuration(value)
}

func getInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func getBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseBool(value)
}
