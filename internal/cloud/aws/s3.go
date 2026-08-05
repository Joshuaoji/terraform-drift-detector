package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// S3Fetcher lists S3 buckets.
type S3Fetcher struct {
	client *s3.Client
}

// NewS3Fetcher creates an S3 resource fetcher.
func NewS3Fetcher() (*S3Fetcher, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	return &S3Fetcher{client: s3.NewFromConfig(cfg)}, nil
}

// Fetch retrieves S3 buckets.
func (f *S3Fetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	out, err := f.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("list buckets: %w", err)
	}

	var resources []models.Resource
	for _, bucket := range out.Buckets {
		name := *bucket.Name
		region := "us-east-1"
		loc, err := f.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{Bucket: bucket.Name})
		if err == nil && loc.LocationConstraint != "" {
			region = string(loc.LocationConstraint)
		}

		if len(filter.Regions) > 0 && !contains(filter.Regions, region) {
			continue
		}

		tags := f.getBucketTags(ctx, bucket.Name)
		attrs := map[string]any{
			"bucket": name,
		}

		resources = append(resources, models.Resource{
			ID:         name,
			Type:       "aws_s3_bucket",
			Provider:   models.ProviderAWS,
			Name:       name,
			Region:     region,
			Attributes: attrs,
			Tags:       tags,
			Metadata:   map[string]string{"arn": fmt.Sprintf("arn:aws:s3:::%s", name)},
		})
	}

	return resources, nil
}

func (f *S3Fetcher) getBucketTags(ctx context.Context, bucket *string) map[string]string {
	tags := map[string]string{}
	out, err := f.client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: bucket})
	if err != nil {
		return tags
	}
	for _, t := range out.TagSet {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
