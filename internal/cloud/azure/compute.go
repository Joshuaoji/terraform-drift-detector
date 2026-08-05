package azurecloud

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// ComputeFetcher lists Azure Linux virtual machines.
type ComputeFetcher struct {
	client         *armcompute.VirtualMachinesClient
	subscriptionID string
}

// NewComputeFetcher creates an Azure VM fetcher.
func NewComputeFetcher() (*ComputeFetcher, error) {
	cred, err := defaultCredential()
	if err != nil {
		return nil, err
	}
	subscriptionID, err := defaultSubscriptionID()
	if err != nil {
		return nil, err
	}
	client, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("compute client: %w", err)
	}
	return &ComputeFetcher{client: client, subscriptionID: subscriptionID}, nil
}

// Fetch retrieves Linux VMs across resource groups.
func (f *ComputeFetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	_ = filter
	var resources []models.Resource

	pager := f.client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list virtual machines: %w", err)
		}
		for _, vm := range page.Value {
			if vm.Name == nil || vm.ID == nil {
				continue
			}
			region := ""
			if vm.Location != nil {
				region = *vm.Location
			}
			attrs := map[string]any{}
			if vm.Properties != nil && vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
				attrs["size"] = string(*vm.Properties.HardwareProfile.VMSize)
			}
			if vm.Zones != nil && len(vm.Zones) > 0 && vm.Zones[0] != nil {
				attrs["zone"] = *vm.Zones[0]
			}

			tags := map[string]string{}
			for k, v := range vm.Tags {
				if v != nil {
					tags[k] = *v
				}
			}

			resources = append(resources, models.Resource{
				ID:         *vm.Name,
				Type:       "azurerm_linux_virtual_machine",
				Provider:   models.ProviderAzure,
				Name:       *vm.Name,
				Region:     region,
				Attributes: attrs,
				Tags:       tags,
				Metadata:   map[string]string{"azure_id": *vm.ID},
			})
		}
	}
	return resources, nil
}
