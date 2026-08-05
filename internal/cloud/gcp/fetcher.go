package gcpcloud

import (
	"context"
	"fmt"
	"sync"

	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Fetcher retrieves GCP resources via cloud APIs.
type Fetcher struct {
	storage *StorageFetcher
	compute *ComputeFetcher
}

// NewFetcher creates a GCP cloud fetcher.
func NewFetcher() (*Fetcher, error) {
	storage, err := NewStorageFetcher()
	if err != nil {
		return nil, fmt.Errorf("init storage fetcher: %w", err)
	}
	compute, err := NewComputeFetcher()
	if err != nil {
		return nil, fmt.Errorf("init compute fetcher: %w", err)
	}
	return &Fetcher{storage: storage, compute: compute}, nil
}

// Provider returns the GCP provider identifier.
func (f *Fetcher) Provider() models.Provider {
	return models.ProviderGCP
}

// SupportedTypes lists resource types this fetcher can retrieve.
func (f *Fetcher) SupportedTypes() []string {
	return []string{"google_storage_bucket", "google_compute_instance"}
}

// Fetch retrieves GCP resources matching the filter.
func (f *Fetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	types := filter.ResourceTypes
	if len(types) == 0 {
		types = f.SupportedTypes()
	}
	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}

	var (
		mu        sync.Mutex
		resources []models.Resource
		errs      []error
		wg        sync.WaitGroup
	)

	fetch := func(fn func(context.Context, cloud.FetchFilter) ([]models.Resource, error), resourceType string) {
		if !typeSet[resourceType] {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := fn(ctx, filter)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", resourceType, err))
				return
			}
			resources = append(resources, res...)
		}()
	}

	fetch(f.storage.Fetch, "google_storage_bucket")
	fetch(f.compute.Fetch, "google_compute_instance")

	wg.Wait()
	if len(errs) > 0 {
		return resources, fmt.Errorf("partial fetch errors: %v", errs)
	}
	return resources, nil
}
