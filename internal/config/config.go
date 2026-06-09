// Package config loads katalyst.yaml from the nearest ancestor
// directory and answers two questions:
//
//  1. Which schemas exist (by name → absolute file path)?
//  2. For a given markdown file path, which rule/check set applies?
//
// See product/decisions.md (D1, D2) for the rationale behind the
// nearest-ancestor lookup and the first-match-wins rule semantics.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

// Filename is the fixed config name we look for during ascent.
const Filename = "katalyst.yaml"

// ErrNotFound is returned when no katalyst.yaml is present in the
// starting directory or any of its ancestors.
var ErrNotFound = errors.New("config: katalyst.yaml not found")

// Config is the parsed, validated, root-relative-resolved configuration.
//
// Schemas maps the schema name to an *absolute* file path; relative
// paths in the source YAML are resolved against Root.
//
// Rules retains the source order (first match wins).
type Config struct {
	// Root is the absolute directory containing katalyst.yaml.
	Root string
	// Schemas is name → absolute path.
	Schemas map[string]string
	// Rules in source order.
	Rules []Rule
}

// CheckKind identifies the check implementation attached to a rule.
type CheckKind string

const (
	CheckObject                        CheckKind = "object"
	CheckObjectRequiredField           CheckKind = "object_required_field"
	CheckObjectFieldType               CheckKind = "object_field_type"
	CheckObjectFieldEnum               CheckKind = "object_field_enum"
	CheckObjectNumberRange             CheckKind = "object_number_range"
	CheckObjectStringLength            CheckKind = "object_string_length"
	CheckMarkdownTitleMatchesH1        CheckKind = "markdown_title_matches_h1"
	CheckMarkdownRequiresH1            CheckKind = "markdown_requires_h1"
	CheckMarkdownSingleH1              CheckKind = "markdown_single_h1"
	CheckMarkdownNoHeadingLevelJumps   CheckKind = "markdown_no_heading_level_jumps"
	CheckMarkdownRequiredSection       CheckKind = "markdown_required_section"
	CheckMarkdownCodeFenceHasLanguage  CheckKind = "markdown_code_fence_language_required"
	CheckFilesystemFilenameMatchesSlug CheckKind = "filesystem_filename_matches_slug"
	CheckFilesystemExtensionIn         CheckKind = "filesystem_extension_in"
	CheckFilesystemFilenameKebabCase   CheckKind = "filesystem_filename_kebab_case"
	CheckFilesystemNoSpacesInPath      CheckKind = "filesystem_no_spaces_in_path"
	CheckFilesystemParentDirIn         CheckKind = "filesystem_parent_dir_in"
	CheckFilesystemFilenamePrefix      CheckKind = "filesystem_filename_prefix"
	defaultMarkdownTitleField                    = "title"
	defaultFilesystemSlugField                   = "slug"
)

// Check configures one validation check.
type Check struct {
	Kind      CheckKind
	Schema    string
	Field     string
	Type      string
	Value     string
	Values    []string
	Min       *float64
	Max       *float64
	MinLength int
	MaxLength int
	Heading   string
}

// Rule binds a glob pattern to one or more checks.
type Rule struct {
	// Paths is a doublestar-syntax glob, relative to Root.
	Paths string
	// Schema is retained for backward compatibility: it mirrors the
	// first object check's schema name, if present.
	Schema string
	// Checks to run when Paths matches.
	Checks []Check
}

// rawConfig mirrors the on-disk YAML shape, before validation.
type rawConfig struct {
	Schemas map[string]string `yaml:"schemas"`
	Rules   []rawRule         `yaml:"rules"`
}

type rawRule struct {
	Paths  string     `yaml:"paths"`
	Schema string     `yaml:"schema"`
	Checks []rawCheck `yaml:"checks"`
}

type rawCheck struct {
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
}

// Load finds katalyst.yaml by walking upward from start and parses
// it. The returned Config has all schema paths resolved to absolute
// form and has been validated for internal consistency (every rule
// references a known schema).
func Load(start string) (*Config, error) {
	root, path, err := find(start)
	if err != nil {
		return nil, err
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(src, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", Filename, err)
	}

	cfg := &Config{
		Root:    root,
		Schemas: make(map[string]string, len(raw.Schemas)),
		Rules:   make([]Rule, 0, len(raw.Rules)),
	}
	for name, p := range raw.Schemas {
		cfg.Schemas[name] = resolve(root, p)
	}
	for i, r := range raw.Rules {
		if r.Paths == "" {
			return nil, fmt.Errorf("rules[%d]: empty paths pattern", i)
		}

		checks := make([]Check, 0, len(r.Checks)+1)
		if r.Schema != "" {
			if _, ok := cfg.Schemas[r.Schema]; !ok {
				return nil, fmt.Errorf("rules[%d]: unknown schema %q", i, r.Schema)
			}
			checks = append(checks, Check{
				Kind:   CheckObject,
				Schema: r.Schema,
			})
		}

		for j, rawCheck := range r.Checks {
			ch, err := normalizeCheck(rawCheck, cfg.Schemas)
			if err != nil {
				return nil, fmt.Errorf("rules[%d].checks[%d]: %w", i, j, err)
			}
			checks = append(checks, ch)
		}
		if len(checks) == 0 {
			return nil, fmt.Errorf("rules[%d]: no checks configured (set schema or checks)", i)
		}

		schemaName := ""
		for _, ch := range checks {
			if ch.Kind == CheckObject {
				schemaName = ch.Schema
				break
			}
		}
		cfg.Rules = append(cfg.Rules, Rule{
			Paths:  r.Paths,
			Schema: schemaName,
			Checks: checks,
		})
	}

	return cfg, nil
}

// SchemaPath returns the absolute file path for a schema name, or "" if
// no such schema exists.
func (c *Config) SchemaPath(name string) string {
	return c.Schemas[name]
}

// SchemaNames returns the schema names in lexicographic order.
func (c *Config) SchemaNames() []string {
	names := make([]string, 0, len(c.Schemas))
	for n := range c.Schemas {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Match returns the schema name that should apply to filePath, using
// first-match-wins semantics over the rule list.
//
// filePath may be absolute or relative; it's compared as a path
// relative to the config root.
func (c *Config) Match(filePath string) (string, bool) {
	rule, ok := c.RuleFor(filePath)
	if !ok {
		return "", false
	}
	for _, ch := range rule.Checks {
		if ch.Kind == CheckObject {
			return ch.Schema, true
		}
	}
	return "", false
}

// RuleFor returns the first matching rule for filePath.
func (c *Config) RuleFor(filePath string) (Rule, bool) {
	rel, err := relativeToRoot(c.Root, filePath)
	if err != nil {
		return Rule{}, false
	}
	for _, r := range c.Rules {
		ok, err := doublestar.Match(r.Paths, rel)
		if err == nil && ok {
			return r, true
		}
	}
	return Rule{}, false
}

// find walks from start upward until it locates a directory containing
// Filename. The returned root is the absolute, symlink-resolved
// directory; path is the absolute path of the config file itself.
//
// Symlink resolution matters on macOS where temp dirs (and sometimes
// user home dirs) live behind /var -> /private/var. Without it, Match's
// later filepath.Rel call against a non-resolved input path produces
// "../../..." chains that never match any glob.
func find(start string) (root, path string, err error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", "", fmt.Errorf("resolve start dir: %w", err)
	}
	dir := abs
	for {
		candidate := filepath.Join(dir, Filename)
		info, statErr := os.Stat(candidate)
		if statErr == nil && !info.IsDir() {
			resolved, err := filepath.EvalSymlinks(dir)
			if err != nil {
				resolved = dir
			}
			return resolved, filepath.Join(resolved, Filename), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", ErrNotFound
		}
		dir = parent
	}
}

// resolve turns a config-relative path into an absolute one. Absolute
// paths in the source pass through unchanged.
func resolve(root, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(root, p))
}

// relativeToRoot makes filePath relative to root using forward slashes,
// which is what doublestar expects regardless of OS. Symlinks on both
// sides are resolved so we compare in a single canonical form, even
// when filePath itself doesn't exist on disk yet.
func relativeToRoot(root, filePath string) (string, error) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	abs = resolveSymlinks(abs)
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

// resolveSymlinks is like filepath.EvalSymlinks except it tolerates
// non-existent tail components: it walks up until it finds something
// that exists, resolves that, then re-attaches the missing tail. This
// matters for `Match` calls where the caller is asking a hypothetical
// question about a path that may not be on disk.
func resolveSymlinks(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	dir, base := filepath.Split(p)
	dir = filepath.Clean(dir)
	if dir == p || dir == "" || dir == string(filepath.Separator) {
		return p
	}
	return filepath.Join(resolveSymlinks(dir), base)
}

func normalizeCheck(raw rawCheck, schemas map[string]string) (Check, error) {
	kind := CheckKind(strings.TrimSpace(raw.Kind))
	switch kind {
	case CheckObject:
		if raw.Schema == "" {
			return Check{}, errors.New(`object check requires "schema"`)
		}
		if _, ok := schemas[raw.Schema]; !ok {
			return Check{}, fmt.Errorf("unknown schema %q", raw.Schema)
		}
		if raw.Field != "" {
			return Check{}, errors.New(`object check does not support "field"`)
		}
		return Check{Kind: CheckObject, Schema: raw.Schema}, nil
	case CheckObjectRequiredField:
		if raw.Field == "" {
			return Check{}, errors.New(`object_required_field requires "field"`)
		}
		return Check{Kind: kind, Field: raw.Field}, nil
	case CheckObjectFieldType:
		if raw.Field == "" {
			return Check{}, errors.New(`object_field_type requires "field"`)
		}
		if raw.Type == "" {
			return Check{}, errors.New(`object_field_type requires "type"`)
		}
		return Check{Kind: kind, Field: raw.Field, Type: raw.Type}, nil
	case CheckObjectFieldEnum:
		if raw.Field == "" {
			return Check{}, errors.New(`object_field_enum requires "field"`)
		}
		if len(raw.Values) == 0 {
			return Check{}, errors.New(`object_field_enum requires "values"`)
		}
		return Check{Kind: kind, Field: raw.Field, Values: raw.Values}, nil
	case CheckObjectNumberRange:
		if raw.Field == "" {
			return Check{}, errors.New(`object_number_range requires "field"`)
		}
		if raw.Min == nil && raw.Max == nil {
			return Check{}, errors.New(`object_number_range requires "min" or "max"`)
		}
		return Check{Kind: kind, Field: raw.Field, Min: raw.Min, Max: raw.Max}, nil
	case CheckObjectStringLength:
		if raw.Field == "" {
			return Check{}, errors.New(`object_string_length requires "field"`)
		}
		if raw.MinLength == 0 && raw.MaxLength == 0 {
			return Check{}, errors.New(`object_string_length requires "min_length" or "max_length"`)
		}
		return Check{Kind: kind, Field: raw.Field, MinLength: raw.MinLength, MaxLength: raw.MaxLength}, nil
	case CheckMarkdownTitleMatchesH1:
		if raw.Schema != "" {
			return Check{}, errors.New(`markdown_title_matches_h1 does not support "schema"`)
		}
		field := raw.Field
		if field == "" {
			field = defaultMarkdownTitleField
		}
		return Check{Kind: CheckMarkdownTitleMatchesH1, Field: field}, nil
	case CheckMarkdownRequiresH1:
		return Check{Kind: kind}, nil
	case CheckMarkdownSingleH1:
		return Check{Kind: kind}, nil
	case CheckMarkdownNoHeadingLevelJumps:
		return Check{Kind: kind}, nil
	case CheckMarkdownRequiredSection:
		if raw.Heading == "" {
			return Check{}, errors.New(`markdown_required_section requires "heading"`)
		}
		return Check{Kind: kind, Heading: raw.Heading}, nil
	case CheckMarkdownCodeFenceHasLanguage:
		return Check{Kind: kind}, nil
	case CheckFilesystemFilenameMatchesSlug:
		if raw.Schema != "" {
			return Check{}, errors.New(`filesystem_filename_matches_slug does not support "schema"`)
		}
		field := raw.Field
		if field == "" {
			field = defaultFilesystemSlugField
		}
		return Check{Kind: CheckFilesystemFilenameMatchesSlug, Field: field}, nil
	case CheckFilesystemExtensionIn:
		if len(raw.Values) == 0 {
			return Check{}, errors.New(`filesystem_extension_in requires "values"`)
		}
		return Check{Kind: kind, Values: raw.Values}, nil
	case CheckFilesystemFilenameKebabCase:
		return Check{Kind: kind}, nil
	case CheckFilesystemNoSpacesInPath:
		return Check{Kind: kind}, nil
	case CheckFilesystemParentDirIn:
		if len(raw.Values) == 0 {
			return Check{}, errors.New(`filesystem_parent_dir_in requires "values"`)
		}
		return Check{Kind: kind, Values: raw.Values}, nil
	case CheckFilesystemFilenamePrefix:
		if raw.Value == "" {
			return Check{}, errors.New(`filesystem_filename_prefix requires "value"`)
		}
		return Check{Kind: kind, Value: raw.Value}, nil
	case "":
		return Check{}, errors.New(`check kind is required`)
	default:
		return Check{}, fmt.Errorf("unknown check kind %q", raw.Kind)
	}
}
