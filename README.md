# API-First

Enterprise-grade, air-gapped API module for internal enterprise systems. Zero data leaves the air-gapped boundary.

> **Design principle:** Every configuration value comes from environment variables at runtime. Nothing is hardcoded.

---

## Architecture

```
┌──────────────────────────────────────────────────┐
│                    Air-Gapped Boundary               │
│                                                  │
│  ┌─────────┐    ┌──────────┐    ┌────────────┐   │
│  │  Gin    │───▶│PostgreSQL│    │   Redis  │   │
│  │  API    │◀───│  v16     │    │  Cache   │   │
│  │  :8080  │    └──────────┘    └────────────┘   │
│  └─────────┘                                    │
│       │                                       │
│       ▼                                       │
│  Structured JSON logs → stdout → SIEM           │
└──────────────────────────────────────────────────┘
```

| Layer           | Technology              |
|------------------|------------------------|
| HTTP framework   | Gin (Go)              |
| ORM            | GORM v2               |
| Database         | PostgreSQL 16       |
| Cache            | Redis 7             |
| Logger           | Custom JSON (stdout)  |
| Container        | Docker (multi-stage) |
| Orchestration     | Kubernetes (Helm)   |

---

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- kubectl + Helm 3 (for K8s deployment)

### Local development with Docker Compose

```bash
# Clone
git clone git@github.com:h9945394143/API-First.git
cd API-First

# Start all services (API + PostgreSQL + Redis)
docker compose up -d

# View logs
docker compose logs -f api
```

The API is available at `http://localhost:8080`.

### Build and run locally (Go)

```bash
# Set required environment variables
export DATABASE_HOST=localhost
export DATABASE_USER=postgres
export DATABASE_PASSWORD=postgres
export DATABASE_NAME=apifirst
export REDIS_ADDR=localhost:6379

# Run
go run ./cmd/server
```

---

## Environment Variables

All configuration is passed via environment variables. No defaults are baked into the binary for sensitive values.

| Variable               | Required | Default        | Description                  |
|---------------------|---------|----------------|------------------------------|
| `SERVER_PORT`       | No      | `8080`        | HTTP listen port              |
| `SERVER_READ_TIMEOUT` | No    | `5s`          | Read timeout                   |
| `SERVER_WRITE_TIMEOUT`| No    | `10s`         | Write timeout                  |
| `SERVER_SHUTDOWN_TIMEOUT`| No | `5s`        | Graceful shutdown timeout     |
| `DATABASE_HOST`     | **Yes** | —            | PostgreSQL hostname          |
| `DATABASE_PORT`     | No      | `5432`        | PostgreSQL port             |
| `DATABASE_USER`     | **Yes** | —            | PostgreSQL username          |
| `DATABASE_PASSWORD` | **Yes** | —            | PostgreSQL password          |
| `DATABASE_NAME`     | No      | `apifirst`    | Database name                |
| `DATABASE_SSLMODE`  | No      | `disable`      | SSL mode                   |
| `DATABASE_MAX_CONNS`| No    | `25`          | Max connections            |
| `DATABASE_MIN_CONNS`| No    | `2`           | Min idle connections       |
| `DATABASE_MAX_LIFETIME`| No | `3600s`     | Max connection lifetime     |
| `REDIS_ADDR`       | No      | `localhost:6379` | Redis address           |
| `REDIS_DB`         | No      | `0`           | Redis DB number            |
| `LOG_LEVEL`       | No      | `info`        | Log level (debug/info/warn/error) |

---

## API Reference

### Health Checks

| Endpoint     | Purpose                        |
|------------|------------------------------|
| `GET /healthz`   | Liveness probe (always 200)  |
| `GET /readyz`   | Readiness probe (checks DB) |

### Modules

| Method | Path              | Description              |
|------|-------------------|--------------------------|
| GET    | `/api/v1/modules`       | List all modules         |
| POST   | `/api/v1/modules`       | Create a new module       |
| GET    | `/api/v1/modules/:id`   | Get module by ID         |
| PUT    | `/api/v1/modules/:id`   | Update a module            |
| DELETE | `/api/v1/modules/:id`   | Delete a module            |

### Deployments

| Method | Path                 | Description                |
|------|---------------------|--------------------------|
| GET    | `/api/v1/deployments`        | List all deployments     |
| POST   | `/api/v1/deployments`        | Create a deployment        |
| GET    | `/api/v1/deployments/:id`    | Get deployment by ID      |
| PUT    | `/api/v1/deployments/:id`    | Update a deployment         |
| DELETE | `/api/v1/deployments/:id`    | Delete a deployment         |

---

## Deployment

### Docker

```bash
# Build
docker build -t api-first:0.1.0 .

# Run
docker run -d \
  -p 8080:8080 \
  -e DATABASE_HOST=my-db \
  -e DATABASE_USER=admin \
  -e DATABASE_PASSWORD=secret \
  -e REDIS_ADDR=my-redis:6379 \
  api-first:0.1.0
```

### Kubernetes (Helm)

```bash
# Install
helm install api-first ./deploy/helm/api-first \
  --set database.host=postgres-svc \
  --set database.user=admin \
  --set database.password=secret \
  --set redis.addr=redis-svc:6379

# Upgrade
helm upgrade api-first ./deploy/helm/api-first -f values override.yaml

# Uninstall
helm uninstall api-first
```

Customise via `deploy/helm/api-first/values.yaml` or pass `-f custom-values.yaml`.

---

## Security

- **Non-root container:** Process runs as UID 1000.
- **Read-only filesystem:** Container filesystem is read-only.
- **No capability escalation:** All Linux capabilities dropped.
- **Air-gapped:** No outbound network calls. All data stays within the boundary.
- **Structured JSON logs:** Every request logged to stdout for SIEM ingestion (Splunk, ELK, etc.).
- **Graceful shutdown:** Clean connection drain on SIGTERM.

---

## Project Structure

```
├── cmd/server/           # Application entry point
├── internal/
│   ├── cache/            # Redis caching layer
│   ├── config/           # Environment configuration
│   ├── database/         # PostgreSQL + GORM
│   ├── handler/           # HTTP handlers & routes
│   ├── logger/           # Structured JSON logger
│   └── models/          # Data models
├── deploy/helm/api-first/  # Helm chart
├── Dockerfile            # Multi-stage build
├── docker-compose.yml      # Local development stack
└── go.mod
```

---

## License

Internal use only. © 2025