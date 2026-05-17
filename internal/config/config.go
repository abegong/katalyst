// Package config loads katabridge.yaml from the nearest ancestor
// directory and answers two questions:
//
//  1. Which schemas exist (by name → absolute file path)?
//  2. For a given markdown file path, which schema name applies?
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

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

// Filename is the fixed config name we look for during ascent.
const Filename = "katabridge.yaml"

// ErrNotFound is returned when no katabridge.yaml is present in the
// starting directory or any of its ancestors.
var ErrNotFound = errors.New("config: katabridge.yaml not found")

// Config is the parsed, validated, root-relative-resolved configuration.
//
// Schemas maps the schema name to an *absolute* file path; relative
// paths in the source YAML are resolved against Root.
//
// Rules retains the source order (first match wins).
type Config struct {
	// Root is the absolute directory containing katabridge.yaml.
	Root string
	// Schemas is name → absolute path.
	Schemas map[string]string
	// Rules in source order.
	Rules []Rule
}

// Rule binds a glob pattern to a named schema.
type Rule struct {
	// Paths is a doublestar-syntax glob, relative to Root.
	Paths string
	// Schema is a key into Config.Schemas.
	Schema string
}

// rawConfig mirrors the on-disk YAML shape, before validation.
type rawConfig struct {
	Schemas map[string]string `yaml:"schemas"`
	Rules   []rawRule         `yaml:"rules"`
}

type rawRule struct {
	Paths  string `yaml:"paths"`
	Schema string `yaml:"schema"`
}

// Load finds katabridge.yaml by walking upward from start and parses
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
		if _, ok := cfg.Schemas[r.Schema]; !ok {
			return nil, fmt.Errorf("rules[%d]: unknown schema %q", i, r.Schema)
		}
		if r.Paths == "" {
			return nil, fmt.Errorf("rules[%d]: empty paths pattern", i)
		}
		cfg.Rules = append(cfg.Rules, Rule{Paths: r.Paths, Schema: r.Schema})
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
	rel, err := relativeToRoot(c.Root, filePath)
	if err != nil {
		return "", false
	}
	for _, r := range c.Rules {
		ok, err := doublestar.Match(r.Paths, rel)
		if err == nil && ok {
			return r.Schema, true
		}
	}
	return "", false
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
