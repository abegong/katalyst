// loader.go holds the project loader: it reads a project's .katalyst/ directory
// and answers two questions:
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
	// Type is the backend kind, validated against the storage registry
	// (storage.Known).
	Type string
	// Root is the absolute, resolved instance root. Relative roots in the
	// source resolve against the repo Root.
	Root string
	// Collections this instance declares, in name order.
	Collections []Collection
}

// Collection, CollectionVariant, and QuerySettings live in
// internal/storage/collection (a collection is a storage concept); config
// re-exports them under their historical names so callers that load config keep
// referring to config.Collection. The loader assembles these types but no longer
// defines them.
type (
	Collection        = collection.Collection
	CollectionVariant = collection.CollectionVariant
	QuerySettings     = collection.QuerySettings
)

// rawConfig mirrors .katalyst/config.yaml. Both blocks are optional; an
// absent file (or an absent block) means convention discovery with the
// default YAML format.
type rawConfig struct {
	Schemas rawSchemaKind        `yaml:"schemas"`
	Storage rawStorageKind       `yaml:"storage"`
	Query   *collection.RawQuery `yaml:"query"`
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
// and the collections it declares (name → definition). The collection mirror
// lives with the Collection type in internal/storage/collection.
type rawStorageInstance struct {
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
func (c *Config) loadStorage(k rawStorageKind, projectQuery *collection.RawQuery) error {
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
func (c *Config) buildInstance(name string, ri rawStorageInstance, exts []string, projectQuery *collection.RawQuery) (StorageInstance, error) {
	typ := ri.Type
	if typ == "" {
		typ = string(storage.Filesystem)
	}
	if !storage.Known(storage.StorageType(typ)) {
		return StorageInstance{}, fmt.Errorf("storage %q: unknown type %q", name, ri.Type)
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
		var rc collection.RawCollection
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
		col, err := collection.Build(collection.BuildInput{
			Name:         cn,
			Raw:          raws[cn],
			InstRoot:     instRoot,
			InstName:     name,
			ProjectQuery: projectQuery,
			SchemaKnown:  c.schemaKnown,
		})
		if err != nil {
			return StorageInstance{}, err
		}
		cols = append(cols, col)
	}
	return StorageInstance{Name: name, Type: typ, Root: instRoot, Collections: cols}, nil
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
