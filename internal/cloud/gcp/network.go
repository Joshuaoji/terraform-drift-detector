package gcpcloud

import (
	"context"
	"fmt"

	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// NetworkFetcher lists GCP VPC networks.
type NetworkFetcher struct {
	service   *compute.Service
	projectID string
}

// NewNetworkFetcher creates a GCP network fetcher.
func NewNetworkFetcher() (*NetworkFetcher, error) {
	projectID, err := defaultProjectID()
	if err != nil {
		return nil, err
	}
	service, err := compute.NewService(context.Background(), option.WithScopes(compute.ComputeReadonlyScope))
	if err != nil {
		return nil, fmt.Errorf("create compute service: %w", err)
	}
	return &NetworkFetcher{service: service, projectID: projectID}, nil
}

// Fetch retrieves VPC networks.
func (f *NetworkFetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	projectID := f.projectID
	if filter.ProjectID != "" {
		projectID = filter.ProjectID
	}
	resp, err := f.service.Networks.List(projectID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	var resources []models.Resource
	for _, net := range resp.Items {
		attrs := map[string]any{
			"name":                    net.Name,
			"auto_create_subnetworks": net.AutoCreateSubnetworks,
		}
		if net.RoutingConfig != nil {
			attrs["routing_mode"] = net.RoutingConfig.RoutingMode
		}
		resources = append(resources, models.Resource{
			ID:         fmt.Sprintf("%d", net.Id),
			Type:       "google_compute_network",
			Provider:   models.ProviderGCP,
			Name:       net.Name,
			Region:     "global",
			Attributes: attrs,
			Metadata: map[string]string{
				"self_link": net.SelfLink,
				"project":   projectID,
			},
		})
	}
	return resources, nil
}
