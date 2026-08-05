package gcpcloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// ComputeFetcher lists GCP compute instances.
type ComputeFetcher struct {
	service   *compute.Service
	projectID string
}

// NewComputeFetcher creates a GCP compute instance fetcher.
func NewComputeFetcher() (*ComputeFetcher, error) {
	projectID, err := defaultProjectID()
	if err != nil {
		return nil, err
	}
	service, err := compute.NewService(context.Background(), option.WithScopes(compute.ComputeReadonlyScope))
	if err != nil {
		return nil, fmt.Errorf("create compute service: %w", err)
	}
	return &ComputeFetcher{service: service, projectID: projectID}, nil
}

// Fetch retrieves compute instances across zones.
func (f *ComputeFetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	projectID := f.projectID
	if filter.AccountID != "" {
		projectID = filter.AccountID
	}
	if filter.ProjectID != "" {
		projectID = filter.ProjectID
	}

	var resources []models.Resource
	zones := filter.Regions
	if len(zones) == 0 {
		zoneList, err := f.service.Zones.List(projectID).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("list zones: %w", err)
		}
		for _, z := range zoneList.Items {
			zones = append(zones, z.Name)
		}
	}

	for _, zone := range zones {
		resp, err := f.service.Instances.List(projectID, zone).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("list instances in %s: %w", zone, err)
		}
		for _, inst := range resp.Items {
			attrs := map[string]any{
				"name":         inst.Name,
				"machine_type": machineTypeName(inst.MachineType),
				"zone":         zone,
			}
			labels := map[string]string{}
			if inst.Labels != nil {
				labels = inst.Labels
			}

			resources = append(resources, models.Resource{
				ID:         fmt.Sprintf("%d", inst.Id),
				Type:       "google_compute_instance",
				Provider:   models.ProviderGCP,
				Name:       inst.Name,
				Region:     zone,
				Attributes: attrs,
				Tags:       labels,
				Metadata: map[string]string{
					"self_link": inst.SelfLink,
					"project":   projectID,
				},
			})
		}
	}
	return resources, nil
}

func machineTypeName(selfLink string) string {
	if selfLink == "" {
		return ""
	}
	if i := strings.LastIndex(selfLink, "/"); i >= 0 {
		return selfLink[i+1:]
	}
	return selfLink
}
