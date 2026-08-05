package state

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-json"
	"github.com/terraform-drift-detector/driftdetect/internal/extract"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

type rawState struct {
	Version   int           `json:"version"`
	Resources []rawResource `json:"resources"`
}

type rawResource struct {
	Mode      string        `json:"mode"`
	Type      string        `json:"type"`
	Name      string        `json:"name"`
	Module    string        `json:"module,omitempty"`
	Provider  string        `json:"provider"`
	Instances []rawInstance `json:"instances"`
}

type rawInstance struct {
	Attributes map[string]any `json:"attributes"`
}

// Parser converts Terraform state JSON bytes into normalized resources.
type Parser struct {
	extractor *extract.StateExtractor
}

// NewParser creates a state JSON parser.
func NewParser() *Parser {
	return &Parser{extractor: extract.NewStateExtractor()}
}

// Parse reads state bytes in terraform show -json or raw v4 format.
func (p *Parser) Parse(data []byte) ([]models.Resource, error) {
	var showState tfjson.State
	if err := json.Unmarshal(data, &showState); err == nil && showState.Values != nil && showState.Values.RootModule != nil {
		return p.readShowJSON(&showState)
	}

	var raw rawState
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse state JSON: %w", err)
	}
	return p.readRawState(&raw)
}

func (p *Parser) readShowJSON(state *tfjson.State) ([]models.Resource, error) {
	if state.Values == nil || state.Values.RootModule == nil {
		return nil, nil
	}
	return p.walkModule(state.Values.RootModule, "")
}

func (p *Parser) walkModule(mod *tfjson.StateModule, prefix string) ([]models.Resource, error) {
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
		res, err := p.extractor.Extract(rc.Type, ref, rc.AttributeValues)
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", ref, err)
		}
		if res != nil {
			resources = append(resources, *res)
		}
	}
	for _, child := range mod.ChildModules {
		childRes, err := p.walkModule(child, child.Address)
		if err != nil {
			return nil, err
		}
		resources = append(resources, childRes...)
	}
	return resources, nil
}

func (p *Parser) readRawState(state *rawState) ([]models.Resource, error) {
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
			res, err := p.extractor.Extract(rc.Type, addr, inst.Attributes)
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
