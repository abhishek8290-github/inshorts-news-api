# Build stage
FROM golang:1.23 AS builder

WORKDIR /app

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN go build -o news-api .

# Final runtime stage
FROM debian:bookworm-slim

# Update package list and install CA certificates
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy built binary and env file
COPY --from=builder /app/news-api .
COPY .env .

EXPOSE 8080

CMD ["./news-api"]
