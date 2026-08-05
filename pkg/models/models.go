package models

import "time"

// Provider identifies a cloud provider.
type Provider string

const (
	ProviderAWS   Provider = "aws"
	ProviderAzure Provider = "azure"
	ProviderGCP   Provider = "gcp"
)

// Resource is the normalized representation of a cloud resource.
type Resource struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Provider     Provider          `json:"provider"`
	Region       string            `json:"region,omitempty"`
	Name         string            `json:"name,omitempty"`
	Attributes   map[string]any    `json:"attributes,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	TerraformRef string            `json:"terraform_ref,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// DriftType categorizes a detected drift.
type DriftType string

const (
	DriftMissingInCloud DriftType = "missing_in_cloud"
	DriftMissingInState DriftType = "missing_in_state"
	DriftAttribute      DriftType = "attribute_drift"
	DriftTag            DriftType = "tag_drift"
)

// AttributeChange records a single attribute difference.
type AttributeChange struct {
	Attribute string `json:"attribute"`
	Expected  any    `json:"expected"`
	Actual    any    `json:"actual"`
}

// TagChange records a tag difference.
type TagChange struct {
	Key      string `json:"key"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Change   string `json:"change"` // added, removed, modified
}

// DriftItem is a single drift finding.
type DriftItem struct {
	Type             DriftType         `json:"type"`
	ResourceType     string            `json:"resource_type"`
	ResourceID       string            `json:"resource_id"`
	ResourceName     string            `json:"resource_name,omitempty"`
	TerraformRef     string            `json:"terraform_ref,omitempty"`
	Region           string            `json:"region,omitempty"`
	AttributeChanges []AttributeChange `json:"attribute_changes,omitempty"`
	TagChanges       []TagChange       `json:"tag_changes,omitempty"`
	Message          string            `json:"message,omitempty"`
}

// DriftSummary aggregates drift counts.
type DriftSummary struct {
	TotalDrifts      int `json:"total_drifts"`
	MissingInCloud   int `json:"missing_in_cloud"`
	MissingInState   int `json:"missing_in_state"`
	AttributeDrifts  int `json:"attribute_drifts"`
	TagDrifts        int `json:"tag_drifts"`
	ResourcesChecked int `json:"resources_checked"`
}

// DriftReport is the full output of a drift scan.
type DriftReport struct {
	ScanID      string       `json:"scan_id"`
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt time.Time    `json:"completed_at"`
	StateSource string       `json:"state_source"`
	Provider    Provider     `json:"provider"`
	Summary     DriftSummary `json:"summary"`
	Drifts      []DriftItem  `json:"drifts"`
}

// ScanStatus represents the lifecycle of a scan.
type ScanStatus string

const (
	ScanPending   ScanStatus = "pending"
	ScanRunning   ScanStatus = "running"
	ScanCompleted ScanStatus = "completed"
	ScanFailed    ScanStatus = "failed"
)

// ScanOptions configures a drift scan.
type ScanOptions struct {
	StatePath     string
	Provider      Provider
	Regions       []string
	ResourceTypes []string
	AccountID     string
}
