package drift_test

import (
	"testing"
	"time"

	"github.com/terraform-drift-detector/driftdetect/internal/drift"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

func fixedTime() time.Time {
	return time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
}

func TestEngine_MissingInCloud(t *testing.T) {
	engine := drift.NewEngine()
	expected := []models.Resource{
		{ID: "my-bucket", Type: "aws_s3_bucket", Provider: models.ProviderAWS, Region: "us-east-1"},
	}
	actual := []models.Resource{}

	report := engine.Compare("test.tfstate", models.ProviderAWS, expected, actual)
	if report.Summary.MissingInCloud != 1 {
		t.Fatalf("expected 1 missing in cloud, got %d", report.Summary.MissingInCloud)
	}
	if report.Drifts[0].Type != models.DriftMissingInCloud {
		t.Fatalf("expected missing_in_cloud drift type")
	}
}

func TestEngine_MissingInState(t *testing.T) {
	engine := drift.NewEngine()
	expected := []models.Resource{}
	actual := []models.Resource{
		{ID: "orphan-bucket", Type: "aws_s3_bucket", Provider: models.ProviderAWS, Region: "us-east-1"},
	}

	report := engine.Compare("test.tfstate", models.ProviderAWS, expected, actual)
	if report.Summary.MissingInState != 1 {
		t.Fatalf("expected 1 missing in state, got %d", report.Summary.MissingInState)
	}
}

func TestEngine_AttributeDrift(t *testing.T) {
	engine := drift.NewEngine()
	expected := []models.Resource{
		{
			ID: "i-123", Type: "aws_instance", Provider: models.ProviderAWS, Region: "us-east-1",
			Attributes: map[string]any{"instance_type": "t3.micro", "ami": "ami-old"},
		},
	}
	actual := []models.Resource{
		{
			ID: "i-123", Type: "aws_instance", Provider: models.ProviderAWS, Region: "us-east-1",
			Attributes: map[string]any{"instance_type": "t3.small", "ami": "ami-old"},
		},
	}

	report := engine.Compare("test.tfstate", models.ProviderAWS, expected, actual)
	if report.Summary.AttributeDrifts != 1 {
		t.Fatalf("expected 1 attribute drift, got %d", report.Summary.AttributeDrifts)
	}
	if len(report.Drifts[0].AttributeChanges) != 1 {
		t.Fatalf("expected 1 attribute change")
	}
	if report.Drifts[0].AttributeChanges[0].Attribute != "instance_type" {
		t.Fatalf("expected instance_type change")
	}
}

func TestEngine_TagDrift(t *testing.T) {
	engine := drift.NewEngine()
	expected := []models.Resource{
		{
			ID: "my-bucket", Type: "aws_s3_bucket", Provider: models.ProviderAWS, Region: "us-east-1",
			Tags: map[string]string{"env": "prod", "team": "platform"},
		},
	}
	actual := []models.Resource{
		{
			ID: "my-bucket", Type: "aws_s3_bucket", Provider: models.ProviderAWS, Region: "us-east-1",
			Tags: map[string]string{"env": "staging", "owner": "devops"},
		},
	}

	report := engine.Compare("test.tfstate", models.ProviderAWS, expected, actual)
	if report.Summary.TagDrifts != 1 {
		t.Fatalf("expected 1 tag drift, got %d", report.Summary.TagDrifts)
	}
	if len(report.Drifts[0].TagChanges) < 2 {
		t.Fatalf("expected multiple tag changes, got %d", len(report.Drifts[0].TagChanges))
	}
}

func TestEngine_NoDrift(t *testing.T) {
	engine := drift.NewEngine()
	resource := models.Resource{
		ID: "my-bucket", Type: "aws_s3_bucket", Provider: models.ProviderAWS, Region: "us-east-1",
		Attributes: map[string]any{"bucket": "my-bucket"},
		Tags:       map[string]string{"env": "prod"},
	}
	report := engine.Compare("test.tfstate", models.ProviderAWS, []models.Resource{resource}, []models.Resource{resource})
	if report.Summary.TotalDrifts != 0 {
		t.Fatalf("expected no drift, got %d", report.Summary.TotalDrifts)
	}
}

func TestEngine_ARNMatching(t *testing.T) {
	engine := drift.NewEngine()
	expected := []models.Resource{
		{ID: "arn:aws:s3:::my-bucket", Type: "aws_s3_bucket", Provider: models.ProviderAWS, Region: "us-east-1"},
	}
	actual := []models.Resource{
		{ID: "my-bucket", Type: "aws_s3_bucket", Provider: models.ProviderAWS, Region: "us-east-1"},
	}
	report := engine.Compare("test.tfstate", models.ProviderAWS, expected, actual)
	if report.Summary.TotalDrifts != 0 {
		t.Fatalf("expected ARN/id match with no drift, got %d drifts", report.Summary.TotalDrifts)
	}
	_ = fixedTime
}
