package filesystemcheck

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/bmatcuk/doublestar/v4"
)

const (
	ParseFailuresError   = "error"
	ParseFailuresWarning = "warning"
)

// RawScope mirrors one filesystemChecks entry.
type RawScope struct {
	Name          string            `yaml:"name"`
	Path          string            `yaml:"path"`
	Include       []string          `yaml:"include"`
	Exclude       []string          `yaml:"exclude"`
	ParseFailures string            `yaml:"parseFailures"`
	Checks        []checks.RawCheck `yaml:"checks"`
}

// Scope is a validated filesystem check scope.
type Scope struct {
	Name          string
	Path          string
	Root          string
	Include       []string
	Exclude       []string
	ParseFailures string
	Checks        []checks.ConfiguredCheck
}

// File is one regular file under a scope root.
type File struct {
	Rel  string
	Path string
}

// Expanded is the deterministic selected and unmatched file set for a scope.
type Expanded struct {
	Selected  []File
	Unmatched []File
}

// BuildInput carries the owning base and source location for a filesystem
// scope.
type BuildInput struct {
	ErrorContext string
	Raw          RawScope
	BaseRoot     string
}

// Build validates and resolves one filesystem scope.
func Build(in BuildInput) (Scope, error) {
	raw := in.Raw
	label := in.ErrorContext
	if label == "" {
		label = "filesystemChecks"
	}
	if len(raw.Include) == 0 {
		return Scope{}, fmt.Errorf("%s: include is required", label)
	}
	if len(raw.Checks) == 0 {
		return Scope{}, fmt.Errorf("%s: checks is required", label)
	}
	parseFailures := raw.ParseFailures
	if parseFailures == "" {
		parseFailures = ParseFailuresError
	}
	switch parseFailures {
	case ParseFailuresError, ParseFailuresWarning:
	default:
		return Scope{}, fmt.Errorf("%s: unknown parseFailures %q (want error or warning)", label, raw.ParseFailures)
	}
	scopePath := raw.Path
	if scopePath == "" {
		scopePath = "."
	}
	root := resolve(in.BaseRoot, scopePath)
	name := raw.Name
	if name == "" {
		name = filepath.ToSlash(scopePath)
	}
	cks, err := checks.BuildConfigured(checks.BuildConfiguredInput{
		ErrorContext:   label,
		Raw:            raw.Checks,
		ConfigurableIn: checks.ConfigFilesystem,
		AllowObject:    false,
	})
	if err != nil {
		return Scope{}, err
	}
	return Scope{
		Name:          name,
		Path:          scopePath,
		Root:          root,
		Include:       append([]string(nil), raw.Include...),
		Exclude:       append([]string(nil), raw.Exclude...),
		ParseFailures: parseFailures,
		Checks:        cks,
	}, nil
}

// Expand walks a scope root and splits regular files into selected and
// unmatched sets. Missing roots yield empty sets.
func Expand(scope Scope) (Expanded, error) {
	info, err := os.Stat(scope.Root)
	if err != nil || !info.IsDir() {
		return Expanded{}, nil
	}
	var out Expanded
	walkErr := filepath.WalkDir(scope.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(scope.Root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		included, err := matchesAny(scope.Include, rel)
		if err != nil {
			return fmt.Errorf("include: %w", err)
		}
		excluded, err := matchesAny(scope.Exclude, rel)
		if err != nil {
			return fmt.Errorf("exclude: %w", err)
		}
		file := File{Rel: rel, Path: path}
		switch {
		case included && !excluded:
			out.Selected = append(out.Selected, file)
		case !included && !excluded:
			out.Unmatched = append(out.Unmatched, file)
		}
		return nil
	})
	if walkErr != nil {
		return Expanded{}, fmt.Errorf("filesystem scope %q: %w", scope.Name, walkErr)
	}
	sort.Slice(out.Selected, func(i, j int) bool { return out.Selected[i].Rel < out.Selected[j].Rel })
	sort.Slice(out.Unmatched, func(i, j int) bool { return out.Unmatched[i].Rel < out.Unmatched[j].Rel })
	return out, nil
}

func matchesAny(patterns []string, rel string) (bool, error) {
	for _, p := range patterns {
		ok, err := doublestar.Match(p, rel)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func resolve(root, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(root, p))
}
