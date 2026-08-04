package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/terraform-drift-detector/driftdetect/internal/cloud"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// EC2Fetcher lists EC2 instances.
type EC2Fetcher struct {
	clients map[string]*ec2.Client
}

// NewEC2Fetcher creates an EC2 resource fetcher.
func NewEC2Fetcher() (*EC2Fetcher, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	return &EC2Fetcher{
		clients: map[string]*ec2.Client{
			cfg.Region: ec2.NewFromConfig(cfg),
		},
	}, nil
}

func (f *EC2Fetcher) clientForRegion(ctx context.Context, region string) (*ec2.Client, error) {
	if c, ok := f.clients[region]; ok {
		return c, nil
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	c := ec2.NewFromConfig(cfg)
	f.clients[region] = c
	return c, nil
}

// Fetch retrieves EC2 instances.
func (f *EC2Fetcher) Fetch(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	regions := filter.Regions
	if len(regions) == 0 {
		cfg, _ := config.LoadDefaultConfig(ctx)
		regions = []string{cfg.Region}
	}

	var resources []models.Resource
	for _, region := range regions {
		client, err := f.clientForRegion(ctx, region)
		if err != nil {
			return nil, fmt.Errorf("ec2 client %s: %w", region, err)
		}

		paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("describe instances %s: %w", region, err)
			}
			for _, reservation := range page.Reservations {
				for _, inst := range reservation.Instances {
					if inst.InstanceId == nil {
						continue
					}
					if inst.State != nil && inst.State.Name == ec2types.InstanceStateNameTerminated {
						continue
					}
					resources = append(resources, instanceToResource(inst, region))
				}
			}
		}
	}
	return resources, nil
}

func instanceToResource(inst ec2types.Instance, region string) models.Resource {
	tags := map[string]string{}
	name := ""
	for _, t := range inst.Tags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
			if *t.Key == "Name" {
				name = *t.Value
			}
		}
	}

	attrs := map[string]any{
		"instance_type": string(inst.InstanceType),
	}
	if inst.ImageId != nil {
		attrs["ami"] = *inst.ImageId
	}
	if inst.Placement != nil && inst.Placement.AvailabilityZone != nil {
		attrs["availability_zone"] = *inst.Placement.AvailabilityZone
	}
	if inst.Monitoring != nil {
		attrs["monitoring"] = inst.Monitoring.State == ec2types.MonitoringStateEnabled
	}

	arn := ""
	if inst.InstanceId != nil {
		arn = fmt.Sprintf("arn:aws:ec2:%s::instance/%s", region, *inst.InstanceId)
	}

	return models.Resource{
		ID:         aws.ToString(inst.InstanceId),
		Type:       "aws_instance",
		Provider:   models.ProviderAWS,
		Name:       name,
		Region:     region,
		Attributes: attrs,
		Tags:       tags,
		Metadata:   map[string]string{"arn": arn},
	}
}
