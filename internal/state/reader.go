package state

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Reader loads and parses Terraform state.
type Reader interface {
	Read(ctx context.Context, source models.StateSource) ([]models.Resource, error)
}

// CompositeReader routes to the correct backend reader.
type CompositeReader struct {
	parser *Parser
}

// NewReader creates a reader that supports local and remote state backends.
func NewReader() *CompositeReader {
	return &CompositeReader{parser: NewParser()}
}

// Read fetches and parses state from the given source.
func (r *CompositeReader) Read(ctx context.Context, source models.StateSource) ([]models.Resource, error) {
	backend := source.Backend
	if backend == "" {
		backend = "local"
	}

	switch backend {
	case "local":
		return r.readLocal(ctx, source)
	case "s3":
		return readS3(ctx, r.parser, source)
	case "gcs":
		return readGCS(ctx, r.parser, source)
	case "azure", "azurerm":
		return readAzureBlob(ctx, r.parser, source)
	default:
		if strings.HasPrefix(source.Path, "s3://") {
			return readS3(ctx, r.parser, models.ParseStatePath(source.Path))
		}
		if strings.HasPrefix(source.Path, "gcs://") {
			return readGCS(ctx, r.parser, models.ParseStatePath(source.Path))
		}
		if strings.HasPrefix(source.Path, "azure://") {
			return readAzureBlob(ctx, r.parser, models.ParseStatePath(source.Path))
		}
		return nil, fmt.Errorf("unsupported state backend: %s", backend)
	}
}

func (r *CompositeReader) readLocal(ctx context.Context, source models.StateSource) ([]models.Resource, error) {
	_ = ctx
	path := source.Path
	if path == "" {
		return nil, fmt.Errorf("local state path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	return r.parser.Parse(data)
}

// LocalReader reads state from a local file (backward compatible wrapper).
type LocalReader struct {
	inner *CompositeReader
}

// NewLocalReader creates a reader for local state files.
func NewLocalReader() *LocalReader {
	return &LocalReader{inner: NewReader()}
}

// Read parses a local terraform.tfstate file.
func (r *LocalReader) Read(ctx context.Context, source models.StateSource) ([]models.Resource, error) {
	if source.Backend == "" && source.Path != "" {
		source.Backend = "local"
	}
	return r.inner.Read(ctx, source)
}
