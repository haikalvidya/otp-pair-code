# OTP Service

Go OTP service built with `chi`, `pgxpool`, raw SQL, `zerolog`, `goose`, and `swaggo` using a lightweight clean and hexagonal-inspired structure.

## Overview

This service exposes two main operations:

- `POST /otp/request` to create an OTP for a user.
- `POST /otp/validate` to validate the latest active OTP for a user.

It is designed to keep OTP state simple and explicit:

- Only one active OTP (`status = created`) is allowed per user.
- OTP validity window is fixed at 2 minutes.
- Successful validation consumes the OTP.
- Repeated wrong validation attempts can block the OTP based on config.

Detailed docs:

- [Design notes](docs/design.md)
- [Double-hit simulation](docs/simulation.md)

## Requirements

- Go 1.25+
- Docker and Docker Compose
- GNU Make

## Environment Files

- `.env.example`: template for local development.
- `.env`: local app config used by `make dev` and `make run`.
- `.env.docker`: config used by `docker compose`.

## Configuration

Main environment variables:

- `PORT`: HTTP port. Default `8080`.
- `DATABASE_URL`: required Postgres connection string.
- `LOG_LEVEL`: logger level such as `info` or `debug`.
- `REQUEST_TIMEOUT`: per-request timeout. Default `5s`.
- `SHUTDOWN_TIMEOUT`: graceful shutdown timeout. Default `5s`.
- `OTP_ALLOW_REISSUE`: allow replacing an active OTP. Default `false`.
- `OTP_MAX_FAILED_ATTEMPTS`: maximum wrong validations before blocking. Default `5`.

## Quick Start

1. Create local env file:

```bash
cp .env.example .env
```

2. Start the database only:

```bash
make db-up
```

3. Run the API locally:

```bash
make run
```

For hot reload during development:

```bash
make dev
```

What `make dev` does:

- Starts only the `postgres` container with `docker compose`.
- Runs the Go API locally with `air`.
- Loads configuration from `.env`.

Useful commands:

```bash
make air-install
make db-up
make db-down
make db-logs
make dev
make run
make test
make swagger
```

Stop the local database container:

```bash
make db-down
```

## Run With Docker Compose

Run the API and Postgres in containers:

```bash
docker compose up --build
```

This mode uses `.env.docker`.

Stop all containers:

```bash
docker compose down
```

## Endpoints

- `POST http://localhost:8080/otp/request`
- `POST http://localhost:8080/otp/validate`
- `GET http://localhost:8080/healthz`
- `GET http://localhost:8080/swagger/index.html`

## Example Requests

Request OTP:

```bash
curl -X POST http://localhost:8080/otp/request \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"Robert"}'
```

Example success response:

```json
{
  "data": {
    "user_id": "Robert",
    "otp": "12345"
  },
  "meta": {
    "request_id": "req-123"
  }
}
```

Validate OTP:

```bash
curl -X POST http://localhost:8080/otp/validate \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"Robert","otp":"12345"}'
```

Example error response:

```json
{
  "error": {
    "code": "otp_invalid",
    "message": "OTP is invalid"
  },
  "meta": {
    "request_id": "req-123"
  }
}
```

## Swagger Docs

Regenerate Swagger docs:

```bash
go generate ./cmd/api
```

## Tests

Run all tests:

```bash
go test ./...
```
