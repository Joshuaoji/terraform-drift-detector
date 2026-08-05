# Terraform Drift Detector

A cloud-agnostic platform that continuously compares Terraform state files against actual cloud infrastructure to identify configuration drift — without requiring `terraform plan` or `apply`.

## Features

- **State vs Cloud comparison** — Reads expected resources from Terraform state and actual resources from cloud provider APIs
- **Drift detection** — Identifies deleted resources, modified attributes, and tag changes
- **Multi-output** — Console (human-readable) and JSON (machine-readable) reports
- **Remote state backends** — S3, GCS, and Azure Blob Storage (Storage Account)
- **Multi-cloud fetchers** — AWS, Azure, and GCP resource inventory
- **REST API** — Trigger scans, list history, and retrieve reports
- **On-demand scans** — CLI-driven or API-driven scans with configurable resource types and regions

## Architecture

```
Terraform State → State Reader → Resource Extractor → Expected Model ─┐
                                                                       ├→ Drift Engine → Report Generator → CLI / JSON / API
Cloud APIs      → Cloud Fetcher → Resource Extractor → Actual Model  ─┘
```

## Quick Start

### Prerequisites

- Go 1.25+
- Cloud credentials for the provider you scan:
  - **AWS**: `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` or IAM role
  - **Azure**: `AZURE_SUBSCRIPTION_ID` + `az login` or managed identity
  - **GCP**: `GOOGLE_CLOUD_PROJECT` + Application Default Credentials

### Build

```bash
make build
```

### Run a Scan

```bash
# Local state file
./bin/driftdetect scan \
  --state testdata/sample.tfstate \
  --provider aws \
  --regions us-east-1 \
  --output console

# Remote S3 state
./bin/driftdetect scan \
  --state s3://my-tf-state-bucket/prod/terraform.tfstate \
  --provider aws \
  --output json

# Remote GCS state
./bin/driftdetect scan \
  --state gcs://my-tf-state-bucket/prod/terraform.tfstate \
  --provider gcp \
  --project-id my-gcp-project

# Remote Azure Blob state (Storage Account)
./bin/driftdetect scan \
  --state azure://mytfstateaccount/tfstate/prod.terraform.tfstate \
  --provider azure

# Use YAML config
./bin/driftdetect scan --config configs/example.yaml --scan-name prod-aws
```

### Start the API Server

```bash
./bin/driftdetect serve --port 8080 --db driftdetect.db
```

**API Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/scans` | Trigger a scan (async) |
| GET | `/api/v1/scans` | List scan history |
| GET | `/api/v1/scans/{id}` | Get scan status |
| GET | `/api/v1/scans/{id}/report` | Get drift report |

**Example API request:**

```bash
curl -X POST http://localhost:8080/api/v1/scans \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aws",
    "state_path": "testdata/sample.tfstate",
    "regions": ["us-east-1"],
    "resource_types": ["aws_s3_bucket", "aws_instance"]
  }'
```

### Exit Codes (scan)

| Code | Meaning |
|------|---------|
| 0 | No drift detected |
| 1 | Error |
| 2 | Drift detected |

## Supported Resources

| Provider | Resource Types |
|----------|----------------|
| AWS | `aws_s3_bucket`, `aws_instance`, `aws_iam_role` |
| Azure | `azurerm_storage_account`, `azurerm_linux_virtual_machine` |
| GCP | `google_storage_bucket`, `google_compute_instance` |

## Remote State Backends

| Backend | URI format | Auth |
|---------|-----------|------|
| Local | `./terraform.tfstate` | File system |
| S3 | `s3://bucket/key` | AWS default credential chain |
| GCS | `gcs://bucket/key` | GCP Application Default Credentials |
| Azure Blob | `azure://account/container/key` | `AZURE_STORAGE_CONNECTION_STRING` or Azure AD |

## Project Structure

```
├── cmd/driftdetect/       # CLI + API server entrypoint
├── internal/
│   ├── api/               # REST API handlers
│   ├── cloud/             # Cloud fetchers (aws/, azure/, gcp/)
│   ├── config/            # YAML configuration
│   ├── drift/             # Drift comparison engine
│   ├── extract/           # Resource normalizers
│   ├── report/            # Output formatters
│   ├── scan/              # Scan orchestration
│   ├── state/             # Terraform state readers (local, S3, GCS, Azure)
│   └── store/             # SQLite persistence
├── pkg/models/            # Domain types
├── configs/               # Example configuration
└── testdata/              # Test fixtures
```

## Development

```bash
make test        # Run unit tests
make test-cover  # Run tests with coverage
make lint        # Run go vet
```

## Roadmap

- **Phase 3**: Web dashboard, cron scheduler
- **Phase 4**: PostgreSQL, auth, webhooks, expanded resource types

## License

MIT
