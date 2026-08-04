# Terraform Drift Detector

A cloud-agnostic platform that continuously compares Terraform state files against actual cloud infrastructure to identify configuration drift — without requiring `terraform plan` or `apply`.

## Features

- **State vs Cloud comparison** — Reads expected resources from Terraform state and actual resources from cloud provider APIs
- **Drift detection** — Identifies deleted resources, modified attributes, and tag changes
- **Multi-output** — Console (human-readable) and JSON (machine-readable) reports
- **Cloud-agnostic design** — Extensible provider interface (AWS implemented in Phase 1)
- **On-demand scans** — CLI-driven scans with configurable resource types and regions

## Architecture

```
Terraform State → State Reader → Resource Extractor → Expected Model ─┐
                                                                       ├→ Drift Engine → Report Generator → CLI / JSON
Cloud APIs      → Cloud Fetcher → Resource Extractor → Actual Model  ─┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- AWS credentials configured (for AWS scans): `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, or IAM role

### Build

```bash
make build
```

### Run a Scan

```bash
# Scan against local state file
./bin/driftdetect scan \
  --state testdata/sample.tfstate \
  --provider aws \
  --regions us-east-1 \
  --output console

# JSON output for CI/CD
./bin/driftdetect scan \
  --state testdata/sample.tfstate \
  --provider aws \
  --output json

# Use YAML config
./bin/driftdetect scan --config configs/example.yaml --scan-name prod-aws
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No drift detected |
| 1 | Error |
| 2 | Drift detected |

## Supported Resources (Phase 1 — AWS)

| Resource Type | State Reader | Cloud Fetcher |
|---------------|-------------|---------------|
| `aws_s3_bucket` | ✅ | ✅ |
| `aws_instance` | ✅ | ✅ |
| `aws_iam_role` | ✅ | ✅ |

## Project Structure

```
├── cmd/driftdetect/       # CLI entrypoint
├── internal/
│   ├── cloud/             # Cloud provider fetchers (aws/)
│   ├── config/            # YAML configuration
│   ├── drift/             # Drift comparison engine
│   ├── extract/           # Resource normalizers
│   ├── report/            # Output formatters
│   ├── scan/              # Scan orchestration
│   └── state/             # Terraform state readers
├── pkg/models/            # Domain types
├── configs/               # Example configuration
└── testdata/              # Test fixtures
```

## Configuration

See `configs/example.yaml` for scan profile examples including regions, resource types, and schedules.

## Development

```bash
make test        # Run unit tests
make test-cover  # Run tests with coverage
make lint        # Run go vet
```

## Roadmap

- **Phase 2**: Azure + GCP fetchers, remote state backends (S3, GCS), REST API
- **Phase 3**: Web dashboard, cron scheduler
- **Phase 4**: PostgreSQL, auth, webhooks, expanded resource types

## License

MIT
