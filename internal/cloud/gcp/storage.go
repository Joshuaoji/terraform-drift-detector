package gcpcloud

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// StorageFetcher lists GCS buckets.
type StorageFetcher struct {
	client    *storage.Client
	projectID string
}

// NewStorageFetcher creates a GCS bucket fetcher.
func NewStorageFetcher() (*StorageFetcher, error) {
	projectID, err := defaultProjectID()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithScopes(storage.ScopeReadOnly))
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	return &StorageFetcher{client: client, projectID: projectID}, nil
}

// Fetch retrieves GCS buckets in the project.
func (f *StorageFetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	projectID := f.projectID
	if filter.AccountID != "" {
		projectID = filter.AccountID
	}

	var resources []models.Resource
	it := f.client.Buckets(ctx, projectID)
	for {
		bucket, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list buckets: %w", err)
		}

		attrs := map[string]any{
			"name": bucket.Name,
		}
		if bucket.Location != "" {
			attrs["location"] = bucket.Location
		}
		if bucket.StorageClass != "" {
			attrs["storage_class"] = bucket.StorageClass
		}

		region := bucket.Location
		if strings.HasPrefix(region, "US") || strings.HasPrefix(region, "EU") {
			region = strings.ToLower(region)
		}

		resources = append(resources, models.Resource{
			ID:         bucket.Name,
			Type:       "google_storage_bucket",
			Provider:   models.ProviderGCP,
			Name:       bucket.Name,
			Region:     region,
			Attributes: attrs,
			Tags:       bucket.Labels,
			Metadata:   map[string]string{"project": projectID},
		})
	}
	return resources, nil
}

func defaultProjectID() (string, error) {
	if id := os.Getenv("GOOGLE_CLOUD_PROJECT"); id != "" {
		return id, nil
	}
	if id := os.Getenv("GCP_PROJECT"); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT or GCP_PROJECT environment variable is required")
}
