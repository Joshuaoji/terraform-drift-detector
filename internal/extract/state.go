package extract

import (
	"fmt"
	"strings"

	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// StateExtractor converts Terraform state attributes to normalized resources.
type StateExtractor struct {
	handlers map[string]stateHandler
}

type stateHandler func(ref string, attrs map[string]any) (*models.Resource, error)

// NewStateExtractor creates a state resource extractor.
func NewStateExtractor() *StateExtractor {
	e := &StateExtractor{handlers: map[string]stateHandler{}}
	e.registerAWSHandlers()
	e.registerAzureHandlers()
	e.registerGCPHandlers()
	return e
}

// Extract normalizes a state resource.
func (e *StateExtractor) Extract(resourceType, ref string, attrs map[string]any) (*models.Resource, error) {
	if handler, ok := e.handlers[resourceType]; ok {
		return handler(ref, attrs)
	}
	return e.genericExtract(resourceType, ref, attrs)
}

func (e *StateExtractor) registerAWSHandlers() {
	e.handlers["aws_s3_bucket"] = func(ref string, attrs map[string]any) (*models.Resource, error) {
		id := stringAttr(attrs, "id", "bucket")
		return &models.Resource{
			ID:           id,
			Type:         "aws_s3_bucket",
			Provider:     models.ProviderAWS,
			Name:         stringAttr(attrs, "bucket"),
			Region:       stringAttr(attrs, "region"),
			Attributes:   pickAttrs(attrs, "bucket", "acl", "force_destroy", "object_lock_enabled"),
			Tags:         extractTags(attrs),
			TerraformRef: ref,
			Metadata:     map[string]string{"arn": stringAttr(attrs, "arn")},
		}, nil
	}

	e.handlers["aws_instance"] = func(ref string, attrs map[string]any) (*models.Resource, error) {
		id := stringAttr(attrs, "id")
		return &models.Resource{
			ID:           id,
			Type:         "aws_instance",
			Provider:     models.ProviderAWS,
			Name:         stringAttr(attrs, "tags", "Name"),
			Region:       regionFromAZ(stringAttr(attrs, "availability_zone")),
			Attributes:   pickAttrs(attrs, "instance_type", "ami", "availability_zone", "monitoring"),
			Tags:         extractTags(attrs),
			TerraformRef: ref,
			Metadata:     map[string]string{"arn": stringAttr(attrs, "arn")},
		}, nil
	}

	e.handlers["aws_iam_role"] = func(ref string, attrs map[string]any) (*models.Resource, error) {
		id := stringAttr(attrs, "id", "name")
		return &models.Resource{
			ID:           id,
			Type:         "aws_iam_role",
			Provider:     models.ProviderAWS,
			Name:         stringAttr(attrs, "name"),
			Region:       "global",
			Attributes:   pickAttrs(attrs, "name", "path", "description", "max_session_duration"),
			Tags:         extractTags(attrs),
			TerraformRef: ref,
			Metadata:     map[string]string{"arn": stringAttr(attrs, "arn")},
		}, nil
	}
}

func (e *StateExtractor) registerAzureHandlers() {
	e.handlers["azurerm_storage_account"] = func(ref string, attrs map[string]any) (*models.Resource, error) {
		id := stringAttr(attrs, "id", "name")
		return &models.Resource{
			ID:           id,
			Type:         "azurerm_storage_account",
			Provider:     models.ProviderAzure,
			Name:         stringAttr(attrs, "name"),
			Region:       stringAttr(attrs, "location"),
			Attributes:   pickAttrs(attrs, "name", "account_tier", "account_replication_type", "account_kind"),
			Tags:         extractTags(attrs),
			TerraformRef: ref,
		}, nil
	}

	e.handlers["azurerm_linux_virtual_machine"] = func(ref string, attrs map[string]any) (*models.Resource, error) {
		id := stringAttr(attrs, "id", "name")
		return &models.Resource{
			ID:           id,
			Type:         "azurerm_linux_virtual_machine",
			Provider:     models.ProviderAzure,
			Name:         stringAttr(attrs, "name"),
			Region:       stringAttr(attrs, "location"),
			Attributes:   pickAttrs(attrs, "name", "size", "zone"),
			Tags:         extractTags(attrs),
			TerraformRef: ref,
		}, nil
	}
}

func (e *StateExtractor) registerGCPHandlers() {
	e.handlers["google_storage_bucket"] = func(ref string, attrs map[string]any) (*models.Resource, error) {
		id := stringAttr(attrs, "id", "name")
		return &models.Resource{
			ID:           id,
			Type:         "google_storage_bucket",
			Provider:     models.ProviderGCP,
			Name:         stringAttr(attrs, "name"),
			Region:       stringAttr(attrs, "location"),
			Attributes:   pickAttrs(attrs, "name", "location", "storage_class"),
			Tags:         extractLabels(attrs),
			TerraformRef: ref,
		}, nil
	}

	e.handlers["google_compute_instance"] = func(ref string, attrs map[string]any) (*models.Resource, error) {
		id := stringAttr(attrs, "id", "name")
		return &models.Resource{
			ID:           id,
			Type:         "google_compute_instance",
			Provider:     models.ProviderGCP,
			Name:         stringAttr(attrs, "name"),
			Region:       stringAttr(attrs, "zone"),
			Attributes:   pickAttrs(attrs, "name", "machine_type", "zone"),
			Tags:         extractLabels(attrs),
			TerraformRef: ref,
		}, nil
	}
}

func extractLabels(attrs map[string]any) map[string]string {
	labels := map[string]string{}
	if raw, ok := attrs["labels"].(map[string]any); ok {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				labels[k] = s
			}
		}
	}
	return labels
}

func (e *StateExtractor) genericExtract(resourceType, ref string, attrs map[string]any) (*models.Resource, error) {
	provider := detectProvider(resourceType)
	if provider == "" {
		return nil, nil
	}
	id := stringAttr(attrs, "id", "name")
	if id == "" {
		return nil, nil
	}
	return &models.Resource{
		ID:           id,
		Type:         resourceType,
		Provider:     provider,
		Name:         stringAttr(attrs, "name"),
		Region:       stringAttr(attrs, "region", "location", "zone"),
		Attributes:   copyAttrs(attrs),
		Tags:         extractTags(attrs),
		TerraformRef: ref,
	}, nil
}

func detectProvider(resourceType string) models.Provider {
	switch {
	case strings.HasPrefix(resourceType, "aws_"):
		return models.ProviderAWS
	case strings.HasPrefix(resourceType, "azurerm_"):
		return models.ProviderAzure
	case strings.HasPrefix(resourceType, "google_"):
		return models.ProviderGCP
	default:
		return ""
	}
}

func stringAttr(attrs map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := attrs[k]; ok && v != nil {
			switch val := v.(type) {
			case string:
				if val != "" {
					return val
				}
			case map[string]any:
				if name, ok := val["Name"].(string); ok {
					return name
				}
			}
		}
	}
	return ""
}

func extractTags(attrs map[string]any) map[string]string {
	tags := map[string]string{}
	if raw, ok := attrs["tags"].(map[string]any); ok {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				tags[k] = s
			}
		}
	}
	return tags
}

func pickAttrs(attrs map[string]any, keys ...string) map[string]any {
	out := make(map[string]any, len(keys))
	for _, k := range keys {
		if v, ok := attrs[k]; ok {
			out[k] = v
		}
	}
	return out
}

func copyAttrs(attrs map[string]any) map[string]any {
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		if k == "tags" || k == "tags_all" {
			continue
		}
		out[k] = v
	}
	return out
}

func regionFromAZ(az string) string {
	if az == "" {
		return ""
	}
	// us-east-1a -> us-east-1
	if len(az) > 1 && az[len(az)-1] >= 'a' && az[len(az)-1] <= 'z' {
		return az[:len(az)-1]
	}
	return az
}

// FilterByTypes returns resources matching the given types (empty = all).
func FilterByTypes(resources []models.Resource, types []string) []models.Resource {
	if len(types) == 0 {
		return resources
	}
	allowed := make(map[string]bool, len(types))
	for _, t := range types {
		allowed[t] = true
	}
	var filtered []models.Resource
	for _, r := range resources {
		if allowed[r.Type] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterByProvider returns resources for a specific provider.
func FilterByProvider(resources []models.Resource, provider models.Provider) []models.Resource {
	var filtered []models.Resource
	for _, r := range resources {
		if r.Provider == provider {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// ValidateProvider ensures all resources belong to expected provider.
func ValidateProvider(resources []models.Resource, provider models.Provider) error {
	for _, r := range resources {
		if r.Provider != provider && r.Provider != "" {
			return fmt.Errorf("resource %s has provider %s, expected %s", r.TerraformRef, r.Provider, provider)
		}
	}
	return nil
}
