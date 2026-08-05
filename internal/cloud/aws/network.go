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

// NetworkFetcher lists VPCs and security groups.
type NetworkFetcher struct {
	clients map[string]*ec2.Client
}

// NewNetworkFetcher creates a VPC/security group fetcher.
func NewNetworkFetcher() (*NetworkFetcher, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	return &NetworkFetcher{
		clients: map[string]*ec2.Client{cfg.Region: ec2.NewFromConfig(cfg)},
	}, nil
}

func (f *NetworkFetcher) clientForRegion(ctx context.Context, region string) (*ec2.Client, error) {
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

// FetchVPCs retrieves VPC resources.
func (f *NetworkFetcher) FetchVPCs(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	regions := filter.Regions
	if len(regions) == 0 {
		cfg, _ := config.LoadDefaultConfig(ctx)
		regions = []string{cfg.Region}
	}
	var resources []models.Resource
	for _, region := range regions {
		client, err := f.clientForRegion(ctx, region)
		if err != nil {
			return nil, err
		}
		out, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
		if err != nil {
			return nil, fmt.Errorf("describe vpcs %s: %w", region, err)
		}
		for _, vpc := range out.Vpcs {
			if vpc.VpcId == nil {
				continue
			}
			tags := tagsToMap(vpc.Tags)
			attrs := map[string]any{
				"cidr_block":            aws.ToString(vpc.CidrBlock),
				"enable_dns_hostnames":  vpc.DhcpOptionsId != nil,
				"instance_tenancy":      string(vpc.InstanceTenancy),
			}
			resources = append(resources, models.Resource{
				ID:         aws.ToString(vpc.VpcId),
				Type:       "aws_vpc",
				Provider:   models.ProviderAWS,
				Name:       tags["Name"],
				Region:     region,
				Attributes: attrs,
				Tags:       tags,
			})
		}
	}
	return resources, nil
}

// FetchSecurityGroups retrieves security group resources.
func (f *NetworkFetcher) FetchSecurityGroups(ctx context.Context, filter cloud.FetchFilter) ([]models.Resource, error) {
	regions := filter.Regions
	if len(regions) == 0 {
		cfg, _ := config.LoadDefaultConfig(ctx)
		regions = []string{cfg.Region}
	}
	var resources []models.Resource
	for _, region := range regions {
		client, err := f.clientForRegion(ctx, region)
		if err != nil {
			return nil, err
		}
		out, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
		if err != nil {
			return nil, fmt.Errorf("describe security groups %s: %w", region, err)
		}
		for _, sg := range out.SecurityGroups {
			if sg.GroupId == nil {
				continue
			}
			tags := tagsToMap(sg.Tags)
			attrs := map[string]any{
				"name":        aws.ToString(sg.GroupName),
				"description": aws.ToString(sg.Description),
				"vpc_id":      aws.ToString(sg.VpcId),
			}
			resources = append(resources, models.Resource{
				ID:         aws.ToString(sg.GroupId),
				Type:       "aws_security_group",
				Provider:   models.ProviderAWS,
				Name:       aws.ToString(sg.GroupName),
				Region:     region,
				Attributes: attrs,
				Tags:       tags,
			})
		}
	}
	return resources, nil
}

func tagsToMap(tags []ec2types.Tag) map[string]string {
	out := map[string]string{}
	for _, t := range tags {
		if t.Key != nil && t.Value != nil {
			out[*t.Key] = *t.Value
		}
	}
	return out
}
