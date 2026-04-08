FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .
RUN go build -o /tmp/otp-service ./cmd/api

FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /tmp/otp-service /usr/local/bin/otp-service
COPY migrations ./migrations
COPY docs ./docs

EXPOSE 8080

CMD ["/usr/local/bin/otp-service"]
