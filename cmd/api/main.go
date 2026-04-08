package main

//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -d .,../../internal -o ../../docs --parseInternal

// @title OTP Service API
// @version 1.0
// @description OTP request and validation service.
// @BasePath /
// @schemes http

import (
	"context"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"otp-pair-code/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := bootstrap.Run(ctx); err != nil {
		stdlog.Printf("application error: %v", err)
		os.Exit(1)
	}
}
