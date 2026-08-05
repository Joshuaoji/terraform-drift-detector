package scan

import (
	"context"
	"fmt"

	azurecloud "github.com/terraform-drift-detector/driftdetect/internal/cloud/azure"
	awscloud "github.com/terraform-drift-detector/driftdetect/internal/cloud/aws"
	gcpcloud "github.com/terraform-drift-detector/driftdetect/internal/cloud/gcp"
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
		stateReader: state.NewReader(),
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
	source := opts.ResolvedStateSource()
	if source.Display() == "" {
		return models.DriftReport{}, fmt.Errorf("state source is required")
	}
	if opts.Provider == "" {
		return models.DriftReport{}, fmt.Errorf("provider is required")
	}

	expected, err := s.stateReader.Read(ctx, source)
	if err != nil {
		return models.DriftReport{}, fmt.Errorf("read state: %w", err)
	}

	expected = extract.FilterByProvider(expected, opts.Provider)
	expected = extract.FilterByTypes(expected, opts.ResourceTypes)

	fetcher, err := s.cloudFetcher(opts.Provider)
	if err != nil {
		return models.DriftReport{}, fmt.Errorf("init cloud fetcher: %w", err)
	}

	projectID := opts.ProjectID
	if projectID == "" {
		projectID = opts.AccountID
	}

	actual, err := fetcher.Fetch(ctx, cloud.FetchFilter{
		Regions:       opts.Regions,
		ResourceTypes: opts.ResourceTypes,
		AccountID:     opts.AccountID,
		ProjectID:     projectID,
	})
	if err != nil {
		return models.DriftReport{}, fmt.Errorf("fetch cloud resources: %w", err)
	}

	report := s.engine.Compare(source.Display(), opts.Provider, expected, actual)
	return report, nil
}

func (s *Scanner) cloudFetcher(provider models.Provider) (cloud.Fetcher, error) {
	switch provider {
	case models.ProviderAWS:
		return awscloud.NewFetcher()
	case models.ProviderAzure:
		return azurecloud.NewFetcher()
	case models.ProviderGCP:
		return gcpcloud.NewFetcher()
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
