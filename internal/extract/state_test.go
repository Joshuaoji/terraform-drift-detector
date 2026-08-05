package extract_test

import (
	"testing"

	"github.com/terraform-drift-detector/driftdetect/internal/extract"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

func TestStateExtractor_AWSS3Bucket(t *testing.T) {
	e := extract.NewStateExtractor()
	attrs := map[string]any{
		"id":     "my-logs-bucket",
		"bucket": "my-logs-bucket",
		"arn":    "arn:aws:s3:::my-logs-bucket",
		"region": "us-east-1",
		"tags": map[string]any{
			"env": "prod",
		},
	}
	res, err := e.Extract("aws_s3_bucket", "aws_s3_bucket.logs", attrs)
	if err != nil {
		t.Fatal(err)
	}
	if res.ID != "my-logs-bucket" {
		t.Fatalf("expected id my-logs-bucket, got %s", res.ID)
	}
	if res.Tags["env"] != "prod" {
		t.Fatalf("expected env=prod tag")
	}
}

func TestStateExtractor_AWSInstance(t *testing.T) {
	e := extract.NewStateExtractor()
	attrs := map[string]any{
		"id":                "i-abc123",
		"instance_type":     "t3.micro",
		"ami":               "ami-12345",
		"availability_zone": "us-east-1a",
		"tags": map[string]any{
			"Name": "web-server",
		},
	}
	res, err := e.Extract("aws_instance", "aws_instance.web", attrs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Region != "us-east-1" {
		t.Fatalf("expected region us-east-1, got %s", res.Region)
	}
	if res.Name != "web-server" {
		t.Fatalf("expected name web-server, got %s", res.Name)
	}
}

func TestFilterByTypes(t *testing.T) {
	resources := []models.Resource{
		{Type: "aws_s3_bucket"},
		{Type: "aws_instance"},
	}
	filtered := extract.FilterByTypes(resources, []string{"aws_s3_bucket"})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(filtered))
	}
}
