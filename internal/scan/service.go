package scan

import (
	"context"
	"log/slog"
	"time"

	"github.com/terraform-drift-detector/driftdetect/internal/drift"
	"github.com/terraform-drift-detector/driftdetect/internal/observability"
	"github.com/terraform-drift-detector/driftdetect/internal/state"
	"github.com/terraform-drift-detector/driftdetect/internal/store"
	"github.com/terraform-drift-detector/driftdetect/internal/webhook"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Service coordinates scan execution with persistence and notifications.
type Service struct {
	scanner  *Scanner
	store    RunnerStore
	webhooks *webhook.Notifier
	fullStore store.Store
	log      *slog.Logger
}

// RunnerStore is the persistence layer required by the scan service.
type RunnerStore interface {
	CreateScan(ctx context.Context, opts models.ScanOptions) (*models.ScanRecord, error)
	UpdateStatus(ctx context.Context, id string, status models.ScanStatus, errMsg string) error
	SaveReport(ctx context.Context, report models.DriftReport) error
}

// NewService creates a scan service.
func NewService(scanner *Scanner, st RunnerStore, opts ...ServiceOption) *Service {
	s := &Service{
		scanner: scanner,
		store:   st,
		log:     slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ServiceOption configures the scan service.
type ServiceOption func(*Service)

// WithWebhooks attaches a webhook notifier.
func WithWebhooks(n *webhook.Notifier) ServiceOption {
	return func(s *Service) {
		s.webhooks = n
	}
}

// WithFullStore provides full store access for post-scan record retrieval.
func WithFullStore(st store.Store) ServiceOption {
	return func(s *Service) {
		s.fullStore = st
	}
}

// WithLogger sets the service logger.
func WithLogger(log *slog.Logger) ServiceOption {
	return func(s *Service) {
		if log != nil {
			s.log = log
		}
	}
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
	started := time.Now()
	_ = s.store.UpdateStatus(ctx, id, models.ScanRunning, "")

	report, err := s.scanner.Run(ctx, opts)
	if err != nil {
		_ = s.store.UpdateStatus(ctx, id, models.ScanFailed, err.Error())
		observability.RecordScanFailed(string(opts.Provider))
		s.log.Error("scan failed", "scan_id", id, "error", err)
		if s.webhooks != nil {
			record := s.getRecord(ctx, id)
			s.webhooks.NotifyScanFailed(record, err.Error())
		}
		return
	}

	report.ScanID = id
	_ = s.store.SaveReport(ctx, report)
	_ = s.store.UpdateStatus(ctx, id, models.ScanCompleted, "")
	observability.RecordScanCompleted(string(opts.Provider), time.Since(started), report.Summary.TotalDrifts)
	s.log.Info("scan completed", "scan_id", id, "drifts", report.Summary.TotalDrifts)

	if s.webhooks != nil {
		record := s.getRecord(ctx, id)
		s.webhooks.NotifyScanCompleted(record, report)
	}
}

func (s *Service) getRecord(ctx context.Context, id string) *models.ScanRecord {
	if s.fullStore == nil {
		return &models.ScanRecord{ID: id}
	}
	record, err := s.fullStore.GetScan(ctx, id)
	if err != nil {
		return &models.ScanRecord{ID: id}
	}
	return record
}

// NewDefaultScanner creates a scanner with default dependencies.
func NewDefaultScanner() *Scanner {
	return &Scanner{
		stateReader: state.NewReader(),
		engine:      drift.NewEngine(),
	}
}
