package models

import (
	"fmt"
	"strings"
)

// StateSource describes where Terraform state is stored.
type StateSource struct {
	Backend        string `json:"backend,omitempty"` // local, s3, gcs, azure
	Path           string `json:"path,omitempty"`
	Bucket         string `json:"bucket,omitempty"`
	Key            string `json:"key,omitempty"`
	Prefix         string `json:"prefix,omitempty"`
	ResourceGroup  string `json:"resource_group,omitempty"`
	StorageAccount string `json:"storage_account,omitempty"`
	Container      string `json:"container,omitempty"`
}

// Display returns a human-readable state source description.
func (s StateSource) Display() string {
	switch s.Backend {
	case "s3":
		return fmt.Sprintf("s3://%s/%s", s.Bucket, s.Key)
	case "gcs":
		key := s.Key
		if key == "" {
			key = s.Prefix
		}
		return fmt.Sprintf("gcs://%s/%s", s.Bucket, key)
	case "azure", "azurerm":
		return fmt.Sprintf("azure://%s/%s/%s", s.StorageAccount, s.Container, s.Key)
	case "local", "":
		return s.Path
	default:
		return s.Path
	}
}

// ParseStatePath infers a StateSource from a path or URI string.
func ParseStatePath(path string) StateSource {
	if strings.HasPrefix(path, "s3://") {
		rest := strings.TrimPrefix(path, "s3://")
		parts := strings.SplitN(rest, "/", 2)
		key := ""
		if len(parts) == 2 {
			key = parts[1]
		}
		return StateSource{Backend: "s3", Bucket: parts[0], Key: key, Path: path}
	}
	if strings.HasPrefix(path, "gcs://") {
		rest := strings.TrimPrefix(path, "gcs://")
		parts := strings.SplitN(rest, "/", 2)
		key := ""
		if len(parts) == 2 {
			key = parts[1]
		}
		return StateSource{Backend: "gcs", Bucket: parts[0], Key: key, Path: path}
	}
	if strings.HasPrefix(path, "azure://") {
		rest := strings.TrimPrefix(path, "azure://")
		parts := strings.SplitN(rest, "/", 3)
		src := StateSource{Backend: "azure", Path: path}
		if len(parts) >= 1 {
			src.StorageAccount = parts[0]
		}
		if len(parts) >= 2 {
			src.Container = parts[1]
		}
		if len(parts) >= 3 {
			src.Key = parts[2]
		}
		return src
	}
	return StateSource{Backend: "local", Path: path}
}

// ResolvedStateSource returns the effective state source for scan options.
func (o ScanOptions) ResolvedStateSource() StateSource {
	if o.StateSource.Backend != "" || o.StateSource.Bucket != "" || o.StateSource.StorageAccount != "" {
		return o.StateSource
	}
	if o.StatePath != "" {
		return ParseStatePath(o.StatePath)
	}
	return StateSource{}
}
