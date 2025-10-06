# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# First copy only the files needed for dependencies
COPY go.mod go.sum ./

# Download dependencies and verify checksums
RUN go mod verify && go mod download

# Copy the rest of the source code
COPY . .

WORKDIR /app/cmd/bulwarkauth
RUN CGO_ENABLED=0 GOOS=linux go build -v -ldflags="-s -w" -o main .

# Final stage
FROM alpine:3.21.3

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/cmd/bulwarkauth/main .
COPY --from=builder /app/verification.html .
COPY --from=builder /app/magic.html .
COPY --from=builder /app/forgot.html .

# Default port and run mode
ENV PORT=8080
EXPOSE $PORT

# Health check (adjust endpoint as needed)
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/health || exit 1

CMD ["./main"]