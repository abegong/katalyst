// Package config loads a project's configuration from its .katalyst/
// directory and answers two questions:
//
//  1. Which schemas exist (by name → absolute file path)?
//  2. Which named collections exist, and what checks does each run?
//
// A project is the nearest ancestor directory that contains a .katalyst/
// subdirectory. Schemas and collections are each defined one named file
// per definition under .katalyst/schemas/ and .katalyst/collections/
// (discovery: convention, the default), or listed explicitly in
// .katalyst/config.yaml (discovery: explicit). The file format (yaml,
// json, or both) is also set per kind in config.yaml. See
// product/specs/project-layout-spec.md and docs/content/explanation/configuration.md.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Dir is the per-project directory that marks a project root and holds
// its config, schemas, and collections.
const Dir = ".katalyst"

// configFile is the optional settings file inside Dir.
const configFile = "config.yaml"

// Subdirectories of Dir holding one named file per definition.
const (
	schemasSubdir     = "schemas"
	collectionsSubdir = "collections"
)

// Discovery modes for a kind (schemas or collections).
const (
	discoveryConvention = "convention"
	discoveryExplicit   = "explicit"
)

// defaultPattern is the glob applied to a collection's directory when the
// collection does not set its own `pattern`.
const defaultPattern = "*.md"

// ErrNotFound is returned when no .katalyst/ directory is present in the
// starting directory or any of its ancestors.
var ErrNotFound = errors.New("config: .katalyst/ not found")

// Config is the parsed, validated, root-relative-resolved configuration.
//
// Schemas maps the schema name to an *absolute* file path; relative
// paths in the source YAML are resolved against Root.
//
// Collections are sorted by name for deterministic output.
type Config struct {
	// Root is the absolute directory containing the .katalyst/ dir.
	Root string
	// Schemas is name → absolute path.
	Schemas map[string]string
	// Collections in name order.
	Collections []Collection
}

// CheckKind identifies the check implementation attached to a collection.
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

// Collection is a named group of items backed by a directory of files.
type Collection struct {
	// Name is the public handle (the key in the config `collections:` map).
	Name string
	// Path is the directory, relative to Root, as written in the config.
	Path string
	// Dir is the absolute directory (Root + Path).
	Dir string
	// Pattern is the filename glob for items (default "*.md").
	Pattern string
	// Schema is the object-schema name associated with the collection, or
	// "" when the collection is configured with explicit checks only. It
	// mirrors the first object check's schema, for display.
	Schema string
	// Checks to run against each item.
	Checks []Check
}

// rawConfig mirrors .katalyst/config.yaml. Both blocks are optional; an
// absent file (or an absent block) means convention discovery with the
// default YAML format.
type rawConfig struct {
	Schemas     rawSchemaKind     `yaml:"schemas"`
	Collections rawCollectionKind `yaml:"collections"`
}

// rawSchemaKind configures how schemas are discovered. Defs is consulted
// only when Discovery is "explicit" (name → file path).
type rawSchemaKind struct {
	Discovery string            `yaml:"discovery"`
	Format    string            `yaml:"format"`
	Defs      map[string]string `yaml:"defs"`
}

// rawCollectionKind configures how collections are discovered. Defs is
// consulted only when Discovery is "explicit" (name → definition).
type rawCollectionKind struct {
	Discovery string                   `yaml:"discovery"`
	Format    string                   `yaml:"format"`
	Defs      map[string]rawCollection `yaml:"defs"`
}

type rawCollection struct {
	Path    string     `yaml:"path"`
	Pattern string     `yaml:"pattern"`
	Schema  string     `yaml:"schema"`
	Checks  []rawCheck `yaml:"checks"`
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

// Load finds the project root (nearest ancestor with a .katalyst/ dir),
// reads the optional .katalyst/config.yaml, and resolves schemas and
// collections per the configured discovery and format. Schema paths are
// absolute; collections are sorted by name and validated for internal
// consistency (every referenced schema exists, every collection has at
// least one check).
func Load(start string) (*Config, error) {
	root, err := find(start)
	if err != nil {
		return nil, err
	}

	raw, err := readConfigFile(root)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Root:        root,
		Schemas:     make(map[string]string),
		Collections: make([]Collection, 0),
	}
	if err := cfg.loadSchemas(raw.Schemas); err != nil {
		return nil, err
	}
	if err := cfg.loadCollections(raw.Collections); err != nil {
		return nil, err
	}
	return cfg, nil
}

// readConfigFile parses .katalyst/config.yaml if it exists. A missing
// file yields a zero rawConfig (all defaults).
func readConfigFile(root string) (rawConfig, error) {
	var raw rawConfig
	rel := filepath.Join(Dir, configFile)
	src, err := os.ReadFile(filepath.Join(root, rel))
	if errors.Is(err, os.ErrNotExist) {
		return raw, nil
	}
	if err != nil {
		return raw, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(src, &raw); err != nil {
		return raw, fmt.Errorf("parse %s: %w", rel, err)
	}
	return raw, nil
}

// loadSchemas populates c.Schemas (name → absolute path) from either the
// schemas directory (convention) or an explicit defs map.
func (c *Config) loadSchemas(k rawSchemaKind) error {
	discovery, err := normDiscovery(k.Discovery)
	if err != nil {
		return fmt.Errorf("schemas: %w", err)
	}
	if discovery == discoveryExplicit {
		if len(k.Defs) == 0 {
			return errors.New(`schemas: discovery "explicit" requires a non-empty "defs" map`)
		}
		for name, p := range k.Defs {
			c.Schemas[name] = resolve(c.Root, p)
		}
		return nil
	}
	exts, err := formatExts(k.Format)
	if err != nil {
		return fmt.Errorf("schemas: %w", err)
	}
	found, err := scanKindDir(filepath.Join(c.Root, Dir, schemasSubdir), exts)
	if err != nil {
		return fmt.Errorf("schemas: %w", err)
	}
	for name, path := range found {
		c.Schemas[name] = path
	}
	return nil
}

// loadCollections populates c.Collections (sorted by name) from either
// the collections directory (convention) or an explicit defs map.
func (c *Config) loadCollections(k rawCollectionKind) error {
	discovery, err := normDiscovery(k.Discovery)
	if err != nil {
		return fmt.Errorf("collections: %w", err)
	}

	defs := map[string]rawCollection{}
	if discovery == discoveryExplicit {
		if len(k.Defs) == 0 {
			return errors.New(`collections: discovery "explicit" requires a non-empty "defs" map`)
		}
		defs = k.Defs
	} else {
		exts, err := formatExts(k.Format)
		if err != nil {
			return fmt.Errorf("collections: %w", err)
		}
		found, err := scanKindDir(filepath.Join(c.Root, Dir, collectionsSubdir), exts)
		if err != nil {
			return fmt.Errorf("collections: %w", err)
		}
		for name, path := range found {
			src, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("collection %q: %w", name, err)
			}
			var rc rawCollection
			if err := yaml.Unmarshal(src, &rc); err != nil {
				return fmt.Errorf("collection %q: %w", name, err)
			}
			defs[name] = rc
		}
	}

	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		col, err := c.buildCollection(name, defs[name])
		if err != nil {
			return err
		}
		c.Collections = append(c.Collections, col)
	}
	return nil
}

// buildCollection turns one raw collection definition into a validated
// Collection. The name comes from the source (filename stem in convention
// mode, map key in explicit mode), never from the file body.
func (c *Config) buildCollection(name string, rc rawCollection) (Collection, error) {
	dirRel := rc.Path
	if dirRel == "" {
		// A collection without an explicit path defaults to a directory
		// named after the collection itself.
		dirRel = name
	}
	pattern := rc.Pattern
	if pattern == "" {
		pattern = defaultPattern
	}

	checks := make([]Check, 0, len(rc.Checks)+1)
	if rc.Schema != "" {
		if _, ok := c.Schemas[rc.Schema]; !ok {
			return Collection{}, fmt.Errorf("collection %q: unknown schema %q", name, rc.Schema)
		}
		checks = append(checks, Check{Kind: CheckObject, Schema: rc.Schema})
	}
	for j, raw := range rc.Checks {
		ch, err := normalizeCheck(raw, c.Schemas)
		if err != nil {
			return Collection{}, fmt.Errorf("collection %q: checks[%d]: %w", name, j, err)
		}
		checks = append(checks, ch)
	}
	if len(checks) == 0 {
		return Collection{}, fmt.Errorf("collection %q: no checks configured (set schema or checks)", name)
	}

	schemaName := ""
	for _, ch := range checks {
		if ch.Kind == CheckObject {
			schemaName = ch.Schema
			break
		}
	}

	return Collection{
		Name:    name,
		Path:    dirRel,
		Dir:     resolve(c.Root, dirRel),
		Pattern: pattern,
		Schema:  schemaName,
		Checks:  checks,
	}, nil
}

// normDiscovery validates and defaults a kind's discovery mode.
func normDiscovery(d string) (string, error) {
	switch d {
	case "", discoveryConvention:
		return discoveryConvention, nil
	case discoveryExplicit:
		return discoveryExplicit, nil
	default:
		return "", fmt.Errorf("unknown discovery %q (want convention or explicit)", d)
	}
}

// formatExts maps a kind's format option to the file extensions a
// convention scan accepts. The empty string defaults to yaml.
func formatExts(format string) ([]string, error) {
	switch format {
	case "", "yaml":
		return []string{".yaml", ".yml"}, nil
	case "json":
		return []string{".json"}, nil
	case "both":
		return []string{".yaml", ".yml", ".json"}, nil
	default:
		return nil, fmt.Errorf("unknown format %q (want yaml, json, or both)", format)
	}
}

// scanKindDir lists files in dir whose extension is in exts and returns a
// map of filename stem → absolute path. A stem claimed by two files (e.g.
// foo.yaml and foo.json under format "both") is an error. A missing dir
// yields an empty map.
func scanKindDir(dir string, exts []string) (map[string]string, error) {
	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if !contains(exts, ext) {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), ext)
		if prev, ok := out[stem]; ok {
			return nil, fmt.Errorf("%q is defined by two files (%s and %s)", stem, filepath.Base(prev), e.Name())
		}
		out[stem] = filepath.Join(dir, e.Name())
	}
	return out, nil
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
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

// Collection returns the named collection, or false if it does not exist.
func (c *Config) Collection(name string) (Collection, bool) {
	for _, col := range c.Collections {
		if col.Name == name {
			return col, true
		}
	}
	return Collection{}, false
}

// CollectionNames returns the collection names in lexicographic order.
func (c *Config) CollectionNames() []string {
	names := make([]string, 0, len(c.Collections))
	for _, col := range c.Collections {
		names = append(names, col.Name)
	}
	return names
}

// Ext returns the file extension implied by the collection's pattern
// (e.g. "*.md" → ".md"). Used for reverse id→path resolution. Falls back
// to ".md" when the pattern has no extension.
func (c Collection) Ext() string {
	ext := filepath.Ext(c.Pattern)
	if ext == "" {
		return ".md"
	}
	return ext
}

// find walks from start upward until it locates a directory containing a
// .katalyst/ subdirectory. The returned root is the absolute,
// symlink-resolved directory.
//
// Symlink resolution matters on macOS where temp dirs (and sometimes
// user home dirs) live behind /var -> /private/var.
func find(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve start dir: %w", err)
	}
	dir := abs
	for {
		info, statErr := os.Stat(filepath.Join(dir, Dir))
		if statErr == nil && info.IsDir() {
			resolved, err := filepath.EvalSymlinks(dir)
			if err != nil {
				resolved = dir
			}
			return resolved, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
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
