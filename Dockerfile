# --- Builder Stage ---
FROM golang:1.24.2-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build app
RUN go build -o realtime-ranking cmd/main.go

# --- Final Stage ---
FROM alpine:3.19

WORKDIR /app

# Cài certs để Swagger hoặc HTTP client không lỗi HTTPS
RUN apk add --no-cache ca-certificates

# Copy binary + swagger docs
COPY --from=builder /app/realtime-ranking .
COPY --from=builder /app/docs ./docs

EXPOSE 8080

CMD ["./realtime-ranking"]
