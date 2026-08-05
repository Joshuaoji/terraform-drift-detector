package scan

import (
	"context"
	"fmt"

	awscloud "github.com/terraform-drift-detector/driftdetect/internal/cloud/aws"
	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/internal/drift"
	"github.com/terraform-drift-detector/driftdetect/internal/extract"
	"github.com/terraform-drift-detector/driftdetect/internal/state"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Scanner orchestrates state reading, cloud fetching, and drift comparison.
type Scanner struct {
	stateReader state.Reader
	engine      *drift.Engine
}

// NewScanner creates a drift scanner.
func NewScanner() *Scanner {
	return &Scanner{
		stateReader: state.NewLocalReader(),
		engine:      drift.NewEngine(),
	}
}

// NewScannerWithDeps creates a scanner with injectable dependencies (for testing).
func NewScannerWithDeps(reader state.Reader, engine *drift.Engine) *Scanner {
	return &Scanner{
		stateReader: reader,
		engine:      engine,
	}
}

// Run executes a full drift scan.
func (s *Scanner) Run(ctx context.Context, opts models.ScanOptions) (models.DriftReport, error) {
	if opts.StatePath == "" {
		return models.DriftReport{}, fmt.Errorf("state path is required")
	}
	if opts.Provider == "" {
		return models.DriftReport{}, fmt.Errorf("provider is required")
	}

	expected, err := s.stateReader.Read(ctx, opts.StatePath)
	if err != nil {
		return models.DriftReport{}, fmt.Errorf("read state: %w", err)
	}

	expected = extract.FilterByProvider(expected, opts.Provider)
	expected = extract.FilterByTypes(expected, opts.ResourceTypes)

	fetcher, err := s.cloudFetcher(opts.Provider)
	if err != nil {
		return models.DriftReport{}, fmt.Errorf("init cloud fetcher: %w", err)
	}

	actual, err := fetcher.Fetch(ctx, cloud.FetchFilter{
		Regions:       opts.Regions,
		ResourceTypes: opts.ResourceTypes,
		AccountID:     opts.AccountID,
	})
	if err != nil {
		return models.DriftReport{}, fmt.Errorf("fetch cloud resources: %w", err)
	}

	report := s.engine.Compare(opts.StatePath, opts.Provider, expected, actual)
	return report, nil
}

func (s *Scanner) cloudFetcher(provider models.Provider) (cloud.Fetcher, error) {
	switch provider {
	case models.ProviderAWS:
		return awscloud.NewFetcher()
	case models.ProviderAzure:
		return nil, fmt.Errorf("azure provider not yet implemented (phase 2)")
	case models.ProviderGCP:
		return nil, fmt.Errorf("gcp provider not yet implemented (phase 2)")
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
