# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o justachatapp .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/justachatapp .
# Copy templates and static files
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Expose port
EXPOSE 8080

# Run the application
CMD ["./justachatapp"]
