# ---------- 1. Build stage ----------
FROM golang:1.24.1 AS builder

WORKDIR /app


# Install build deps for confluent-kafka-go
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        pkg-config \
        librdkafka-dev \
        gcc \
        g++ \
        make && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./

RUN go mod download

# Copy full repo (producer, shared, etc.)
COPY . .


# Build producer binary
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /producer ./producer/cmd

# ---------- 2. Runtime stage ----------
FROM debian:stable-slim

WORKDIR /app

# runtime deps
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        librdkafka1 && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /producer /app/producer

CMD ["/app/producer"]
