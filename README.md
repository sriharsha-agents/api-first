# API-First

Enterprise-grade, air-gapped API module. Zero data leaves the boundary. All config via environment variables.

---

## Architecture

```
┌──────────────────────────────────────────────┐
│           Air-Gapped Boundary                │
│  ┌───────┐   ┌──────────┐   ┌────────┐      │
│  │ Gin │──▶│PostgreSQL│   │ Redis │      │
│  │ API │◀──│  v16    │   │ Cache  │      │
│  │8080  │   └──────────┘   └────────┘      │
│  └───────┘                                   │
│       ▼                                  │
│  JSON logs → stdout → SIEM               │
└──────────────────────────────────────────────┘
```

| Layer | Technology |
|---|---|
| HTTP | Gin (Go) |
| ORM | GORM v2 |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Logger | logrus (JSON) |
| License | ECDSA P-256 (offline) |
| Container | Docker (multi-stage) |
| Orchestration | Kubernetes (Helm) |

---

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- kubectl + Helm 3 (for K8s)

### Docker Compose

```bash
git clone git@github.com:h9945394143/API-First.git
cd API-First
docker compose up -d
docker compose logs -f api
```

API available at `http://localhost:8080`.

### Build from source

```bash
export DATABASE_HOST=localhost DATABASE_USER=postgres DATABASE_PASSWORD=postgres
export DATABASE_NAME=apifirst REDIS_ADDR=localhost:6379
go run ./cmd/server
```

### Makefile targets

```bash
make build        # Build server binary to ./out/api-server
make license-gen    # Build license generator CLI to ./out/license-gen
make test          # Run all tests (go test ./...)
make dev-license    # Generate a development license (./dev.lic)
make clean        # Remove ./out directory
```

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `LICENSE_FILE_PATH` | No | `/etc/myapp/license.lic` | Path to license file |
| `SERVER_PORT` | No | `8080` | HTTP listen port |
| `SERVER_READ_TIMEOUT` | No | `5s` | Read timeout |
| `SERVER_WRITE_TIMEOUT` | No | `10s` | Write timeout |
| `SERVER_SHUTDOWN_TIMEOUT` | No | `5s` | Graceful shutdown timeout |
| `DATABASE_HOST` | **Yes** | — | PostgreSQL hostname |
| `DATABASE_PORT` | No | `5432` | PostgreSQL port |
| `DATABASE_USER` | **Yes** | — | PostgreSQL username |
| `DATABASE_PASSWORD` | **Yes** | — | PostgreSQL password |
| `DATABASE_NAME` | No | `apifirst` | Database name |
| `DATABASE_SSLMODE` | No | `disable` | SSL mode |
| `DATABASE_MAX_CONNS` | No | `25` | Max connections |
| `DATABASE_MIN_CONNS` | No | `2` | Min idle connections |
| `DATABASE_MAX_LIFETIME` | No | `3600s` | Max conn lifetime |
| `REDIS_ADDR` | No | `localhost:6379` | Redis address |
| `REDIS_DB` | No | `0` | Redis DB number |
| `LOG_LEVEL` | No | `info` | Log level (debug/info/warn/error) |

---

## API Reference

### Health Checks

| Method | Path | Purpose |
|---|---|---|
| GET | `/healthz` | Liveness probe (always 200) |
| GET | `/readyz` | Readiness probe (checks DB + Redis) |

### Modules

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/modules` | List all modules |
| POST | `/api/v1/modules` | Create a module |
| GET | `/api/v1/modules/:id` | Get module by ID |
| PUT | `/api/v1/modules/:id` | Update a module |
| DELETE | `/api/v1/modules/:id` | Delete a module |

### Deployments

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/deployments` | List all deployments |
| POST | `/api/v1/deployments` | Create a deployment |
| PUT | `/api/v1/deployments/:id/status` | Update deployment status |

**Deployment status values:** `pending`, `running`, `completed`, `failed`
- Setting status to `running` auto-sets `started_at`
- Setting status to `completed` or `failed` auto-sets `completed_at`

---

## License System

The server validates an offline cryptographic license on startup before accepting any requests.

### How it works

1. A license file is generated using the `license-gen` CLI (signed with ECDSA P-256)
2. The server embeds the corresponding public key at compile time
3. On startup, the server validates: signature, expiration, hardware ID binding, clock tampering
4. Validation fails → server refuses to start

### Generate a license

```bash
# Build the license generator
make license-gen

# Generate a license file
SIGNING_PRIVATE_KEY="$(cat cmd/license-gen/private.pem)" \
  ./out/license-gen \
    -customer="ACME-CORP" \
    -expiry=365 \
    -hwid="" \
    -features="ha-mode,audit-log" \
    -max-instances=5 \
    -out=acme.lic
```

### Clock tampering detection

The build timestamp is injected into the binary via `ldflags`. If the system clock is rolled back to a time before the binary was built, the server detects tampering and refuses to start.

### Hardware ID locking

Licenses can be bound to a specific machine using a hardware fingerprint (MAC + hostname). A license generated with `-hwid=<id>` will only validate on that machine.

---

## Deployment

### Docker

```bash
docker build -t api-first:0.1.0 .

docker run -d \
  -p 8080:8080 \
  -v ./acme.lic:/etc/myapp/license.lic:ro \
  -e DATABASE_HOST=my-db \
  -e DATABASE_USER=admin \
  -e DATABASE_PASSWORD=secret \
  -e REDIS_ADDR=my-redis:6379 \
  api-first:0.1.0
```

### Kubernetes (Helm)

```bash
helm install api-first ./deploy/helm/api-first \
  --set database.host=postgres-svc \
  --set database.user=admin \
  --set database.password=secret \
  --set redis.addr=redis-svc:6379

helm upgrade api-first ./deploy/helm/api-first -f custom-values.yaml
helm uninstall api-first
```

Mount the license file via a Kubernetes Secret or ConfigMap volume.

---

## Security

- **Non-root container:** Runs as UID 1000
- **Read-only filesystem:** Container FS is read-only
- **No capability escalation:** All Linux caps dropped
- **Air-gapped:** No outbound network calls
- **Structured JSON logs:** Every request to stdout for SIEM (Splunk, ELK)
- **ECDSA license signing:** Offline cryptographic validation
- **Clock tampering detection:** Build-time timestamp injection
- **Hardware ID locking:** Node-bound license files
- **Graceful shutdown:** Clean connection drain on SIGTERM

---

## Project Structure

```
├── cmd/
│   ├── server/           # Main server entry point
│   └── license-gen/      # License generation CLI (private key)
├── internal/
│   ├── cache/            # Redis client wrapper
│   ├── config/           # Environment config loader
│   ├── database/         # PostgreSQL + GORM setup
│   ├── handler/           # HTTP handlers & route registration
│   ├── logger/           # logrus JSON logger
│   └── models/          # GORM data models
├── pkg/
│   └── license/          # License validation (embedded public key)
├── deploy/
│   └── helm/api-first/ # Helm chart
├── licensing/            # Commercial license text
├── test.lic/            # Development test license
├── Dockerfile           # Multi-stage build
├── docker-compose.yml     # Local dev stack
├── Makefile             # Build targets
└── go.mod
```

---

## Licensing

- **Community Edition:** Apache 2.0. Free for internal use.
- **Enterprise Edition:** Commercial license required. Adds HA clustering, audit logs, and more.
- **Air-gapped validation:** License checked entirely offline via cryptographic signatures. No phone-home.