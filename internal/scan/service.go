package scan

import (
	"context"

	"github.com/terraform-drift-detector/driftdetect/internal/drift"
	"github.com/terraform-drift-detector/driftdetect/internal/state"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Service coordinates scan execution with persistence.
type Service struct {
	scanner *Scanner
	store   RunnerStore
}

// RunnerStore is the persistence layer required by the scan service.
type RunnerStore interface {
	CreateScan(ctx context.Context, opts models.ScanOptions) (*models.ScanRecord, error)
	UpdateStatus(ctx context.Context, id string, status models.ScanStatus, errMsg string) error
	SaveReport(ctx context.Context, report models.DriftReport) error
}

// NewService creates a scan service.
func NewService(scanner *Scanner, st RunnerStore) *Service {
	return &Service{scanner: scanner, store: st}
}

// RunAsync creates a scan record and executes it in the background.
func (s *Service) RunAsync(opts models.ScanOptions) (*models.ScanRecord, error) {
	record, err := s.store.CreateScan(context.Background(), opts)
	if err != nil {
		return nil, err
	}
	go s.execute(record.ID, opts)
	return record, nil
}

// Execute runs a scan synchronously for an existing scan ID.
func (s *Service) Execute(id string, opts models.ScanOptions) {
	s.execute(id, opts)
}

func (s *Service) execute(id string, opts models.ScanOptions) {
	ctx := context.Background()
	_ = s.store.UpdateStatus(ctx, id, models.ScanRunning, "")
	report, err := s.scanner.Run(ctx, opts)
	if err != nil {
		_ = s.store.UpdateStatus(ctx, id, models.ScanFailed, err.Error())
		return
	}
	report.ScanID = id
	_ = s.store.SaveReport(ctx, report)
	_ = s.store.UpdateStatus(ctx, id, models.ScanCompleted, "")
}

// NewDefaultScanner creates a scanner with default dependencies.
func NewDefaultScanner() *Scanner {
	return &Scanner{
		stateReader: state.NewReader(),
		engine:      drift.NewEngine(),
	}
}
