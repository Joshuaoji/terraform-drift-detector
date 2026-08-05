package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/terraform-drift-detector/driftdetect/pkg/models"
	"gopkg.in/yaml.v3"
)

// StateConfig describes where Terraform state is stored.
type StateConfig struct {
	Backend        string `yaml:"backend"` // local, s3, gcs, azure
	Path           string `yaml:"path,omitempty"`
	Bucket         string `yaml:"bucket,omitempty"`
	Key            string `yaml:"key,omitempty"`
	Prefix         string `yaml:"prefix,omitempty"`
	ResourceGroup  string `yaml:"resource_group,omitempty"`
	StorageAccount string `yaml:"storage_account,omitempty"`
	Container      string `yaml:"container,omitempty"`
}

// ScanProfile defines a named scan configuration.
type ScanProfile struct {
	Name          string          `yaml:"name"`
	Provider      models.Provider `yaml:"provider"`
	Regions       []string        `yaml:"regions,omitempty"`
	ResourceTypes []string        `yaml:"resource_types,omitempty"`
	State         StateConfig     `yaml:"state"`
	Schedule      string          `yaml:"schedule,omitempty"`
	AccountID     string          `yaml:"account_id,omitempty"`
	Project       string          `yaml:"project,omitempty"`
}

// Config is the root application configuration.
type Config struct {
	Server   ServerConfig    `yaml:"server,omitempty"`
	Scans    []ScanProfile   `yaml:"scans"`
	Webhooks []WebhookConfig `yaml:"webhooks,omitempty"`
}

// ServerConfig holds API server settings.
type ServerConfig struct {
	APIKeys []string `yaml:"api_keys,omitempty"`
}

// WebhookConfig defines an outbound webhook.
type WebhookConfig struct {
	Name   string   `yaml:"name"`
	URL    string   `yaml:"url"`
	Events []string `yaml:"events,omitempty"`
	Secret string   `yaml:"secret,omitempty"`
}

// APIKeys returns configured API keys, including DRIFTDETECT_API_KEYS env (comma-separated).
func (c *Config) APIKeys() []string {
	var keys []string
	if c != nil {
		keys = append(keys, c.Server.APIKeys...)
	}
	if env := os.Getenv("DRIFTDETECT_API_KEYS"); env != "" {
		for _, k := range strings.Split(env, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				keys = append(keys, k)
			}
		}
	}
	return keys
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
	src := models.StateSource{Backend: p.State.Backend}
	switch p.State.Backend {
	case "s3":
		src.Bucket = p.State.Bucket
		src.Key = p.State.Key
		src.Path = fmt.Sprintf("s3://%s/%s", p.State.Bucket, p.State.Key)
	case "gcs":
		src.Bucket = p.State.Bucket
		src.Key = p.State.Key
		if src.Key == "" {
			src.Key = p.State.Prefix
			src.Prefix = p.State.Prefix
		}
		src.Path = fmt.Sprintf("gcs://%s/%s", p.State.Bucket, src.Key)
	case "azure", "azurerm":
		src.StorageAccount = p.State.StorageAccount
		src.Container = p.State.Container
		src.Key = p.State.Key
		src.ResourceGroup = p.State.ResourceGroup
		src.Path = fmt.Sprintf("azure://%s/%s/%s", p.State.StorageAccount, p.State.Container, p.State.Key)
	default:
		src.Backend = "local"
		src.Path = p.State.Path
	}
	return models.ScanOptions{
		StateSource:   src,
		StatePath:     src.Path,
		Provider:      p.Provider,
		Regions:       p.Regions,
		ResourceTypes: p.ResourceTypes,
		AccountID:     p.AccountID,
		ProjectID:     p.Project,
	}
}
