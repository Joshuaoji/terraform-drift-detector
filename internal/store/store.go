package store

import (
	"context"

	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Store persists scan records and drift reports.
type Store interface {
	CreateScan(ctx context.Context, opts models.ScanOptions) (*models.ScanRecord, error)
	UpdateStatus(ctx context.Context, id string, status models.ScanStatus, errMsg string) error
	SaveReport(ctx context.Context, report models.DriftReport) error
	GetScan(ctx context.Context, id string) (*models.ScanRecord, error)
	ListScans(ctx context.Context, limit int) ([]models.ScanRecord, error)
}
