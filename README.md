# OTP Service

Go OTP service built with `chi`, `pgxpool`, raw SQL, `zerolog`, `goose`, and `swaggo` using a lightweight clean and hexagonal-inspired structure.

## Requirements

- Go 1.25+
- Docker and Docker Compose

## Run Locally With Docker

```bash
docker compose up --build
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

Validate OTP:

```bash
curl -X POST http://localhost:8080/otp/validate \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"Robert","otp":"12345"}'
```
