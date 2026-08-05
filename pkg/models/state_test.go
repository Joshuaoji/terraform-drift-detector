package models_test

import (
	"testing"

	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

func TestParseStatePath_S3(t *testing.T) {
	src := models.ParseStatePath("s3://my-bucket/prod/terraform.tfstate")
	if src.Backend != "s3" || src.Bucket != "my-bucket" || src.Key != "prod/terraform.tfstate" {
		t.Fatalf("unexpected s3 source: %+v", src)
	}
}

func TestParseStatePath_GCS(t *testing.T) {
	src := models.ParseStatePath("gcs://tf-state/prod/default.tfstate")
	if src.Backend != "gcs" || src.Bucket != "tf-state" {
		t.Fatalf("unexpected gcs source: %+v", src)
	}
}

func TestParseStatePath_Azure(t *testing.T) {
	src := models.ParseStatePath("azure://mystorageaccount/tfstate/prod.terraform.tfstate")
	if src.Backend != "azure" || src.StorageAccount != "mystorageaccount" || src.Container != "tfstate" {
		t.Fatalf("unexpected azure source: %+v", src)
	}
}

func TestParseStatePath_Local(t *testing.T) {
	src := models.ParseStatePath("./terraform.tfstate")
	if src.Backend != "local" || src.Path != "./terraform.tfstate" {
		t.Fatalf("unexpected local source: %+v", src)
	}
}

func TestResolvedStateSource(t *testing.T) {
	opts := models.ScanOptions{StatePath: "s3://bucket/key"}
	src := opts.ResolvedStateSource()
	if src.Backend != "s3" {
		t.Fatalf("expected s3 backend, got %s", src.Backend)
	}
}
