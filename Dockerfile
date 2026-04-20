# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sysguard ./cmd/sysguard

# Final stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates procps

RUN addgroup -S sysguard && adduser -S -G sysguard sysguard

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/sysguard .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/docs ./docs

RUN mkdir -p /app/logs /app/data && chown -R sysguard:sysguard /app

USER sysguard

# Run the application
CMD ["./sysguard", "-config", "./configs/config.yaml"]
