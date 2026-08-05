package azurecloud

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// ResourceGroupFetcher lists Azure resource groups.
type ResourceGroupFetcher struct {
	client *armresources.ResourceGroupsClient
}

// NewResourceGroupFetcher creates a resource group fetcher.
func NewResourceGroupFetcher() (*ResourceGroupFetcher, error) {
	cred, err := defaultCredential()
	if err != nil {
		return nil, err
	}
	subscriptionID, err := defaultSubscriptionID()
	if err != nil {
		return nil, err
	}
	client, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("resource groups client: %w", err)
	}
	return &ResourceGroupFetcher{client: client}, nil
}

// Fetch retrieves resource groups.
func (f *ResourceGroupFetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	_ = filter
	var resources []models.Resource
	pager := f.client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list resource groups: %w", err)
		}
		for _, rg := range page.Value {
			if rg.Name == nil || rg.ID == nil {
				continue
			}
			region := ""
			if rg.Location != nil {
				region = *rg.Location
			}
			tags := map[string]string{}
			for k, v := range rg.Tags {
				if v != nil {
					tags[k] = *v
				}
			}
			resources = append(resources, models.Resource{
				ID:         *rg.Name,
				Type:       "azurerm_resource_group",
				Provider:   models.ProviderAzure,
				Name:       *rg.Name,
				Region:     region,
				Attributes: map[string]any{"name": *rg.Name, "location": region},
				Tags:       tags,
				Metadata:   map[string]string{"azure_id": *rg.ID},
			})
		}
	}
	return resources, nil
}
