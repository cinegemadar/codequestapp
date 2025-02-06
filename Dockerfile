# Stage 1: Build the application with Go 1.23 on Alpine.
FROM golang:1.23-alpine AS builder

# Install required packages for building.
RUN apk add --no-cache bash build-base sqlite

# Set the working directory.
WORKDIR /app

# Copy go.mod and go.sum, then download dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code.
COPY . .

# Build the Go application.
RUN go build -o code-quest-app

# Stage 2: Create the final image.
FROM alpine:latest

# Install SQLite and bash in the final image.
RUN apk add --no-cache sqlite bash

# Create a non-root user and switch to it.
RUN adduser -D appuser
USER appuser

# Set the working directory.
WORKDIR /app

# Copy the built binary and necessary files from the builder stage.
COPY --from=builder /app/code-quest-app .
COPY --from=builder /app/quest.txt .
COPY --from=builder /app/template.html .

# Expose port 8080.
EXPOSE 8080

# Run the application.
CMD ["./code-quest-app"]
