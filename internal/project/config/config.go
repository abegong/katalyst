// Package config loads a project's configuration from its .katalyst/
// directory and answers two questions:
//
//  1. Which schemas exist (by name → absolute file path)?
//  2. Which storage instances exist, what collections does each declare, and
//     what checks does each collection run?
//
// A project is the nearest ancestor directory that contains a .katalyst/
// subdirectory. Schemas are defined one named file per definition under
// .katalyst/schemas/; storage instances one named file per instance under
// .katalyst/storage/ (discovery: convention, the default), or listed
// explicitly in .katalyst/config.yaml (discovery: explicit). A storage
// instance embeds the collections it maps. The file format (yaml, json, or
// both) is set per kind in config.yaml. See
// docs/content/reference/configuration.md.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/storage/collection/query"
	"gopkg.in/yaml.v3"
)

// Dir is the per-project directory that marks a project root and holds
// its config, schemas, and collections.
const Dir = ".katalyst"

// configFile is the optional settings file inside Dir.
const configFile = "config.yaml"

// Subdirectories of Dir holding one named file per definition.
const (
	schemasSubdir = "schemas"
	storageSubdir = "storage"
)

// Discovery modes for a kind (schemas or collections).
const (
	discoveryConvention = "convention"
	discoveryExplicit   = "explicit"
)

// defaultPattern is the glob applied to a collection's directory when the
// collection does not set its own `pattern`.
const defaultPattern = "*.md"

// storageTypeFilesystem is the only backend kind with an implementation today.
const storageTypeFilesystem = "filesystem"

// knownStorageTypes is the parse-time allowlist of backend kinds. The
// implementations live in internal/storage; config validates the declared
// `type` here so a typo fails at load rather than at command time. This set
// grows alongside the internal/storage registry when a backend is added (config
// cannot import storage, which depends on config).
var knownStorageTypes = map[string]bool{
	storageTypeFilesystem: true,
}

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
	// Storage holds the configured storage instances, in name order. Each
	// instance declares its own collections.
	Storage []StorageInstance
	// Collections is the flattened view across all instances, in name order.
	// Collection names are unique project-wide (selectors carry no instance
	// qualifier), so this is the canonical lookup most callers use.
	Collections []Collection
}

// StorageInstance is one configured backend store plus the collections it maps
// onto the domain model. For StorageType filesystem, Root is a directory.
type StorageInstance struct {
	// Name is the public handle (filename stem under .katalyst/storage/, or
	// the key in the inline `storage.defs` map).
	Name string
	// Type is the backend kind, validated against knownStorageTypes.
	Type string
	// Root is the absolute, resolved instance root. Relative roots in the
	// source resolve against the repo Root.
	Root string
	// Collections this instance declares, in name order.
	Collections []Collection
}

// CheckType identifies the reusable check definition attached to a
// collection. Its string value is the `kind:` selector in YAML.
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
	Checks []checks.ConfiguredCheck
	// Query holds the resolved `item list` query behavior for this
	// collection (collection config over project config over defaults).
	Query QuerySettings
	// Storage is the name of the storage instance that declares this
	// collection.
	Storage string
	// Variants are discriminated check groups: an item runs the first
	// variant (in order) whose Where predicates it all satisfies, in
	// addition to the base Checks. Empty for a collection without variants.
	Variants []CollectionVariant
	// UseExhaustiveVariants makes an item that matches no variant a check
	// failure ("matches no variant") instead of running the base checks
	// alone. Default false.
	UseExhaustiveVariants bool
}

// CollectionVariant is one discriminated check group inside a collection. An
// item whose metadata satisfies every predicate in Where, the first such
// variant in declaration order, runs Checks on top of the collection's base
// checks. A variant's `schema:` shorthand is folded into a leading object
// check in Checks, mirroring a collection's `schema:`.
type CollectionVariant struct {
	// Where are the discriminator predicates, ANDed together. Non-empty.
	Where []query.Predicate
	// Checks run on an item routed to this variant, added to the base.
	Checks []checks.ConfiguredCheck
}

// QuerySettings configures the behavior of `item list` filtering and
// sorting. Values are resolved at load time; see the `query:` block in
// .katalyst/config.yaml (project default) and a collection's file (override).
type QuerySettings struct {
	// FilterTypeMismatch decides what happens when a --filter comparison
	// hits an incompatible type: "skip" (item does not match) or "error".
	FilterTypeMismatch string
	// SortMissing decides where items lacking the sort key land: "last"
	// (end, both directions) or "lowest" (below any present value).
	SortMissing string
}

// Built-in query defaults, used when neither the collection nor the
// project config sets a value.
const (
	defaultFilterTypeMismatch = "skip"
	defaultSortMissing        = "last"
)

// rawConfig mirrors .katalyst/config.yaml. Both blocks are optional; an
// absent file (or an absent block) means convention discovery with the
// default YAML format.
type rawConfig struct {
	Schemas rawSchemaKind  `yaml:"schemas"`
	Storage rawStorageKind `yaml:"storage"`
	Query   *rawQuery      `yaml:"query"`
}

// rawQuery mirrors a `query:` block. A nil pointer (or a nil field within)
// means "unset" so resolution can fall through to the next level.
type rawQuery struct {
	FilterTypeMismatch string `yaml:"filterTypeMismatch"`
	SortMissing        string `yaml:"sortMissing"`
}

// rawSchemaKind configures how schemas are discovered. Defs is consulted
// only when Discovery is "explicit" (name → file path).
type rawSchemaKind struct {
	Discovery string            `yaml:"discovery"`
	Format    string            `yaml:"format"`
	Defs      map[string]string `yaml:"defs"`
}

// rawStorageKind configures how storage instances are discovered. Defs is
// consulted only when Discovery is "explicit" (name → instance).
type rawStorageKind struct {
	Discovery string                        `yaml:"discovery"`
	Format    string                        `yaml:"format"`
	Defs      map[string]rawStorageInstance `yaml:"defs"`
}

// rawStorageInstance mirrors one storage instance: its backend type, its root,
// and the collections it declares (name → definition).
type rawStorageInstance struct {
	Type        string                   `yaml:"type"`
	Root        string                   `yaml:"root"`
	Collections map[string]rawCollection `yaml:"collections"`
}

type rawCollection struct {
	Path                  string       `yaml:"path"`
	Pattern               string       `yaml:"pattern"`
	Schema                string       `yaml:"schema"`
	Checks                []rawCheck   `yaml:"checks"`
	Query                 *rawQuery    `yaml:"query"`
	Variants              []rawVariant `yaml:"variants"`
	UseExhaustiveVariants bool         `yaml:"useExhaustiveVariants"`
}

// rawVariant mirrors one entry of a collection's `variants:` list: a `when`
// discriminator plus the schema/checks to add for matching items.
type rawVariant struct {
	When   rawWhen    `yaml:"when"`
	Schema string     `yaml:"schema"`
	Checks []rawCheck `yaml:"checks"`
}

// rawWhen is a variant discriminator: a list of `item list --filter` predicate
// strings. It accepts three YAML shapes that all desugar to that list:
//
//	when: "kind=section"             # one predicate
//	when: ["kind=section", "w>1"]    # a list of predicates
//	when: { where: [ ... ] }         # the explicit block form
type rawWhen []string

// UnmarshalYAML accepts the scalar, sequence, and {where: [...]} forms.
func (w *rawWhen) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		*w = rawWhen{s}
	case yaml.SequenceNode:
		var ss []string
		if err := value.Decode(&ss); err != nil {
			return err
		}
		*w = rawWhen(ss)
	case yaml.MappingNode:
		var block struct {
			Where []string `yaml:"where"`
		}
		if err := value.Decode(&block); err != nil {
			return err
		}
		*w = rawWhen(block.Where)
	default:
		return fmt.Errorf("invalid when: expected a string, a list, or {where: [...]}")
	}
	return nil
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

	// node is the raw YAML node for this entry, retained so a distributed check
	// parser can decode its own args. Captured in UnmarshalYAML.
	node *yaml.Node
}

// UnmarshalYAML decodes the entry's fields and stashes the raw node, so the
// node can travel to a check type's own parser (checks.RegisterParsed).
func (rc *rawCheck) UnmarshalYAML(value *yaml.Node) error {
	type plain rawCheck
	var p plain
	if err := value.Decode(&p); err != nil {
		return err
	}
	*rc = rawCheck(p)
	rc.node = value
	return nil
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
	if err := cfg.loadStorage(raw.Storage, raw.Query); err != nil {
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

// loadStorage populates c.Storage and the flattened c.Collections (both sorted
// by name) from either the storage directory (convention: one file per
// instance) or an explicit defs map in config.yaml. Collection names are
// validated unique across every instance.
func (c *Config) loadStorage(k rawStorageKind, projectQuery *rawQuery) error {
	discovery, err := normDiscovery(k.Discovery)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	exts, err := formatExts(k.Format)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}

	defs := map[string]rawStorageInstance{}
	if discovery == discoveryExplicit {
		if len(k.Defs) == 0 {
			return errors.New(`storage: discovery "explicit" requires a non-empty "defs" map`)
		}
		defs = k.Defs
	} else {
		found, err := scanKindDir(filepath.Join(c.Root, Dir, storageSubdir), exts)
		if err != nil {
			return fmt.Errorf("storage: %w", err)
		}
		for name, path := range found {
			src, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("storage %q: %w", name, err)
			}
			var ri rawStorageInstance
			if err := yaml.Unmarshal(src, &ri); err != nil {
				return fmt.Errorf("storage %q: %w", name, err)
			}
			defs[name] = ri
		}
	}

	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	sort.Strings(names)

	// instanceOf records which instance first claimed a collection name, so a
	// collision across instances is reported with both sides.
	instanceOf := map[string]string{}
	for _, name := range names {
		inst, err := c.buildInstance(name, defs[name], exts, projectQuery)
		if err != nil {
			return err
		}
		for _, col := range inst.Collections {
			if prev, dup := instanceOf[col.Name]; dup {
				return fmt.Errorf("collection %q is declared by two storage instances (%q and %q); collection names must be unique across the project", col.Name, prev, name)
			}
			instanceOf[col.Name] = name
			c.Collections = append(c.Collections, col)
		}
		c.Storage = append(c.Storage, inst)
	}
	sort.Slice(c.Collections, func(i, j int) bool {
		return c.Collections[i].Name < c.Collections[j].Name
	})
	return nil
}

// buildInstance turns one raw storage instance into a validated
// StorageInstance, building each of its collections against the instance root.
// Collections come from the instance's inline `collections:` block and, as an
// escape hatch for instances that outgrow inline, from one file per collection
// under .katalyst/storage/<name>/. A name declared in both places is an error.
// The instance name comes from the source (filename stem or map key), never the
// body.
func (c *Config) buildInstance(name string, ri rawStorageInstance, exts []string, projectQuery *rawQuery) (StorageInstance, error) {
	typ := ri.Type
	if typ == "" {
		typ = storageTypeFilesystem
	}
	if !knownStorageTypes[typ] {
		return StorageInstance{}, fmt.Errorf("storage %q: unknown type %q", name, ri.Type)
	}

	rootRel := ri.Root
	if rootRel == "" {
		rootRel = "."
	}
	instRoot := resolve(c.Root, rootRel)

	// Start with the inline collections, then fold in any per-collection files.
	raws := make(map[string]rawCollection, len(ri.Collections))
	for cn, rc := range ri.Collections {
		raws[cn] = rc
	}
	instDir := filepath.Join(c.Root, Dir, storageSubdir, name)
	found, err := scanKindDir(instDir, exts)
	if err != nil {
		return StorageInstance{}, fmt.Errorf("storage %q: %w", name, err)
	}
	for cn, path := range found {
		if _, dup := raws[cn]; dup {
			return StorageInstance{}, fmt.Errorf("storage %q: collection %q is declared both inline and in a file", name, cn)
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return StorageInstance{}, fmt.Errorf("storage %q: collection %q: %w", name, cn, err)
		}
		var rc rawCollection
		if err := yaml.Unmarshal(src, &rc); err != nil {
			return StorageInstance{}, fmt.Errorf("storage %q: collection %q: %w", name, cn, err)
		}
		raws[cn] = rc
	}

	colNames := make([]string, 0, len(raws))
	for cn := range raws {
		colNames = append(colNames, cn)
	}
	sort.Strings(colNames)

	cols := make([]Collection, 0, len(colNames))
	for _, cn := range colNames {
		col, err := c.buildCollection(cn, raws[cn], instRoot, name, projectQuery)
		if err != nil {
			return StorageInstance{}, err
		}
		cols = append(cols, col)
	}
	return StorageInstance{Name: name, Type: typ, Root: instRoot, Collections: cols}, nil
}

// buildCollection turns one raw collection definition into a validated
// Collection, resolving its directory against the owning instance's root. The
// name comes from the source (map key), never the file body.
func (c *Config) buildCollection(name string, rc rawCollection, instRoot, instName string, projectQuery *rawQuery) (Collection, error) {
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

	cks, err := c.buildChecks(fmt.Sprintf("collection %q", name), rc.Schema, rc.Checks)
	if err != nil {
		return Collection{}, err
	}

	variants, err := c.buildVariants(name, rc.Variants)
	if err != nil {
		return Collection{}, err
	}

	if len(cks) == 0 && len(variants) == 0 {
		return Collection{}, fmt.Errorf("collection %q: no checks configured (set schema, checks, or variants)", name)
	}

	schemaName := ""
	for _, ch := range cks {
		if ch.Kind == checks.CheckObject {
			schemaName = ch.Schema
			break
		}
	}

	qs, err := resolveQuery(name, rc.Query, projectQuery)
	if err != nil {
		return Collection{}, err
	}

	return Collection{
		Name:                  name,
		Path:                  dirRel,
		Dir:                   resolve(instRoot, dirRel),
		Pattern:               pattern,
		Schema:                schemaName,
		Checks:                cks,
		Query:                 qs,
		Storage:               instName,
		Variants:              variants,
		UseExhaustiveVariants: rc.UseExhaustiveVariants,
	}, nil
}

// buildChecks folds an optional schema name into a leading object check and
// normalizes the remaining raw checks. errCtx prefixes any error (e.g.
// `collection "books"` or `collection "books": variants[0]`).
func (c *Config) buildChecks(errCtx, schema string, raws []rawCheck) ([]checks.ConfiguredCheck, error) {
	out := make([]checks.ConfiguredCheck, 0, len(raws)+1)
	if schema != "" {
		if _, ok := c.Schemas[schema]; !ok {
			return nil, fmt.Errorf("%s: unknown schema %q", errCtx, schema)
		}
		out = append(out, checks.ConfiguredCheck{Kind: checks.CheckObject, Schema: schema})
	}
	for j, raw := range raws {
		kind := checks.CheckType(strings.TrimSpace(raw.Kind))
		if kind == checks.CheckObject {
			// An explicit `kind: object` names a schema, validated here because
			// the loader owns schema resolution; the engine builds it.
			if raw.Schema == "" {
				return nil, fmt.Errorf("%s: checks[%d]: object check requires \"schema\"", errCtx, j)
			}
			if _, ok := c.Schemas[raw.Schema]; !ok {
				return nil, fmt.Errorf("%s: checks[%d]: unknown schema %q", errCtx, j, raw.Schema)
			}
			out = append(out, checks.ConfiguredCheck{Kind: checks.CheckObject, Schema: raw.Schema})
			continue
		}
		args, err := checks.Parse(kind, raw.node)
		if err != nil {
			return nil, fmt.Errorf("%s: checks[%d]: %w", errCtx, j, err)
		}
		out = append(out, checks.ConfiguredCheck{Kind: kind, Args: args})
	}
	return out, nil
}

// buildVariants parses and validates a collection's variants: each `when`
// becomes a non-empty list of predicates, and each variant's schema/checks are
// built like the collection's own (the schema folds into a leading object
// check). A variant may add no checks (an exemption).
func (c *Config) buildVariants(name string, raws []rawVariant) ([]CollectionVariant, error) {
	if len(raws) == 0 {
		return nil, nil
	}
	variants := make([]CollectionVariant, 0, len(raws))
	for i, rv := range raws {
		if len(rv.When) == 0 {
			return nil, fmt.Errorf("collection %q: variants[%d]: when requires at least one predicate", name, i)
		}
		preds := make([]query.Predicate, 0, len(rv.When))
		for k, expr := range rv.When {
			p, err := query.ParseFilter(expr)
			if err != nil {
				return nil, fmt.Errorf("collection %q: variants[%d]: when[%d]: %w", name, i, k, err)
			}
			preds = append(preds, p)
		}
		vchecks, err := c.buildChecks(fmt.Sprintf("collection %q: variants[%d]", name, i), rv.Schema, rv.Checks)
		if err != nil {
			return nil, err
		}
		variants = append(variants, CollectionVariant{Where: preds, Checks: vchecks})
	}
	return variants, nil
}

// resolveQuery merges a collection's query block over the project's over
// the built-in defaults, key by key, then validates each value. An unset
// key (empty string at a level) falls through to the next.
func resolveQuery(name string, collQuery, projectQuery *rawQuery) (QuerySettings, error) {
	q := QuerySettings{
		FilterTypeMismatch: defaultFilterTypeMismatch,
		SortMissing:        defaultSortMissing,
	}
	for _, raw := range []*rawQuery{projectQuery, collQuery} {
		if raw == nil {
			continue
		}
		if raw.FilterTypeMismatch != "" {
			q.FilterTypeMismatch = raw.FilterTypeMismatch
		}
		if raw.SortMissing != "" {
			q.SortMissing = raw.SortMissing
		}
	}
	switch q.FilterTypeMismatch {
	case "skip", "error":
	default:
		return QuerySettings{}, fmt.Errorf("collection %q: unknown filterTypeMismatch %q (want skip or error)", name, q.FilterTypeMismatch)
	}
	switch q.SortMissing {
	case "last", "lowest":
	default:
		return QuerySettings{}, fmt.Errorf("collection %q: unknown sortMissing %q (want last or lowest)", name, q.SortMissing)
	}
	return q, nil
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

// HasCollectionChecks reports whether the collection configures any
// collection-scoped check.
func (c Collection) HasCollectionChecks() bool {
	for _, cc := range c.Checks {
		if checks.CollectionScoped(cc.Kind) {
			return true
		}
	}
	return false
}
