package state_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/terraform-drift-detector/driftdetect/internal/state"
)

func TestLocalReader_RawStateV4(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.tfstate")
	reader := state.NewLocalReader()
	resources, err := reader.Read(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) < 2 {
		t.Fatalf("expected at least 2 resources, got %d", len(resources))
	}

	types := map[string]bool{}
	for _, r := range resources {
		types[r.Type] = true
	}
	if !types["aws_s3_bucket"] {
		t.Fatal("expected aws_s3_bucket in state")
	}
	if !types["aws_instance"] {
		t.Fatal("expected aws_instance in state")
	}
}

func TestLocalReader_FileNotFound(t *testing.T) {
	reader := state.NewLocalReader()
	_, err := reader.Read(context.Background(), "/nonexistent/path.tfstate")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
