package state

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
	"google.golang.org/api/option"
)

func readGCS(ctx context.Context, parser *Parser, source models.StateSource) ([]models.Resource, error) {
	if source.Bucket == "" {
		return nil, fmt.Errorf("gcs state requires bucket")
	}
	key := source.Key
	if key == "" {
		key = source.Prefix
	}
	if key == "" {
		return nil, fmt.Errorf("gcs state requires key or prefix")
	}

	client, err := storage.NewClient(ctx, option.WithScopes(storage.ScopeReadOnly))
	if err != nil {
		return nil, fmt.Errorf("create GCS client: %w", err)
	}
	defer client.Close()

	reader, err := client.Bucket(source.Bucket).Object(key).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("read gcs object gs://%s/%s: %w", source.Bucket, key, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read gcs object body: %w", err)
	}
	return parser.Parse(data)
}
