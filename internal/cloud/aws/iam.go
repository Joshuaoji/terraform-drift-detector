package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// IAMFetcher lists IAM roles.
type IAMFetcher struct {
	client *iam.Client
}

// NewIAMFetcher creates an IAM resource fetcher.
func NewIAMFetcher() (*IAMFetcher, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	return &IAMFetcher{client: iam.NewFromConfig(cfg)}, nil
}

// Fetch retrieves IAM roles.
func (f *IAMFetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	_ = filter
	var resources []models.Resource

	paginator := iam.NewListRolesPaginator(f.client, &iam.ListRolesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list roles: %w", err)
		}
		for _, role := range page.Roles {
			if role.RoleName == nil {
				continue
			}
			tags := f.getRoleTags(ctx, role.RoleName)
			attrs := map[string]any{
				"name": aws.ToString(role.RoleName),
				"path": aws.ToString(role.Path),
			}
			if role.Description != nil {
				attrs["description"] = *role.Description
			}
			if role.MaxSessionDuration != nil {
				attrs["max_session_duration"] = int(*role.MaxSessionDuration)
			}

			resources = append(resources, models.Resource{
				ID:         aws.ToString(role.RoleName),
				Type:       "aws_iam_role",
				Provider:   models.ProviderAWS,
				Name:       aws.ToString(role.RoleName),
				Region:     "global",
				Attributes: attrs,
				Tags:       tags,
				Metadata:   map[string]string{"arn": aws.ToString(role.Arn)},
			})
		}
	}
	return resources, nil
}

func (f *IAMFetcher) getRoleTags(ctx context.Context, roleName *string) map[string]string {
	tags := map[string]string{}
	out, err := f.client.ListRoleTags(ctx, &iam.ListRoleTagsInput{RoleName: roleName})
	if err != nil {
		return tags
	}
	for _, t := range out.Tags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags
}
