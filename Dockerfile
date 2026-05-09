FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

# Final Stage
FROM alpine:3.18

WORKDIR /app

# Copy built binary
COPY --from=builder /app/main .

# Expose port
EXPOSE 3000

# Run the binary
CMD ["./main"]
