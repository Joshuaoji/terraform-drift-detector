package drift

import (
	"strings"

	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

type matcher struct{}

func newMatcher() *matcher {
	return &matcher{}
}

// index builds a lookup map from normalized match keys to resources.
func (m *matcher) index(resources []models.Resource) map[string]models.Resource {
	idx := make(map[string]models.Resource, len(resources))
	for _, r := range resources {
		key := m.matchKey(r)
		if key == "" {
			continue
		}
		idx[key] = r
	}
	return idx
}

func (m *matcher) matchKey(r models.Resource) string {
	id := normalizeID(r.ID)
	if id == "" {
		id = normalizeID(r.Name)
	}
	if id == "" {
		return ""
	}
	return strings.Join([]string{string(r.Provider), r.Type, r.Region, id}, "|")
}

func normalizeID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	// Strip ARN prefix to get resource suffix for matching.
	if strings.HasPrefix(id, "arn:") {
		parts := strings.Split(id, ":")
		if len(parts) >= 6 {
			return parts[len(parts)-1]
		}
	}
	return strings.ToLower(id)
}
