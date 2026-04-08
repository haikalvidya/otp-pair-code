# OTP Service

Go OTP service built with `chi`, `pgxpool`, raw SQL, `zerolog`, `goose`, and `swaggo` using a lightweight clean and hexagonal-inspired structure.

## Requirements

- Go 1.25+
- Docker and Docker Compose
- GNU Make

## Environment Files

- `.env.example`: template for local development
- `.env`: local app config used by `make dev` and `make run`
- `.env.docker`: config used by `docker compose`

## Run Locally With Air

1. Create local env:

```bash
cp .env.example .env
```

2. Start the app with Air. This will also start only the `postgres` container:

```bash
make dev
```

This flow:

- Starts only the `postgres` container with `docker compose`
- Runs the Go API locally with `air` hot reload
- Loads configuration from `.env`

Useful commands:

```bash
make db-up
make db-down
make db-logs
make run
make test
make swagger
```

To stop the local database container:

```bash
make db-down
```

If you want a global `air` binary instead of the `go run` fallback:

```bash
make air-install
```

## Run Everything With Docker Compose

Run the API and Postgres in containers:

```bash
docker compose up --build
```

This flow uses `.env.docker`.

To stop all containers:

```bash
docker compose down
```

Service endpoints:

- `POST http://localhost:8080/otp/request`
- `POST http://localhost:8080/otp/validate`
- `GET http://localhost:8080/healthz`
- `GET http://localhost:8080/swagger/index.html`

## Regenerate Swagger Docs

```bash
go generate ./cmd/api
```

## Run Tests

```bash
go test ./...
```

## Example Requests

Request OTP:

```bash
curl -X POST http://localhost:8080/otp/request \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"Robert"}'
```

Example response:

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
