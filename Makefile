AIR_VERSION ?= v1.61.7
AIR ?= $(shell command -v air 2>/dev/null)
AIR_CMD := $(if $(AIR),$(AIR),go run github.com/air-verse/air@$(AIR_VERSION))
COMPOSE ?= docker compose

ifneq (,$(wildcard .env))
include .env
endif

export PORT DATABASE_URL LOG_LEVEL REQUEST_TIMEOUT SHUTDOWN_TIMEOUT OTP_ALLOW_REISSUE OTP_MAX_FAILED_ATTEMPTS

.PHONY: air-install db-up db-down db-logs dev run test swagger

air-install:
	go install github.com/air-verse/air@$(AIR_VERSION)

db-up:
	$(COMPOSE) up -d postgres

db-down:
	$(COMPOSE) stop postgres

db-logs:
	$(COMPOSE) logs -f postgres

dev: db-up
	$(AIR_CMD) -c .air.toml

run: db-up
	go run ./cmd/api

test:
	go test ./...

swagger:
	go generate ./cmd/api
