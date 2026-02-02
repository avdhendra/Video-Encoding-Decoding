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

# Copy module metadata first
COPY go.mod go.sum ./
RUN go mod download

# Copy full repo (consumer, shared, etc.)
COPY . .

# Build consumer binary
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /consumer ./consumer/cmd

# ---------- 2. Runtime stage ----------
FROM debian:stable-slim

WORKDIR /app

# Install ffmpeg + certificates
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates ffmpeg \
  && rm -rf /var/lib/apt/lists/*
# runtime: librdkafka shared lib + certs
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        librdkafka1 && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /consumer /app/consumer

CMD ["/app/consumer"]
