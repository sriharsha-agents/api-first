# ============================================
# Stage 1: Build
# ============================================
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Copy dependency manifests first for layer caching.
COPY go.mod go.sum ./
RUN go mod tidy

# Copy source code.
COPY . .

# Inject build metadata via ldflags.
ARG VERSION=dev
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.BuildAt=${BUILD_TIME}" \
    -o /out/server ./cmd/server

# ============================================
# Stage 2: Minimal runtime (non-root)
# ============================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1000 appgroup \
    && adduser -u 1000 -G appgroup -D -s /bin/sh appuser

WORKDIR /app

# Copy binary from builder stage.
COPY --from=builder /out/server /app/server

# Run as non-root user (enterprise security requirement).
USER appuser

EXPOSE 8080

# Health check probe for container orchestrators.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/server"]