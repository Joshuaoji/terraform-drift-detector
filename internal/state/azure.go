package state

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

func readAzureBlob(ctx context.Context, parser *Parser, source models.StateSource) ([]models.Resource, error) {
	if source.StorageAccount == "" || source.Container == "" || source.Key == "" {
		return nil, fmt.Errorf("azure state requires storage_account, container, and key")
	}

	accountURL := fmt.Sprintf("https://%s.blob.core.windows.net", source.StorageAccount)
	client, err := newAzureBlobClient(accountURL)
	if err != nil {
		return nil, err
	}

	resp, err := client.DownloadStream(ctx, source.Container, source.Key, nil)
	if err != nil {
		return nil, fmt.Errorf("download blob azure://%s/%s/%s: %w", source.StorageAccount, source.Container, source.Key, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read blob body: %w", err)
	}
	return parser.Parse(data)
}

func newAzureBlobClient(accountURL string) (*azblob.Client, error) {
	if connStr := os.Getenv("AZURE_STORAGE_CONNECTION_STRING"); connStr != "" {
		client, err := azblob.NewClientFromConnectionString(connStr, nil)
		if err != nil {
			return nil, fmt.Errorf("create blob client from connection string: %w", err)
		}
		return client, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("create Azure credential: %w", err)
	}
	client, err := azblob.NewClient(accountURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create blob client: %w", err)
	}
	return client, nil
}
