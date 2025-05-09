FROM golang:alpine AS builder

WORKDIR /app

# Copy Go module files first to leverage Docker cache
# If go.mod/go.sum don't change, Docker reuses the dependency layer
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application as a static binary (recommended for alpine runtime)
# -ldflags="-w -s" reduces binary size (strips debug info)
# CGO_ENABLED=0 ensures static linking
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/main ./cmd/main.go


# --- Runtime Stage ---
# Use a minimal base image like Alpine for the final image
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy ONLY the compiled binary from the build stage
COPY --from=builder /app/main /app/main
COPY --from=builder /app/api/openapi.yaml /app/api/openapi.yaml

# (Optional but Recommended) Create a non-root user to run the application
# RUN addgroup -S appgroup && adduser -S appuser -G appgroup
# USER appuser
RUN chmod +x /app/main

# Expose the port the application listens on inside the container
EXPOSE 8000

# Command to run the executable when the container starts
ENTRYPOINT ["/app/main"]
