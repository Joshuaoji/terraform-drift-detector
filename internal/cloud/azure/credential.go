package azurecloud

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func defaultSubscriptionID() (string, error) {
	if id := os.Getenv("AZURE_SUBSCRIPTION_ID"); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("AZURE_SUBSCRIPTION_ID environment variable is required")
}

func defaultCredential() (*azidentity.DefaultAzureCredential, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("azure credential: %w", err)
	}
	return cred, nil
}
