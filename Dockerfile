# Build stage


FROM golang:1.23 AS builder   

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o news-api .


FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/news-api .

EXPOSE 8080
CMD ["./news-api"]
