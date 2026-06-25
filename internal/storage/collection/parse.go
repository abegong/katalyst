package collection

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/storage/collection/predicate"
	"gopkg.in/yaml.v3"
)

// Collection is a named group of items backed by one storage backend.
type Collection struct {
	// Name is the public handle (the key in the config `collections:` map).
	Name string
	// Path is the directory, relative to Root, as written in the config.
	Path string
	// Dir is the absolute directory (Root + Path).
	Dir string
	// StorageType is the backend kind of the declaring storage instance.
	StorageType string
	// Table is the SQLite table backing this collection. Empty for filesystem.
	Table string
	// IDColumn is the SQLite column that provides item identity.
	IDColumn string
	// Attributes maps item attribute names to SQLite column captures.
	Attributes map[string]AttributeCapture
	// ContentKind is the optional content shape mapped from a SQLite column.
	ContentKind string
	// ContentColumn is the optional SQLite column that provides content bytes.
	ContentColumn string
	// Pattern is the filename glob for items (default "*.md").
	Pattern string
	// Schema is the object-schema name associated with the collection, or
	// "" when the collection is configured with explicit checks only. It
	// mirrors the first object check's schema, for display.
	Schema string
	// Checks to run against each item.
	Checks []checks.ConfiguredCheck
	// ListingDefaults holds the resolved `item list` behavior for this
	// collection (collection config over project config over defaults).
	ListingDefaults ListingDefaults
	// Base is the name of the base that declares this
	// collection.
	Base string
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
	Where []predicate.Predicate
	// Checks run on an item routed to this variant, added to the base.
	Checks []checks.ConfiguredCheck
}

// ListingDefaults configures the behavior of `item list` filtering and
// sorting. Values are resolved at load time; see the `listing:` block in
// .katalyst/config.yaml (project default) and a collection's file (override).
type ListingDefaults struct {
	// FilterTypeMismatch decides what happens when a --filter comparison
	// hits an incompatible type: "skip" (item does not match) or "error".
	FilterTypeMismatch string
	// SortMissing decides where items lacking the sort key land: "last"
	// (end, both directions) or "lowest" (below any present value).
	SortMissing string
}

// AttributeCapture describes how one item attribute is captured from a SQLite
// row. A simple attribute comes from one column; a structured attribute comes
// from a set of columns and becomes an object with one field per entry.
type AttributeCapture struct {
	Column  string
	Columns map[string]string
}

// UnmarshalYAML accepts both shorthand and explicit attribute capture forms:
//
//	title: title
//	title: {column: title}
//	author: {columns: {first: author_first, last: author_last}}
func (a *AttributeCapture) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		a.Column = s
		return nil
	case yaml.MappingNode:
		var raw struct {
			Column  string            `yaml:"column"`
			Columns map[string]string `yaml:"columns"`
		}
		if err := value.Decode(&raw); err != nil {
			return err
		}
		a.Column = raw.Column
		a.Columns = raw.Columns
		return nil
	default:
		return fmt.Errorf("invalid attribute capture: expected a column string or mapping")
	}
}

// ContentConfig describes an optional content shape captured from one SQLite
// column. `body:` is still accepted as a deprecated compatibility alias.
type ContentConfig struct {
	Kind   string `yaml:"kind"`
	Column string `yaml:"column"`
}

// Built-in defaults, used when neither the collection nor the project config
// sets a value.
const (
	defaultPattern            = "*.md"
	defaultFilterTypeMismatch = "skip"
	defaultSortMissing        = "last"
)

// RawCollection mirrors one collection definition in YAML. The loader
// unmarshals it (inline under a base, or one file per collection)
// and hands it to Build.
type RawCollection struct {
	Path                  string                      `yaml:"path"`
	Pattern               string                      `yaml:"pattern"`
	Table                 string                      `yaml:"table"`
	ID                    string                      `yaml:"id"`
	Attributes            map[string]AttributeCapture `yaml:"attributes"`
	Content               ContentConfig               `yaml:"content"`
	Body                  string                      `yaml:"body"`
	Schema                string                      `yaml:"schema"`
	Checks                []RawCheck                  `yaml:"checks"`
	Listing               *RawListingDefaults         `yaml:"listing"`
	Query                 *RawListingDefaults         `yaml:"query"`
	Variants              []RawVariant                `yaml:"variants"`
	UseExhaustiveVariants bool                        `yaml:"useExhaustiveVariants"`
}

// RawListingDefaults mirrors a `listing:` block. A nil pointer (or a nil field
// within) means "unset" so resolution can fall through to the next level.
type RawListingDefaults struct {
	FilterTypeMismatch string `yaml:"filterTypeMismatch"`
	SortMissing        string `yaml:"sortMissing"`
}

// RawVariant mirrors one entry of a collection's `variants:` list: a `when`
// discriminator plus the schema/checks to add for matching items.
type RawVariant struct {
	When   RawWhen    `yaml:"when"`
	Schema string     `yaml:"schema"`
	Checks []RawCheck `yaml:"checks"`
}

// RawWhen is a variant discriminator: a list of `item list --filter` predicate
// strings. It accepts three YAML shapes that all desugar to that list:
//
//	when: "kind=section"             # one predicate
//	when: ["kind=section", "w>1"]    # a list of predicates
//	when: { where: [ ... ] }         # the explicit block form
type RawWhen []string

// UnmarshalYAML accepts the scalar, sequence, and {where: [...]} forms.
func (w *RawWhen) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		*w = RawWhen{s}
	case yaml.SequenceNode:
		var ss []string
		if err := value.Decode(&ss); err != nil {
			return err
		}
		*w = RawWhen(ss)
	case yaml.MappingNode:
		var block struct {
			Where []string `yaml:"where"`
		}
		if err := value.Decode(&block); err != nil {
			return err
		}
		*w = RawWhen(block.Where)
	default:
		return fmt.Errorf("invalid when: expected a string, a list, or {where: [...]}")
	}
	return nil
}

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

	// node is the raw YAML node for this entry, retained so a distributed check
	// parser can decode its own args. Captured in UnmarshalYAML.
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
// node can travel to a check type's own parser (checks.RegisterParsed).
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

// BuildInput carries everything Build needs to validate and resolve one
// collection: its raw definition and name, the owning base's root
// and name, the project-level listing defaults, and a predicate that reports
// whether a schema name is defined (schema resolution belongs to the loader).
type BuildInput struct {
	Name           string
	Raw            RawCollection
	StorageType    string
	BaseRoot       string
	BaseName       string
	ProjectListing *RawListingDefaults
	SchemaKnown    func(string) bool
}

// Build turns one raw collection definition into a validated Collection,
// resolving its directory against the owning base's root. The name comes
// from the source (map key), never the file body.
func Build(in BuildInput) (Collection, error) {
	storageType := in.StorageType
	if storageType == "" {
		storageType = "filesystem"
	}

	dirRel := in.Raw.Path
	if dirRel == "" && storageType == "filesystem" {
		// A collection without an explicit path defaults to a directory
		// named after the collection itself.
		dirRel = in.Name
	}
	pattern := in.Raw.Pattern
	if pattern == "" {
		pattern = defaultPattern
	}

	cks, err := buildChecks(fmt.Sprintf("collection %q", in.Name), in.Raw.Schema, in.Raw.Checks, in.SchemaKnown)
	if err != nil {
		return Collection{}, err
	}

	variants, err := buildVariants(in.Name, in.Raw.Variants, in.SchemaKnown)
	if err != nil {
		return Collection{}, err
	}

	if len(cks) == 0 && len(variants) == 0 {
		return Collection{}, fmt.Errorf("collection %q: no checks configured (set schema, checks, or variants)", in.Name)
	}
	if storageType == "sqlite" {
		if in.Raw.Table == "" {
			return Collection{}, fmt.Errorf("collection %q: sqlite collection requires \"table\"", in.Name)
		}
		if in.Raw.ID == "" {
			return Collection{}, fmt.Errorf("collection %q: sqlite collection requires \"id\"", in.Name)
		}
		if err := validateAttributeCaptures(in.Name, in.Raw.Attributes); err != nil {
			return Collection{}, err
		}
		if err := rejectUnsupportedSQLiteChecks(in.Name, cks, variants); err != nil {
			return Collection{}, err
		}
	}

	schemaName := ""
	for _, ch := range cks {
		if ch.Kind == checks.CheckObject {
			schemaName = ch.Schema
			break
		}
	}

	if in.Raw.Query != nil {
		return Collection{}, fmt.Errorf("collection %q: query is no longer a config block; use listing", in.Name)
	}

	contentKind, contentColumn, err := resolveContentConfig(in.Name, in.Raw.Content, in.Raw.Body)
	if err != nil {
		return Collection{}, err
	}

	ld, err := resolveListingDefaults(in.Name, in.Raw.Listing, in.ProjectListing)
	if err != nil {
		return Collection{}, err
	}

	return Collection{
		Name:                  in.Name,
		Path:                  dirRel,
		Dir:                   resolveDir(in.BaseRoot, dirRel),
		StorageType:           storageType,
		Table:                 in.Raw.Table,
		IDColumn:              in.Raw.ID,
		Attributes:            in.Raw.Attributes,
		ContentKind:           contentKind,
		ContentColumn:         contentColumn,
		Pattern:               pattern,
		Schema:                schemaName,
		Checks:                cks,
		ListingDefaults:       ld,
		Base:                  in.BaseName,
		Variants:              variants,
		UseExhaustiveVariants: in.Raw.UseExhaustiveVariants,
	}, nil
}

func validateAttributeCaptures(name string, attrs map[string]AttributeCapture) error {
	for attr, capture := range attrs {
		if strings.TrimSpace(attr) == "" {
			return fmt.Errorf("collection %q: attributes contains an empty name", name)
		}
		hasColumn := capture.Column != ""
		hasColumns := len(capture.Columns) > 0
		if hasColumn == hasColumns {
			return fmt.Errorf("collection %q: attribute %q must set exactly one of column or columns", name, attr)
		}
		if hasColumn {
			continue
		}
		for field, col := range capture.Columns {
			if strings.TrimSpace(field) == "" {
				return fmt.Errorf("collection %q: attribute %q contains an empty field name", name, attr)
			}
			if strings.TrimSpace(col) == "" {
				return fmt.Errorf("collection %q: attribute %q field %q has an empty column", name, attr, field)
			}
		}
	}
	return nil
}

func resolveContentConfig(name string, content ContentConfig, bodyAlias string) (string, string, error) {
	if bodyAlias != "" && content.Column != "" {
		return "", "", fmt.Errorf("collection %q: use content.column or body, not both", name)
	}
	if bodyAlias != "" {
		return "markdown", bodyAlias, nil
	}
	if content.Kind == "" && content.Column == "" {
		return "", "", nil
	}
	if content.Column == "" {
		return "", "", fmt.Errorf("collection %q: content requires column", name)
	}
	kind := content.Kind
	if kind == "" {
		kind = "text"
	}
	if kind != "text" && kind != "markdown" {
		return "", "", fmt.Errorf("collection %q: content kind must be text or markdown (got %q)", name, kind)
	}
	return kind, content.Column, nil
}

func rejectUnsupportedSQLiteChecks(name string, base []checks.ConfiguredCheck, variants []CollectionVariant) error {
	checkSet := append([]checks.ConfiguredCheck{}, base...)
	for _, v := range variants {
		checkSet = append(checkSet, v.Checks...)
	}
	for _, ch := range checkSet {
		desc, ok := checks.DescriptorFor(ch.Kind)
		if !ok {
			continue
		}
		if desc.Family == "fileSystem" {
			return fmt.Errorf("collection %q: sqlite storage does not support filesystem check %q", name, ch.Kind)
		}
	}
	return nil
}

// buildChecks folds an optional schema name into a leading object check and
// normalizes the remaining raw checks. errCtx prefixes any error (e.g.
// `collection "books"` or `collection "books": variants[0]`).
func buildChecks(errCtx, schema string, raws []RawCheck, schemaKnown func(string) bool) ([]checks.ConfiguredCheck, error) {
	out := make([]checks.ConfiguredCheck, 0, len(raws)+1)
	if schema != "" {
		if !schemaKnown(schema) {
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
			if !schemaKnown(raw.Schema) {
				return nil, fmt.Errorf("%s: checks[%d]: unknown schema %q", errCtx, j, raw.Schema)
			}
			if raw.Field != "" {
				return nil, fmt.Errorf("%s: checks[%d]: object check does not support \"field\"", errCtx, j)
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
func buildVariants(name string, raws []RawVariant, schemaKnown func(string) bool) ([]CollectionVariant, error) {
	if len(raws) == 0 {
		return nil, nil
	}
	variants := make([]CollectionVariant, 0, len(raws))
	for i, rv := range raws {
		if len(rv.When) == 0 {
			return nil, fmt.Errorf("collection %q: variants[%d]: when requires at least one predicate", name, i)
		}
		preds := make([]predicate.Predicate, 0, len(rv.When))
		for k, expr := range rv.When {
			p, err := predicate.Parse(expr)
			if err != nil {
				return nil, fmt.Errorf("collection %q: variants[%d]: when[%d]: %w", name, i, k, err)
			}
			preds = append(preds, p)
		}
		vchecks, err := buildChecks(fmt.Sprintf("collection %q: variants[%d]", name, i), rv.Schema, rv.Checks, schemaKnown)
		if err != nil {
			return nil, err
		}
		variants = append(variants, CollectionVariant{Where: preds, Checks: vchecks})
	}
	return variants, nil
}

// resolveListingDefaults merges a collection's listing block over the project's over
// the built-in defaults, key by key, then validates each value. An unset
// key (empty string at a level) falls through to the next.
func resolveListingDefaults(name string, collListing, projectListing *RawListingDefaults) (ListingDefaults, error) {
	ld := ListingDefaults{
		FilterTypeMismatch: defaultFilterTypeMismatch,
		SortMissing:        defaultSortMissing,
	}
	for _, raw := range []*RawListingDefaults{projectListing, collListing} {
		if raw == nil {
			continue
		}
		if raw.FilterTypeMismatch != "" {
			ld.FilterTypeMismatch = raw.FilterTypeMismatch
		}
		if raw.SortMissing != "" {
			ld.SortMissing = raw.SortMissing
		}
	}
	switch ld.FilterTypeMismatch {
	case "skip", "error":
	default:
		return ListingDefaults{}, fmt.Errorf("collection %q: unknown filterTypeMismatch %q (want skip or error)", name, ld.FilterTypeMismatch)
	}
	switch ld.SortMissing {
	case "last", "lowest":
	default:
		return ListingDefaults{}, fmt.Errorf("collection %q: unknown sortMissing %q (want last or lowest)", name, ld.SortMissing)
	}
	return ld, nil
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

// resolveDir turns a collection-relative path into an absolute one. Absolute
// paths in the source pass through unchanged.
func resolveDir(root, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(root, p))
}
