package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PlanOptions struct {
	Start               string
	ConfigPath          string
	ProjectDir          string
	DisableNestedConfig bool
}

type Plan struct {
	Root      *Config
	Delegates []PlanDelegate
}

type PlanDelegate struct {
	Delegate NestedDelegate
	Config   *Config
	Active   bool
}

func BuildPlan(opts PlanOptions) (*Plan, error) {
	root, err := rootForPlan(opts)
	if err != nil {
		return nil, err
	}
	cfg, err := LoadRoot(root)
	if err != nil {
		return nil, err
	}
	plan := &Plan{Root: cfg}
	if opts.ConfigPath != "" || opts.DisableNestedConfig || cfg.NestedConfigs.Discovery == NestedDiscoveryNone {
		return plan, nil
	}

	delegates := cfg.NestedConfigs.Delegates
	if cfg.NestedConfigs.Discovery == NestedDiscoveryWalk {
		walked, err := walkDelegates(cfg.Root)
		if err != nil {
			return nil, err
		}
		delegates = mergeDelegates(delegates, walked)
	}
	for _, delegate := range delegates {
		active := delegateHasActiveAuthority(cfg.NestedConfigs, delegate)
		pd := PlanDelegate{Delegate: delegate, Active: active}
		if active || cfg.NestedConfigs.Discovery != NestedDiscoveryNone {
			child, err := LoadRoot(filepath.Join(cfg.Root, filepath.FromSlash(delegate.Path)))
			if err != nil {
				return nil, fmt.Errorf("nested config %s: %w", delegate.Path, err)
			}
			pd.Config = child
		}
		plan.Delegates = append(plan.Delegates, pd)
	}
	return plan, nil
}

func rootForPlan(opts PlanOptions) (string, error) {
	if opts.ConfigPath != "" {
		return rootFromConfigPath(opts.ConfigPath)
	}
	if opts.ProjectDir != "" {
		return FindRoot(opts.ProjectDir)
	}
	start := opts.Start
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return FindRoot(start)
}

func rootFromConfigPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("--config: %w", err)
	}
	if info.IsDir() {
		if filepath.Base(abs) == Dir {
			return filepath.Dir(abs), nil
		}
		return abs, nil
	}
	if filepath.Base(abs) == configFile && filepath.Base(filepath.Dir(abs)) == Dir {
		return filepath.Dir(filepath.Dir(abs)), nil
	}
	return "", errors.New("--config: expected a project root, .katalyst directory, or .katalyst/config.yaml")
}

func walkDelegates(root string) ([]NestedDelegate, error) {
	var delegates []NestedDelegate
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if filepath.Base(path) != Dir {
			return nil
		}
		if path == filepath.Join(root, Dir) {
			return filepath.SkipDir
		}
		parent := filepath.Dir(path)
		rel, err := filepath.Rel(root, parent)
		if err != nil {
			return err
		}
		delegates = append(delegates, NestedDelegate{
			Path:      filepath.ToSlash(rel),
			Config:    Dir,
			Authority: map[AuthoritySubsystem]AuthorityRule{},
		})
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(delegates, func(i, j int) bool {
		return delegates[i].Path < delegates[j].Path
	})
	return delegates, nil
}

func mergeDelegates(configured, walked []NestedDelegate) []NestedDelegate {
	byPath := map[string]NestedDelegate{}
	for _, d := range walked {
		byPath[d.Path] = d
	}
	for _, d := range configured {
		byPath[d.Path] = d
	}
	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	out := make([]NestedDelegate, 0, len(paths))
	for _, p := range paths {
		out = append(out, byPath[p])
	}
	return out
}

func delegateHasActiveAuthority(settings NestedConfigSettings, delegate NestedDelegate) bool {
	for _, subsystem := range authoritySubsystems {
		if settings.AuthorityFor(delegate, subsystem) != AuthorityRootNearest {
			return true
		}
	}
	for _, rule := range delegate.Authority {
		if rule.Default != "" && rule.Default != AuthorityRootNearest {
			return true
		}
		for _, policy := range rule.Kinds {
			if policy != AuthorityRootNearest {
				return true
			}
		}
		for _, policy := range rule.Families {
			if policy != AuthorityRootNearest {
				return true
			}
		}
	}
	return false
}

func DelegateContains(delegate NestedDelegate, root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == delegate.Path || strings.HasPrefix(rel, delegate.Path+"/")
}
