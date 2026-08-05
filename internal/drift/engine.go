package drift

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// ignoredAttributes are never compared for drift.
var ignoredAttributes = map[string]bool{
	"arn":              true,
	"id":               true,
	"tags_all":         true,
	"tags":             true,
	"last_modified":    true,
	"creation_date":    true,
	"created_at":       true,
	"updated_at":       true,
	"etag":             true,
	"request_id":       true,
	"revision":         true,
	"self_link":        true,
	"terraform_ref":    true,
	"fingerprint":      true,
	"generation":       true,
	"resource_version": true,
}

// comparableAttributes per resource type; empty means compare all non-ignored.
var comparableAttributes = map[string][]string{
	"aws_s3_bucket": {
		"bucket", "acl", "force_destroy", "object_lock_enabled",
	},
	"aws_instance": {
		"instance_type", "ami", "availability_zone", "monitoring",
	},
	"aws_iam_role": {
		"name", "path", "description", "max_session_duration",
	},
	"aws_vpc": {
		"cidr_block", "instance_tenancy", "enable_dns_hostnames",
	},
	"aws_security_group": {
		"name", "description", "vpc_id",
	},
	"azurerm_storage_account": {
		"name", "account_tier", "account_replication_type", "account_kind",
	},
	"azurerm_linux_virtual_machine": {
		"name", "size", "zone",
	},
	"azurerm_resource_group": {
		"name", "location",
	},
	"google_storage_bucket": {
		"name", "location", "storage_class",
	},
	"google_compute_instance": {
		"name", "machine_type", "zone",
	},
	"google_compute_network": {
		"name", "auto_create_subnetworks", "routing_mode",
	},
}

// Engine compares expected and actual resource sets.
type Engine struct {
	now func() time.Time
}

// NewEngine creates a drift comparison engine.
func NewEngine() *Engine {
	return &Engine{now: time.Now}
}

// Compare produces a drift report from expected (state) and actual (cloud) resources.
func (e *Engine) Compare(stateSource string, provider models.Provider, expected, actual []models.Resource) models.DriftReport {
	started := e.now()
	matcher := newMatcher()

	expectedIdx := matcher.index(expected)
	actualIdx := matcher.index(actual)

	var drifts []models.DriftItem
	matchedActual := make(map[string]bool)

	for key, exp := range expectedIdx {
		act, ok := actualIdx[key]
		if !ok {
			drifts = append(drifts, models.DriftItem{
				Type:         models.DriftMissingInCloud,
				ResourceType: exp.Type,
				ResourceID:   exp.ID,
				ResourceName: exp.Name,
				TerraformRef: exp.TerraformRef,
				Region:       exp.Region,
				Message:      fmt.Sprintf("resource %s exists in state but not in cloud", exp.ID),
			})
			continue
		}
		matchedActual[key] = true
		drifts = append(drifts, diffResource(exp, act)...)
	}

	for key, act := range actualIdx {
		if matchedActual[key] {
			continue
		}
		drifts = append(drifts, models.DriftItem{
			Type:         models.DriftMissingInState,
			ResourceType: act.Type,
			ResourceID:   act.ID,
			ResourceName: act.Name,
			Region:       act.Region,
			Message:      fmt.Sprintf("resource %s exists in cloud but not in state", act.ID),
		})
	}

	sort.Slice(drifts, func(i, j int) bool {
		if drifts[i].Type != drifts[j].Type {
			return drifts[i].Type < drifts[j].Type
		}
		return drifts[i].ResourceID < drifts[j].ResourceID
	})

	completed := e.now()
	return models.DriftReport{
		ScanID:      uuid.New().String(),
		StartedAt:   started,
		CompletedAt: completed,
		StateSource: stateSource,
		Provider:    provider,
		Summary:     summarize(drifts, len(expectedIdx), len(actualIdx)),
		Drifts:      drifts,
	}
}

func diffResource(expected, actual models.Resource) []models.DriftItem {
	var items []models.DriftItem

	attrChanges := diffAttributes(expected.Type, expected.Attributes, actual.Attributes)
	if len(attrChanges) > 0 {
		items = append(items, models.DriftItem{
			Type:             models.DriftAttribute,
			ResourceType:     expected.Type,
			ResourceID:       expected.ID,
			ResourceName:     expected.Name,
			TerraformRef:     expected.TerraformRef,
			Region:           expected.Region,
			AttributeChanges: attrChanges,
			Message:          fmt.Sprintf("%d attribute(s) differ", len(attrChanges)),
		})
	}

	tagChanges := diffTags(expected.Tags, actual.Tags)
	if len(tagChanges) > 0 {
		items = append(items, models.DriftItem{
			Type:         models.DriftTag,
			ResourceType: expected.Type,
			ResourceID:   expected.ID,
			ResourceName: expected.Name,
			TerraformRef: expected.TerraformRef,
			Region:       expected.Region,
			TagChanges:   tagChanges,
			Message:      fmt.Sprintf("%d tag change(s)", len(tagChanges)),
		})
	}

	return items
}

func diffAttributes(resourceType string, expected, actual map[string]any) []models.AttributeChange {
	if expected == nil {
		expected = map[string]any{}
	}
	if actual == nil {
		actual = map[string]any{}
	}

	keys := attributeKeys(resourceType, expected, actual)
	var changes []models.AttributeChange

	for _, k := range keys {
		if ignoredAttributes[k] || strings.HasPrefix(k, "timeouts.") {
			continue
		}
		ev, eok := expected[k]
		av, aok := actual[k]
		if !eok && !aok {
			continue
		}
		if !eok || !aok || !valuesEqual(ev, av) {
			changes = append(changes, models.AttributeChange{
				Attribute: k,
				Expected:  ev,
				Actual:    av,
			})
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Attribute < changes[j].Attribute
	})
	return changes
}

func attributeKeys(resourceType string, maps ...map[string]any) []string {
	seen := make(map[string]bool)
	var keys []string

	if allowed, ok := comparableAttributes[resourceType]; ok {
		for _, k := range allowed {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
		return keys
	}

	for _, m := range maps {
		for k := range m {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

func diffTags(expected, actual map[string]string) []models.TagChange {
	if expected == nil {
		expected = map[string]string{}
	}
	if actual == nil {
		actual = map[string]string{}
	}

	seen := make(map[string]bool)
	var changes []models.TagChange

	for k, ev := range expected {
		seen[k] = true
		av, ok := actual[k]
		if !ok {
			changes = append(changes, models.TagChange{Key: k, Expected: ev, Change: "removed"})
			continue
		}
		if ev != av {
			changes = append(changes, models.TagChange{Key: k, Expected: ev, Actual: av, Change: "modified"})
		}
	}

	for k, av := range actual {
		if seen[k] {
			continue
		}
		changes = append(changes, models.TagChange{Key: k, Actual: av, Change: "added"})
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Key < changes[j].Key
	})
	return changes
}

func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if av.Kind() == reflect.Slice && bv.Kind() == reflect.Slice {
		if av.Len() != bv.Len() {
			return false
		}
		for i := 0; i < av.Len(); i++ {
			if !valuesEqual(av.Index(i).Interface(), bv.Index(i).Interface()) {
				return false
			}
		}
		return true
	}

	if av.Kind() == reflect.Map && bv.Kind() == reflect.Map {
		if av.Len() != bv.Len() {
			return false
		}
		for _, k := range av.MapKeys() {
			bvVal := bv.MapIndex(k)
			if !bvVal.IsValid() {
				return false
			}
			if !valuesEqual(av.MapIndex(k).Interface(), bvVal.Interface()) {
				return false
			}
		}
		return true
	}

	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func summarize(drifts []models.DriftItem, expectedCount, actualCount int) models.DriftSummary {
	s := models.DriftSummary{
		TotalDrifts:      len(drifts),
		ResourcesChecked: expectedCount + actualCount,
	}
	for _, d := range drifts {
		switch d.Type {
		case models.DriftMissingInCloud:
			s.MissingInCloud++
		case models.DriftMissingInState:
			s.MissingInState++
		case models.DriftAttribute:
			s.AttributeDrifts++
		case models.DriftTag:
			s.TagDrifts++
		}
	}
	return s
}
