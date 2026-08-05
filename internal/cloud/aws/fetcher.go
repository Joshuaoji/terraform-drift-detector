package aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Fetcher retrieves AWS resources via cloud APIs.
type Fetcher struct {
	s3Fetcher  *S3Fetcher
	ec2Fetcher *EC2Fetcher
	iamFetcher *IAMFetcher
}

// NewFetcher creates an AWS cloud fetcher using default credential chain.
func NewFetcher() (*Fetcher, error) {
	s3f, err := NewS3Fetcher()
	if err != nil {
		return nil, fmt.Errorf("init s3 fetcher: %w", err)
	}
	ec2f, err := NewEC2Fetcher()
	if err != nil {
		return nil, fmt.Errorf("init ec2 fetcher: %w", err)
	}
	iamf, err := NewIAMFetcher()
	if err != nil {
		return nil, fmt.Errorf("init iam fetcher: %w", err)
	}
	return &Fetcher{
		s3Fetcher:  s3f,
		ec2Fetcher: ec2f,
		iamFetcher: iamf,
	}, nil
}

// Provider returns the AWS provider identifier.
func (f *Fetcher) Provider() models.Provider {
	return models.ProviderAWS
}

// SupportedTypes lists resource types this fetcher can retrieve.
func (f *Fetcher) SupportedTypes() []string {
	return []string{"aws_s3_bucket", "aws_instance", "aws_iam_role"}
}

// Fetch retrieves AWS resources matching the filter.
func (f *Fetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	types := filter.ResourceTypes
	if len(types) == 0 {
		types = f.SupportedTypes()
	}

	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}

	var (
		mu        sync.Mutex
		resources []models.Resource
		errs      []error
		wg        sync.WaitGroup
	)

	fetch := func(fn func(context.Context, cloud.FetchFilter) ([]models.Resource, error), resourceType string) {
		if !typeSet[resourceType] {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := fn(ctx, filter)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", resourceType, err))
				return
			}
			resources = append(resources, res...)
		}()
	}

	fetch(f.s3Fetcher.Fetch, "aws_s3_bucket")
	fetch(f.ec2Fetcher.Fetch, "aws_instance")
	fetch(f.iamFetcher.Fetch, "aws_iam_role")

	wg.Wait()

	if len(errs) > 0 {
		return resources, fmt.Errorf("partial fetch errors: %v", errs)
	}
	return resources, nil
}
