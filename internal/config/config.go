package config

import (
	"fmt"
	"os"

	"github.com/terraform-drift-detector/driftdetect/pkg/models"
	"gopkg.in/yaml.v3"
)

// StateConfig describes where Terraform state is stored.
type StateConfig struct {
	Backend string `yaml:"backend"` // local, s3, gcs, azure
	Path    string `yaml:"path,omitempty"`
	Bucket  string `yaml:"bucket,omitempty"`
	Key     string `yaml:"key,omitempty"`
	Prefix  string `yaml:"prefix,omitempty"`
}

// ScanProfile defines a named scan configuration.
type ScanProfile struct {
	Name          string         `yaml:"name"`
	Provider      models.Provider `yaml:"provider"`
	Regions       []string       `yaml:"regions,omitempty"`
	ResourceTypes []string       `yaml:"resource_types,omitempty"`
	State         StateConfig    `yaml:"state"`
	Schedule      string         `yaml:"schedule,omitempty"`
	AccountID     string         `yaml:"account_id,omitempty"`
	Project       string         `yaml:"project,omitempty"`
}

// Config is the root application configuration.
type Config struct {
	Scans []ScanProfile `yaml:"scans"`
}

// Load reads configuration from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// FindScan returns a scan profile by name.
func (c *Config) FindScan(name string) (*ScanProfile, error) {
	for i := range c.Scans {
		if c.Scans[i].Name == name {
			return &c.Scans[i], nil
		}
	}
	return nil, fmt.Errorf("scan profile %q not found", name)
}

// ToScanOptions converts a profile to scan options.
func (p *ScanProfile) ToScanOptions() models.ScanOptions {
	statePath := p.State.Path
	switch p.State.Backend {
	case "s3":
		if p.State.Bucket != "" {
			statePath = fmt.Sprintf("s3://%s/%s", p.State.Bucket, p.State.Key)
		}
	case "gcs":
		if p.State.Bucket != "" {
			statePath = fmt.Sprintf("gcs://%s/%s", p.State.Bucket, p.State.Prefix)
		}
	}
	return models.ScanOptions{
		StatePath:     statePath,
		Provider:      p.Provider,
		Regions:       p.Regions,
		ResourceTypes: p.ResourceTypes,
		AccountID:     p.AccountID,
	}
}
