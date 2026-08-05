# Terraform Drift Detector

A cloud-agnostic platform that continuously compares Terraform state files against actual cloud infrastructure to identify configuration drift — without requiring `terraform plan` or `apply`.

## Features

- **State vs Cloud comparison** — Reads expected resources from Terraform state and actual resources from cloud provider APIs
- **Drift detection** — Identifies deleted resources, modified attributes, and tag changes
- **Multi-output** — Console (human-readable) and JSON (machine-readable) reports
- **Remote state backends** — S3, GCS, and Azure Blob Storage (Storage Account)
- **Multi-cloud fetchers** — AWS, Azure, and GCP resource inventory
- **REST API** — Trigger scans, list history, and retrieve reports
- **Web dashboard** — View scan history, drift reports, and trigger scans from the browser
- **Cron scheduler** — Run scan profiles on a schedule from YAML config
- **PostgreSQL or SQLite** — Production-ready Postgres store with SQLite for local dev
- **API key authentication** — Protect `/api/v1` routes with `X-API-Key` or Bearer tokens
- **Webhooks** — HMAC-signed notifications on `scan.completed` and `scan.failed`
- **Prometheus metrics** — Scan counts, durations, and drift totals at `/metrics`
- **On-demand scans** — CLI-driven or API-driven scans with configurable resource types and regions

## Architecture

```
Terraform State → State Reader → Resource Extractor → Expected Model ─┐
                                                                       ├→ Drift Engine → Report → CLI / JSON / API / Dashboard
Cloud APIs      → Cloud Fetcher → Resource Extractor → Actual Model  ─┘
                                        ↑
                              Cron Scheduler (YAML profiles)
```

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 22+ (for frontend builds)
- Cloud credentials for the provider you scan

### Build

```bash
make build-all   # builds React frontend + Go binary
```

For Go-only builds (without rebuilding the frontend):

```bash
make build
```

### Start Server with Dashboard

```bash
./bin/driftdetect serve --port 8080 --config configs/example.yaml
```

Open **http://localhost:8080/** for the React dashboard.

### Docker Compose (PostgreSQL)

```bash
docker compose up --build
```

This starts PostgreSQL and the API on port 8080 with metrics enabled. The default API key is `dev-api-key` (set via `DRIFTDETECT_API_KEYS`).

### Frontend development

```bash
# Terminal 1: API
./bin/driftdetect serve --port 8080 --config configs/example.yaml

# Terminal 2: Vite dev server (hot reload)
make dev-web
```

Open **http://localhost:5173** — the frontend lives in `web/` and proxies API calls to port 8080.

When API keys are enabled, set `VITE_API_KEY` in `web/.env.local`:

```bash
VITE_API_KEY=dev-api-key
```

The scheduler automatically runs any scan profile that defines a `schedule` cron expression in the config file.

### Run a CLI Scan

```bash
./bin/driftdetect scan \
  --state testdata/sample.tfstate \
  --provider aws \
  --output console
```

## Web Dashboard

The React dashboard (`web/`) provides:

- **Scan profiles** — View configured profiles and trigger on-demand runs
- **Scan history** — Status, provider, drift counts, timestamps (auto-refreshes while scans run)
- **Drift detail** — Summary cards and per-resource drift breakdown
- **New scan form** — Trigger ad-hoc scans without editing config

## Scheduler

Define cron schedules in `configs/example.yaml`:

```yaml
scans:
  - name: prod-aws
    provider: aws
    schedule: "0 */6 * * *"   # every 6 hours
    state:
      backend: local
      path: ./testdata/sample.tfstate
```

Start the server with `--config` to activate scheduled scans. Overlapping runs for the same profile are skipped.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (includes DB status) |
| GET | `/metrics` | Prometheus metrics (when `--metrics` is set) |
| GET | `/api/v1/profiles` | List scan profiles from config |
| POST | `/api/v1/scans` | Trigger a scan (async) |
| POST | `/api/v1/scans/profile/{name}` | Trigger a named profile |
| GET | `/api/v1/scans?summary=true` | List scans (lightweight) |
| GET | `/api/v1/scans/{id}` | Get scan status + report |
| GET | `/api/v1/scans/{id}/report` | Get drift report |

### Authentication

When API keys are configured (via `server.api_keys` in YAML, `DRIFTDETECT_API_KEYS` env, or `--api-key`), protect API routes with:

```
X-API-Key: your-key
```

or

```
Authorization: Bearer your-key
```

The dashboard and `/health` remain public.

## Webhooks

Configure outbound webhooks in YAML:

```yaml
webhooks:
  - name: alerts
    url: https://hooks.example.com/driftdetect
    events:
      - scan.completed
      - scan.failed
    secret: signing-secret
```

Payloads are signed with `X-Driftdetect-Signature: sha256=<hmac>` when a secret is set.

## Supported Resources

| Provider | Resource Types |
|----------|----------------|
| AWS | `aws_s3_bucket`, `aws_instance`, `aws_iam_role`, `aws_vpc`, `aws_security_group` |
| Azure | `azurerm_storage_account`, `azurerm_linux_virtual_machine`, `azurerm_resource_group` |
| GCP | `google_storage_bucket`, `google_compute_instance`, `google_compute_network`, `google_compute_firewall` |

## Remote State Backends

| Backend | URI format |
|---------|-----------|
| Local | `./terraform.tfstate` |
| S3 | `s3://bucket/key` |
| GCS | `gcs://bucket/key` |
| Azure Blob | `azure://account/container/key` |

## Production Deployment

```bash
./bin/driftdetect serve \
  --port 8080 \
  --db-url postgres://user:pass@host:5432/driftdetect \
  --config configs/example.yaml \
  --metrics
```

## Development

```bash
make test
make build-all
```

CI runs on every push/PR via GitHub Actions (`.github/workflows/ci.yml`).

## License

MIT
