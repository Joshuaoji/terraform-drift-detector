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

### Frontend development

```bash
# Terminal 1: API
./bin/driftdetect serve --port 8080 --config configs/example.yaml

# Terminal 2: Vite dev server (hot reload)
make dev-web
```

Open **http://localhost:5173** — the frontend lives in `web/` and proxies API calls to port 8080.

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
| GET | `/health` | Health check |
| GET | `/api/v1/profiles` | List scan profiles from config |
| POST | `/api/v1/scans` | Trigger a scan (async) |
| POST | `/api/v1/scans/profile/{name}` | Trigger a named profile |
| GET | `/api/v1/scans?summary=true` | List scans (lightweight) |
| GET | `/api/v1/scans/{id}` | Get scan status + report |
| GET | `/api/v1/scans/{id}/report` | Get drift report |

## Supported Resources

| Provider | Resource Types |
|----------|----------------|
| AWS | `aws_s3_bucket`, `aws_instance`, `aws_iam_role` |
| Azure | `azurerm_storage_account`, `azurerm_linux_virtual_machine` |
| GCP | `google_storage_bucket`, `google_compute_instance` |

## Remote State Backends

| Backend | URI format |
|---------|-----------|
| Local | `./terraform.tfstate` |
| S3 | `s3://bucket/key` |
| GCS | `gcs://bucket/key` |
| Azure Blob | `azure://account/container/key` |

## Development

```bash
make test
make build-all
```

## Roadmap

- **Phase 4**: PostgreSQL, auth, webhooks, expanded resource types

## License

MIT
