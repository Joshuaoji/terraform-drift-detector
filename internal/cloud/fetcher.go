package cloud

import (
	"context"

	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// FetchFilter scopes a cloud inventory fetch.
type FetchFilter struct {
	Regions       []string
	ResourceTypes []string
	AccountID     string
	ProjectID     string
}

// Fetcher retrieves live resources from a cloud provider.
type Fetcher interface {
	Provider() models.Provider
	Fetch(ctx context.Context, filter FetchFilter) ([]models.Resource, error)
	SupportedTypes() []string
}
