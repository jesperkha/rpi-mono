# ---------- Build Stage ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git gcc musl-dev

# Copy everything first
COPY . .

# Ensure modules are clean
RUN go mod tidy

# Download modules
RUN go mod download

# Build (CGO required for mattn/go-sqlite3)
RUN CGO_ENABLED=1 GOOS=linux \
    go build -o app ./cmd

# ---------- Runtime Stage ----------
FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/app .
COPY --from=builder /app/cenv.schema.json .
COPY --from=builder /app/sql sql
COPY --from=builder /app/web web

RUN mkdir -p /data/images

EXPOSE 8080
CMD ["./app"]