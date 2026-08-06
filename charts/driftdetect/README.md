# Driftdetect Helm Chart

Deploy the Terraform drift detector to Kubernetes (AWS EKS).

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  EKS Cluster                                            │
│                                                         │
│  ┌──────────────┐    ┌─────────────────────────────┐   │
│  │ ALB Ingress  │───▶│ driftdetect Deployment (x2) │   │
│  └──────────────┘    │  • API + embedded dashboard │   │
│                      │  • /metrics                   │   │
│                      └──────────────┬────────────────┘   │
│                                     │                    │
│                      ┌──────────────▼────────────────┐   │
│                      │ Amazon RDS (recommended)      │   │
│                      │ or in-cluster PostgreSQL      │   │
│                      └───────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

The Helm chart deploys a **single application image** (Go backend with embedded React frontend). There is no separate frontend pod.

## Prerequisites

- EKS cluster (1.27+)
- `kubectl` and `helm` 3.x
- AWS Load Balancer Controller (for ALB ingress)
- Amazon RDS PostgreSQL (recommended) **or** enable in-cluster PostgreSQL
- ECR repository for the application image
- (Optional) IRSA role for S3/GCS/Azure state access and cloud API calls

## Quick start (local Helm)

```bash
# Lint the chart
helm lint ./charts/driftdetect

# Dry-run with in-cluster PostgreSQL (dev/staging only)
helm upgrade --install driftdetect ./charts/driftdetect \
  --namespace driftdetect --create-namespace \
  --set image.repository=<account>.dkr.ecr.<region>.amazonaws.com/driftdetect \
  --set image.tag=latest \
  --set postgresql.enabled=true \
  --set ingress.enabled=false

# Production-style deploy with external secrets
kubectl create namespace driftdetect

kubectl -n driftdetect create secret generic driftdetect-database \
  --from-literal=database-url='postgres://user:pass@rds-endpoint:5432/driftdetect?sslmode=require'

kubectl -n driftdetect create secret generic driftdetect-api-keys \
  --from-literal=api-keys='your-production-api-key'

helm upgrade --install driftdetect ./charts/driftdetect \
  --namespace driftdetect \
  -f charts/driftdetect/values-eks.example.yaml \
  --set image.repository=<account>.dkr.ecr.<region>.amazonaws.com/driftdetect \
  --set image.tag=<git-sha>
```

## GitHub Actions CD

The repository includes `.github/workflows/cd.yaml` which:

1. Builds the Docker image
2. Pushes to Amazon ECR
3. Deploys to EKS with `helm upgrade --install`

### Required GitHub configuration

**Repository variables** (`Settings → Secrets and variables → Actions → Variables`):

| Variable | Example |
|----------|---------|
| `AWS_REGION` | `us-east-1` |
| `ECR_REPOSITORY` | `driftdetect` |
| `EKS_CLUSTER_NAME` | `my-eks-cluster` |
| `EKS_NAMESPACE` | `driftdetect` |
| `AWS_ROLE_ARN` | `arn:aws:iam::123456789012:role/github-actions-eks` |

**OIDC trust**: Configure an IAM role that trusts GitHub Actions OIDC and grants:
- `ecr:GetAuthorizationToken`, `ecr:BatchCheckLayerAvailability`, `ecr:PutImage`, etc.
- `eks:DescribeCluster`
- `sts:AssumeRole` for any IRSA roles if needed

### Manual deploy trigger

```bash
gh workflow run cd.yaml -f environment=production
```

## Values reference

| Key | Description |
|-----|-------------|
| `image.repository` | ECR image URI (without tag) |
| `image.tag` | Image tag (defaults to chart appVersion) |
| `database.url` | Postgres DSN (if not using `existingSecret`) |
| `database.existingSecret` | Secret containing RDS connection string |
| `apiKeys.existingSecret` | Secret with comma-separated API keys |
| `config.scans` | Scan profiles (rendered into ConfigMap) |
| `ingress.enabled` | Enable ALB ingress |
| `postgresql.enabled` | Deploy in-cluster Postgres (non-prod only) |
| `serviceAccount.annotations` | IRSA role ARN |
| `serviceMonitor.enabled` | Prometheus Operator scrape config |

## Upgrades

```bash
helm upgrade driftdetect ./charts/driftdetect \
  --namespace driftdetect \
  -f my-values.yaml \
  --set image.tag=<new-tag>
```

## Uninstall

```bash
helm uninstall driftdetect --namespace driftdetect
```
