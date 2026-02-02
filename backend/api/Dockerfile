# ---------- 1. Build stage ----------
FROM golang:1.24.1 AS builder

WORKDIR /app

# Copy only go.mod/go.sum first for better caching
COPY go.mod go.sum ./

RUN ls -R . && cat go.mod && echo "--- end go.mod ---"

RUN go mod download

# Now copy the rest of the repo (shared, iot-service, etc.)
COPY . .

# Build the iot-service binary
# NOTE: static build, CGO disabled because iot-service doesn't need librdkafka
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /api ./api/cmd

# ---------- 2. Runtime stage ----------
FROM alpine:3.20

WORKDIR /app

# For HTTPS to Elasticsearch, Postgres logs, etc.
RUN apk add --no-cache ca-certificates

COPY --from=builder /api /app/api

EXPOSE 8080

CMD ["/app/api"]
