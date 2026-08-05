package azurecloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// StorageFetcher lists Azure storage accounts.
type StorageFetcher struct {
	accountsClient *armstorage.AccountsClient
	subscriptionID string
}

// NewStorageFetcher creates an Azure storage account fetcher.
func NewStorageFetcher() (*StorageFetcher, error) {
	cred, err := defaultCredential()
	if err != nil {
		return nil, err
	}
	subscriptionID, err := defaultSubscriptionID()
	if err != nil {
		return nil, err
	}
	accountsClient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("storage accounts client: %w", err)
	}
	return &StorageFetcher{
		accountsClient: accountsClient,
		subscriptionID: subscriptionID,
	}, nil
}

// Fetch retrieves storage accounts across resource groups.
func (f *StorageFetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	_ = filter
	var resources []models.Resource

	pager := f.accountsClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list storage accounts: %w", err)
		}
		for _, account := range page.Value {
			if account.Name == nil || account.ID == nil {
				continue
			}
			region := ""
			if account.Location != nil {
				region = *account.Location
			}
			attrs := map[string]any{
				"name": *account.Name,
			}
			if account.SKU != nil && account.SKU.Name != nil {
				attrs["account_tier"] = tierFromSKU(*account.SKU.Name)
				attrs["account_replication_type"] = replicationFromSKU(*account.SKU.Name)
			}
			if account.Kind != nil {
				attrs["account_kind"] = string(*account.Kind)
			}

			tags := map[string]string{}
			for k, v := range account.Tags {
				if v != nil {
					tags[k] = *v
				}
			}

			resources = append(resources, models.Resource{
				ID:         *account.Name,
				Type:       "azurerm_storage_account",
				Provider:   models.ProviderAzure,
				Name:       *account.Name,
				Region:     region,
				Attributes: attrs,
				Tags:       tags,
				Metadata:   map[string]string{"azure_id": *account.ID},
			})
		}
	}
	return resources, nil
}

func tierFromSKU(sku armstorage.SKUName) string {
	parts := strings.SplitN(string(sku), "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return string(sku)
}

func replicationFromSKU(sku armstorage.SKUName) string {
	parts := strings.SplitN(string(sku), "_", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return string(sku)
}
