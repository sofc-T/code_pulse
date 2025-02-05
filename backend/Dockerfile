# Build stage
FROM golang:1.21 AS builder
WORKDIR /app

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy source code
COPY . .

# Build the binary for Linux with CGO disabled (static binary)
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/main.go

# Deployment stage
FROM alpine:latest
WORKDIR /root/

# Install libc compatibility (if needed)
RUN apk add --no-cache libc6-compat

# Copy the statically linked binary
COPY --from=builder /app/app .

# Ensure the binary is executable
RUN chmod +x /root/app

# Copy environment variables
COPY .env .env

# Expose port and set entrypoint
EXPOSE 8080
ENTRYPOINT ["/root/app"]
