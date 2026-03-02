# ---------- Build Stage ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

# Copy everything first
COPY . .

# Ensure modules are clean
RUN go mod tidy

# Download modules
RUN go mod download

# Build
RUN CGO_ENABLED=0 go build -o app ./cmd

# ---------- Runtime Stage ----------
FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/app .

# Copy over runtime files
COPY cenv.schema.json .
COPY web web

# Create data directory for recipe storage
RUN mkdir -p data

EXPOSE 8080
CMD ["./app"]