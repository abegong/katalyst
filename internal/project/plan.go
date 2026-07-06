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

type planRoot struct {
	Root      string
	ConfigDir string
}

func BuildPlan(opts PlanOptions) (*Plan, error) {
	selected, err := rootForPlan(opts)
	if err != nil {
		return nil, err
	}
	cfg, err := LoadRootWithConfigDir(selected.Root, selected.ConfigDir)
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
		if active {
			childRoot := filepath.Join(cfg.Root, filepath.FromSlash(delegate.Path))
			childConfigDir := filepath.Join(childRoot, filepath.FromSlash(delegate.Config))
			child, err := LoadRootWithConfigDir(childRoot, childConfigDir)
			if err != nil {
				return nil, fmt.Errorf("nested config %s/%s: %w", delegate.Path, delegate.Config, err)
			}
			pd.Config = child
		}
		plan.Delegates = append(plan.Delegates, pd)
	}
	return plan, nil
}

func rootForPlan(opts PlanOptions) (planRoot, error) {
	if opts.ConfigPath != "" {
		return rootFromConfigPath(opts.ConfigPath)
	}
	if opts.ProjectDir != "" {
		root, err := FindRoot(opts.ProjectDir)
		return planRoot{Root: root}, err
	}
	start := opts.Start
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return planRoot{}, err
		}
	}
	root, err := FindRoot(start)
	return planRoot{Root: root}, err
}

func rootFromConfigPath(path string) (planRoot, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return planRoot{}, fmt.Errorf("resolve config path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return planRoot{}, fmt.Errorf("--config: %w", err)
	}
	if info.IsDir() {
		if filepath.Base(abs) == Dir {
			return planRoot{Root: filepath.Dir(abs), ConfigDir: abs}, nil
		}
		if ok, err := dirExists(filepath.Join(abs, Dir)); err != nil {
			return planRoot{}, fmt.Errorf("--config: %w", err)
		} else if ok {
			return planRoot{Root: abs, ConfigDir: filepath.Join(abs, Dir)}, nil
		}
		if looksLikeConfigDir(abs) {
			return planRoot{Root: filepath.Dir(abs), ConfigDir: abs}, nil
		}
		return planRoot{}, errors.New("--config: expected a project root, config directory, or config.yaml file")
	}
	if filepath.Base(abs) == configFile {
		configDir := filepath.Dir(abs)
		return planRoot{Root: filepath.Dir(configDir), ConfigDir: configDir}, nil
	}
	return planRoot{}, errors.New("--config: expected a project root, config directory, or config.yaml file")
}

func looksLikeConfigDir(dir string) bool {
	for _, name := range []string{configFile, schemasSubdir, basesSubdir, storageSubdir} {
		if ok, err := pathExists(filepath.Join(dir, name)); err == nil && ok {
			return true
		}
	}
	return false
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
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
