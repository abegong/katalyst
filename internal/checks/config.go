package checks

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// RawCheck mirrors one `checks:` entry. The struct fields exist so a misspelled
// key fails YAML's known-field validation; the retained node is what a check
// type's own parser decodes for its real args.
type RawCheck struct {
	Kind      string   `yaml:"kind"`
	Schema    string   `yaml:"schema"`
	Field     string   `yaml:"field"`
	Type      string   `yaml:"type"`
	Value     string   `yaml:"value"`
	Values    []string `yaml:"values"`
	Min       *float64 `yaml:"min"`
	Max       *float64 `yaml:"max"`
	MinLength int      `yaml:"min_length"`
	MaxLength int      `yaml:"max_length"`
	Heading   string   `yaml:"heading"`
	Style     string   `yaml:"style"`
	Target    string   `yaml:"target"`
	Transform string   `yaml:"transform"`
	Prefix    string   `yaml:"prefix"`
	Suffix    string   `yaml:"suffix"`
	Allow     []string `yaml:"allow"`
	Deny      []string `yaml:"deny"`
	Pattern   string   `yaml:"pattern"`
	Fields    []string `yaml:"fields"`
	Name      string   `yaml:"name"`
	Match     string   `yaml:"match"`
	Select    string   `yaml:"select"`
	Fix       string   `yaml:"fix"`

	node *yaml.Node
}

var rawCheckKeys = map[string]bool{
	"kind": true, "schema": true, "field": true, "type": true,
	"value": true, "values": true, "min": true, "max": true,
	"min_length": true, "max_length": true, "heading": true,
	"style": true, "target": true, "transform": true,
	"prefix": true, "suffix": true, "allow": true, "deny": true,
	"pattern": true, "fields": true, "name": true, "match": true,
	"select": true, "fix": true,
}

// UnmarshalYAML decodes the entry's fields and stashes the raw node, so the
// node can travel to a check type's own parser.
func (rc *RawCheck) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("invalid check: expected a mapping")
	}
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i].Value
		if !rawCheckKeys[key] {
			return fmt.Errorf("unknown check key %q", key)
		}
	}
	type plain RawCheck
	var p plain
	if err := value.Decode(&p); err != nil {
		return err
	}
	*rc = RawCheck(p)
	rc.node = value
	return nil
}

// BuildConfiguredInput carries the shared pieces needed to turn raw config
// into validated configured checks.
type BuildConfiguredInput struct {
	ErrorContext   string
	Schema         string
	Raw            []RawCheck
	SchemaKnown    func(string) bool
	ConfigurableIn string
	AllowObject    bool
}

// BuildConfigured folds an optional schema name into a leading object check
// and parses all raw checks through the registry.
func BuildConfigured(in BuildConfiguredInput) ([]ConfiguredCheck, error) {
	configurableIn := in.ConfigurableIn
	if configurableIn == "" {
		configurableIn = ConfigCollection
	}
	out := make([]ConfiguredCheck, 0, len(in.Raw)+1)
	if in.Schema != "" {
		if !in.AllowObject {
			return nil, fmt.Errorf("%s: schema is not supported for %s checks", in.ErrorContext, configurableIn)
		}
		if in.SchemaKnown != nil && !in.SchemaKnown(in.Schema) {
			return nil, fmt.Errorf("%s: unknown schema %q", in.ErrorContext, in.Schema)
		}
		out = append(out, ConfiguredCheck{Kind: CheckObject, Schema: in.Schema})
	}
	for j, raw := range in.Raw {
		kind := CheckType(strings.TrimSpace(raw.Kind))
		if kind == CheckObject {
			if !in.AllowObject {
				return nil, fmt.Errorf("%s: checks[%d]: object check is not supported for %s checks", in.ErrorContext, j, configurableIn)
			}
			if raw.Schema == "" {
				return nil, fmt.Errorf("%s: checks[%d]: object check requires \"schema\"", in.ErrorContext, j)
			}
			if in.SchemaKnown != nil && !in.SchemaKnown(raw.Schema) {
				return nil, fmt.Errorf("%s: checks[%d]: unknown schema %q", in.ErrorContext, j, raw.Schema)
			}
			if raw.Field != "" {
				return nil, fmt.Errorf("%s: checks[%d]: object check does not support \"field\"", in.ErrorContext, j)
			}
			out = append(out, ConfiguredCheck{Kind: CheckObject, Schema: raw.Schema})
			continue
		}
		args, err := Parse(kind, raw.node)
		if err != nil {
			return nil, fmt.Errorf("%s: checks[%d]: %w", in.ErrorContext, j, err)
		}
		if !SupportsConfiguration(kind, configurableIn) {
			return nil, fmt.Errorf("%s: checks[%d]: check type %q does not support %s checks", in.ErrorContext, j, kind, configurableIn)
		}
		out = append(out, ConfiguredCheck{Kind: kind, Args: args})
	}
	return out, nil
}
