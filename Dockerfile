# Stage 1: Build the Go binary
FROM golang:alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install git (required for fetching some Go dependencies) and ca-certificates
RUN apk update && apk add --no-cache git ca-certificates tzdata

# Copy go.mod and go.sum files first to leverage Docker cache for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application as a static binary
# CGO_ENABLED=0 ensures it's statically linked, reducing dependencies on the target OS
# -ldflags="-w -s" strips debugging information, further shrinking the binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o api ./cmd/api

# Stage 2: Create the minimal final image
# Scratch is a special empty image in Docker. It contains literally nothing, not even a shell!
FROM scratch

# Copy CA certificates for HTTPS requests (e.g., if the API talks to external APIs)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data (useful if the app does timezone formatting)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the compiled binary from the builder stage
COPY --from=builder /app/api /api

# Copy the frontend files so the API can serve them
COPY --from=builder /app/frontend /frontend

# Expose the port the API runs on
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/api"]
