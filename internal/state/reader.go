package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-json"
	"github.com/terraform-drift-detector/driftdetect/internal/extract"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// rawState represents Terraform state file format version 4.
type rawState struct {
	Version   int          `json:"version"`
	Resources []rawResource `json:"resources"`
}

type rawResource struct {
	Mode      string         `json:"mode"`
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	Module    string         `json:"module,omitempty"`
	Provider  string         `json:"provider"`
	Instances []rawInstance  `json:"instances"`
}

type rawInstance struct {
	Attributes map[string]any `json:"attributes"`
}

// Reader loads and parses Terraform state.
type Reader interface {
	Read(ctx context.Context, path string) ([]models.Resource, error)
}

// LocalReader reads state from a local file.
type LocalReader struct {
	extractor *extract.StateExtractor
}

// NewLocalReader creates a reader for local state files.
func NewLocalReader() *LocalReader {
	return &LocalReader{extractor: extract.NewStateExtractor()}
}

// Read parses a local terraform.tfstate file (v4 raw or show -json format).
func (r *LocalReader) Read(ctx context.Context, path string) ([]models.Resource, error) {
	_ = ctx
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}

	// Try terraform show -json format first.
	var showState tfjson.State
	if err := json.Unmarshal(data, &showState); err == nil && showState.Values != nil && showState.Values.RootModule != nil {
		return r.readShowJSON(&showState)
	}

	// Fall back to raw state v4 format.
	var raw rawState
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse state JSON: %w", err)
	}
	return r.readRawState(&raw)
}

func (r *LocalReader) readShowJSON(state *tfjson.State) ([]models.Resource, error) {
	var resources []models.Resource
	if state.Values == nil || state.Values.RootModule == nil {
		return resources, nil
	}
	root, err := r.walkModule(state.Values.RootModule, "")
	if err != nil {
		return nil, err
	}
	resources = append(resources, root...)
	return resources, nil
}

func (r *LocalReader) walkModule(mod *tfjson.StateModule, prefix string) ([]models.Resource, error) {
	var resources []models.Resource
	for _, rc := range mod.Resources {
		if rc.Mode != "managed" {
			continue
		}
		ref := rc.Address
		if prefix != "" && ref == "" {
			ref = prefix + "." + rc.Type + "." + rc.Name
		}
		if rc.Index != nil {
			ref = fmt.Sprintf("%s[%v]", ref, rc.Index)
		}
		res, err := r.extractor.Extract(rc.Type, ref, rc.AttributeValues)
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", ref, err)
		}
		if res != nil {
			resources = append(resources, *res)
		}
	}
	for _, child := range mod.ChildModules {
		childPrefix := child.Address
		childRes, err := r.walkModule(child, childPrefix)
		if err != nil {
			return nil, err
		}
		resources = append(resources, childRes...)
	}
	return resources, nil
}

func (r *LocalReader) readRawState(state *rawState) ([]models.Resource, error) {
	var resources []models.Resource
	for _, rc := range state.Resources {
		if rc.Mode != "managed" {
			continue
		}
		ref := rc.Type + "." + rc.Name
		if rc.Module != "" {
			ref = rc.Module + "." + ref
		}
		for i, inst := range rc.Instances {
			addr := ref
			if len(rc.Instances) > 1 {
				addr = fmt.Sprintf("%s[%d]", ref, i)
			}
			res, err := r.extractor.Extract(rc.Type, addr, inst.Attributes)
			if err != nil {
				return nil, fmt.Errorf("extract %s: %w", addr, err)
			}
			if res != nil {
				resources = append(resources, *res)
			}
		}
	}
	return resources, nil
}

// NewReader returns the appropriate reader for a state path.
func NewReader(path string) Reader {
	return NewLocalReader()
}
