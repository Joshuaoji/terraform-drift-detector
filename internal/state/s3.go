package state

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

func readS3(ctx context.Context, parser *Parser, source models.StateSource) ([]models.Resource, error) {
	if source.Bucket == "" || source.Key == "" {
		return nil, fmt.Errorf("s3 state requires bucket and key")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	client := s3.NewFromConfig(cfg)

	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(source.Bucket),
		Key:    aws.String(source.Key),
	})
	if err != nil {
		return nil, fmt.Errorf("get s3 object s3://%s/%s: %w", source.Bucket, source.Key, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read s3 object body: %w", err)
	}
	return parser.Parse(data)
}
