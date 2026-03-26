package normalize

import "github.com/compatgate/compatgate/internal/findings"

type Contract struct {
	Protocol  findings.Protocol `json:"protocol"`
	Resources []Resource        `json:"resources"`
}

type Resource struct {
	Kind       string                   `json:"kind"`
	Name       string                   `json:"name"`
	Parent     string                   `json:"parent,omitempty"`
	Identifier string                   `json:"identifier"`
	Required   bool                     `json:"required,omitempty"`
	Type       string                   `json:"type,omitempty"`
	EnumValues []string                 `json:"enum_values,omitempty"`
	Meta       map[string]string        `json:"meta,omitempty"`
	Source     *findings.SourceLocation `json:"source,omitempty"`
}
