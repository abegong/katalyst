// loader.go holds the project loader: it reads a project's .katalyst/ directory
// and answers two questions:
//
//  1. Which schemas exist (by name → absolute file path)?
//  2. Which bases exist, what collections does each declare, and
//     what checks does each collection run?
//
// A project is the nearest ancestor directory that contains a .katalyst/
// subdirectory. Schemas are defined one named file per definition under
// .katalyst/schemas/; bases are defined one named file per definition under
// .katalyst/bases/ (discovery: convention, the default), or listed explicitly
// in .katalyst/config.yaml (discovery: explicit). A base embeds the collections
// it maps. Legacy projects may still use storage: and .katalyst/storage/. The
// file format (yaml, json, or both) is set per kind in config.yaml. See
// docs/content/reference/configuration.md.
package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/abegong/katalyst/internal/storage"
	"github.com/abegong/katalyst/internal/storage/collection"
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
	basesSubdir   = "bases"
	storageSubdir = "storage"
)

// Discovery modes for a kind (schemas or collections).
const (
	discoveryConvention = "convention"
	discoveryExplicit   = "explicit"
)

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
	// Bases holds the configured bases, in name order. Each base declares its
	// own collections.
	Bases []BaseInstance
	// Collections is the flattened view across all bases, in name order.
	// Collection names are unique project-wide (selectors carry no base
	// qualifier), so this is the canonical lookup most callers use.
	Collections []Collection
}

// BaseInstance is one configured backend store plus the collections it maps
// onto the domain model. For BaseType filesystem, Root is a directory.
type BaseInstance struct {
	// Name is the public handle (filename stem under .katalyst/bases/, or
	// the key in the inline `bases.defs` map).
	Name string
	// Type is the backend kind, validated against the storage registry
	// (storage.Known).
	Type string
	// Root is the absolute, resolved base root. Relative roots in the
	// source resolve against the repo Root.
	Root string
	// Collections this base declares, in name order.
	Collections []Collection
}

// Collection, CollectionVariant, and ListingDefaults live in
// internal/storage/collection (a collection is a storage concept); project
// re-exports them under their historical names so callers that load project
// config keep referring to project.Collection. The loader assembles these types
// but no longer defines them.
type (
	Collection        = collection.Collection
	CollectionVariant = collection.CollectionVariant
	ListingDefaults   = collection.ListingDefaults
)

// rawConfig mirrors .katalyst/config.yaml. Both blocks are optional; an
// absent file (or an absent block) means convention discovery with the
// default YAML format.
type rawConfig struct {
	Schemas rawSchemaKind                  `yaml:"schemas"`
	Bases   *rawBaseKind                   `yaml:"bases"`
	Storage *rawBaseKind                   `yaml:"storage"`
	Listing *collection.RawListingDefaults `yaml:"listing"`
	Query   *collection.RawListingDefaults `yaml:"query"`
}

// rawSchemaKind configures how schemas are discovered. Defs is consulted
// only when Discovery is "explicit" (name → file path).
type rawSchemaKind struct {
	Discovery string            `yaml:"discovery"`
	Format    string            `yaml:"format"`
	Defs      map[string]string `yaml:"defs"`
}

// rawBaseKind configures how bases are discovered. Defs is consulted only when
// Discovery is "explicit" (name → base).
type rawBaseKind struct {
	Discovery string                     `yaml:"discovery"`
	Format    string                     `yaml:"format"`
	Defs      map[string]rawBaseInstance `yaml:"defs"`
}

// rawBaseInstance mirrors one base: its backend type, its root, and the
// collections it declares (name → definition). The collection mirror lives with
// the Collection type in internal/storage/collection.
type rawBaseInstance struct {
	Type        string                              `yaml:"type"`
	Root        string                              `yaml:"root"`
	Collections map[string]collection.RawCollection `yaml:"collections"`
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
	if raw.Query != nil {
		return nil, errors.New("config: query is no longer a config block; use listing")
	}
	if err := cfg.loadBases(raw.Bases, raw.Storage, raw.Listing); err != nil {
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

// loadBases populates c.Bases and the flattened c.Collections (both sorted by
// name) from either the bases directory (convention: one file per base) or an
// explicit defs map in config.yaml. Collection names are validated unique
// across every base. The legacy storage block and directory stay readable, but
// cannot be mixed with the new bases form.
func (c *Config) loadBases(bases, legacy *rawBaseKind, projectListing *collection.RawListingDefaults) error {
	if bases != nil && legacy != nil {
		return errors.New("config: use bases, not both bases and storage")
	}
	label := "bases"
	k := rawBaseKind{}
	if bases != nil {
		k = *bases
	} else if legacy != nil {
		k = *legacy
		label = "storage"
	}

	discovery, err := normDiscovery(k.Discovery)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	exts, err := formatExts(k.Format)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}

	baseSubdir, err := c.baseSubdir(label)
	if err != nil {
		return err
	}

	defs := map[string]rawBaseInstance{}
	if discovery == discoveryExplicit {
		if len(k.Defs) == 0 {
			return fmt.Errorf(`%s: discovery "explicit" requires a non-empty "defs" map`, label)
		}
		defs = k.Defs
	} else {
		found, err := scanKindDir(filepath.Join(c.Root, Dir, baseSubdir), exts)
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		for name, path := range found {
			src, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("%s %q: %w", label, name, err)
			}
			var ri rawBaseInstance
			if err := yaml.Unmarshal(src, &ri); err != nil {
				return fmt.Errorf("%s %q: %w", label, name, err)
			}
			defs[name] = ri
		}
	}

	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	sort.Strings(names)

	// baseOf records which base first claimed a collection name, so a collision
	// across bases is reported with both sides.
	baseOf := map[string]string{}
	for _, name := range names {
		inst, err := c.buildInstance(name, defs[name], exts, projectListing, baseSubdir, label)
		if err != nil {
			return err
		}
		for _, col := range inst.Collections {
			if prev, dup := baseOf[col.Name]; dup {
				return fmt.Errorf("collection %q is declared by two bases (%q and %q); collection names must be unique across the project", col.Name, prev, name)
			}
			baseOf[col.Name] = name
			c.Collections = append(c.Collections, col)
		}
		c.Bases = append(c.Bases, inst)
	}
	sort.Slice(c.Collections, func(i, j int) bool {
		return c.Collections[i].Name < c.Collections[j].Name
	})
	return nil
}

// baseSubdir chooses the directory that holds base definition files. New config
// uses .katalyst/bases/. Legacy .katalyst/storage/ remains readable, but the
// two directories cannot be mixed.
func (c *Config) baseSubdir(label string) (string, error) {
	hasBases, err := dirExists(filepath.Join(c.Root, Dir, basesSubdir))
	if err != nil {
		return "", fmt.Errorf("bases: %w", err)
	}
	hasStorage, err := dirExists(filepath.Join(c.Root, Dir, storageSubdir))
	if err != nil {
		return "", fmt.Errorf("storage: %w", err)
	}
	if hasBases && hasStorage {
		return "", errors.New("config: use .katalyst/bases, not both .katalyst/bases and .katalyst/storage")
	}
	if hasBases {
		return basesSubdir, nil
	}
	if hasStorage {
		return storageSubdir, nil
	}
	if label == "storage" {
		return storageSubdir, nil
	}
	return basesSubdir, nil
}

// buildInstance turns one raw base into a validated
// BaseInstance, building each of its collections against the base root.
// Collections come from the base's inline `collections:` block and, as an
// escape hatch for bases that outgrow inline, from one file per collection
// under .katalyst/bases/<name>/. A name declared in both places is an error.
// The base name comes from the source (filename stem or map key), never the body.
func (c *Config) buildInstance(name string, ri rawBaseInstance, exts []string, projectListing *collection.RawListingDefaults, baseSubdir, label string) (BaseInstance, error) {
	typ := ri.Type
	if typ == "" {
		typ = string(storage.Filesystem)
	}
	if !storage.Known(storage.BaseType(typ)) {
		return BaseInstance{}, fmt.Errorf("%s %q: unknown type %q", label, name, ri.Type)
	}

	rootRel := ri.Root
	if rootRel == "" {
		rootRel = "."
	}
	instRoot := resolve(c.Root, rootRel)

	// Start with the inline collections, then fold in any per-collection files.
	raws := make(map[string]collection.RawCollection, len(ri.Collections))
	for cn, rc := range ri.Collections {
		raws[cn] = rc
	}
	instDir := filepath.Join(c.Root, Dir, baseSubdir, name)
	found, err := scanKindDir(instDir, exts)
	if err != nil {
		return BaseInstance{}, fmt.Errorf("%s %q: %w", label, name, err)
	}
	for cn, path := range found {
		if _, dup := raws[cn]; dup {
			return BaseInstance{}, fmt.Errorf("%s %q: collection %q is declared both inline and in a file", label, name, cn)
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return BaseInstance{}, fmt.Errorf("%s %q: collection %q: %w", label, name, cn, err)
		}
		var rc collection.RawCollection
		if err := yaml.Unmarshal(src, &rc); err != nil {
			return BaseInstance{}, fmt.Errorf("%s %q: collection %q: %w", label, name, cn, err)
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
		col, err := collection.Build(collection.BuildInput{
			Name:           cn,
			Raw:            raws[cn],
			BaseRoot:       instRoot,
			BaseName:       name,
			ProjectListing: projectListing,
			SchemaKnown:    c.schemaKnown,
		})
		if err != nil {
			return BaseInstance{}, err
		}
		cols = append(cols, col)
	}
	return BaseInstance{Name: name, Type: typ, Root: instRoot, Collections: cols}, nil
}

func dirExists(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// schemaKnown reports whether a schema name is defined. The collection builder
// validates schema references through this predicate so it never needs the
// loader's name→path map directly.
func (c *Config) schemaKnown(name string) bool {
	_, ok := c.Schemas[name]
	return ok
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
