package state_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/terraform-drift-detector/driftdetect/internal/state"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

func TestLocalReader_RawStateV4(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.tfstate")
	reader := state.NewReader()
	resources, err := reader.Read(context.Background(), models.StateSource{Backend: "local", Path: path})
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
	reader := state.NewReader()
	_, err := reader.Read(context.Background(), models.StateSource{Backend: "local", Path: "/nonexistent/path.tfstate"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParser_RawStateV4(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.tfstate")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parser := state.NewParser()
	resources, err := parser.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) < 3 {
		t.Fatalf("expected at least 3 resources, got %d", len(resources))
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
