package project

import (
	"fmt"
	"path/filepath"
	"sort"
)

type NestedConfigDiscovery string

const (
	NestedDiscoveryNone     NestedConfigDiscovery = "none"
	NestedDiscoveryExplicit NestedConfigDiscovery = "explicit"
	NestedDiscoveryWalk     NestedConfigDiscovery = "walk"
)

type AuthorityPolicy string

const (
	AuthorityRootNearest AuthorityPolicy = "root_nearest"
	AuthorityFileNearest AuthorityPolicy = "file_nearest"
	AuthorityCompose     AuthorityPolicy = "compose"
)

type AuthoritySubsystem string

const (
	AuthorityFilesystemChecks AuthoritySubsystem = "filesystemChecks"
	AuthorityCollections      AuthoritySubsystem = "collections"
	AuthorityCollectionChecks AuthoritySubsystem = "collectionChecks"
	AuthoritySchemas          AuthoritySubsystem = "schemas"
	AuthorityFix              AuthoritySubsystem = "fix"
)

var authoritySubsystems = []AuthoritySubsystem{
	AuthorityFilesystemChecks,
	AuthorityCollections,
	AuthorityCollectionChecks,
	AuthoritySchemas,
	AuthorityFix,
}

type NestedConfigSettings struct {
	Discovery        NestedConfigDiscovery
	DefaultAuthority AuthorityPolicy
	Delegates        []NestedDelegate
}

type NestedDelegate struct {
	Path      string
	Config    string
	Authority map[AuthoritySubsystem]AuthorityRule
}

type AuthorityRule struct {
	Default  AuthorityPolicy
	Kinds    map[string]AuthorityPolicy
	Families map[string]AuthorityPolicy
}

type rawNestedConfigSettings struct {
	Discovery        string              `yaml:"discovery"`
	DefaultAuthority string              `yaml:"defaultAuthority"`
	Delegates        []rawNestedDelegate `yaml:"delegates"`
}

type rawNestedDelegate struct {
	Path      string                            `yaml:"path"`
	Config    string                            `yaml:"config"`
	Authority map[string]rawAuthorityRuleOrText `yaml:"authority"`
}

type rawAuthorityRuleOrText struct {
	Text     string
	Default  string            `yaml:"default"`
	Kinds    map[string]string `yaml:"kinds"`
	Families map[string]string `yaml:"families"`
}

func (r *rawAuthorityRuleOrText) UnmarshalYAML(unmarshal func(any) error) error {
	var text string
	if err := unmarshal(&text); err == nil {
		r.Text = text
		return nil
	}
	type expanded rawAuthorityRuleOrText
	var e expanded
	if err := unmarshal(&e); err != nil {
		return err
	}
	*r = rawAuthorityRuleOrText(e)
	return nil
}

func buildNestedSettings(raw rawNestedConfigSettings) (NestedConfigSettings, error) {
	discovery, err := normNestedDiscovery(raw.Discovery)
	if err != nil {
		return NestedConfigSettings{}, err
	}
	defaultAuthority, err := normAuthorityPolicy(raw.DefaultAuthority)
	if err != nil {
		return NestedConfigSettings{}, fmt.Errorf("defaultAuthority: %w", err)
	}
	if len(raw.Delegates) > 0 && discovery == NestedDiscoveryNone {
		discovery = NestedDiscoveryExplicit
	}

	settings := NestedConfigSettings{
		Discovery:        discovery,
		DefaultAuthority: defaultAuthority,
		Delegates:        make([]NestedDelegate, 0, len(raw.Delegates)),
	}
	seen := map[string]bool{}
	for i, rawDelegate := range raw.Delegates {
		delegate, err := buildNestedDelegate(rawDelegate)
		if err != nil {
			return NestedConfigSettings{}, fmt.Errorf("delegates[%d]: %w", i, err)
		}
		if seen[delegate.Path] {
			return NestedConfigSettings{}, fmt.Errorf("delegates[%d]: duplicate path %q", i, delegate.Path)
		}
		seen[delegate.Path] = true
		settings.Delegates = append(settings.Delegates, delegate)
	}
	sort.Slice(settings.Delegates, func(i, j int) bool {
		return settings.Delegates[i].Path < settings.Delegates[j].Path
	})
	return settings, nil
}

func buildNestedDelegate(raw rawNestedDelegate) (NestedDelegate, error) {
	if raw.Path == "" {
		return NestedDelegate{}, fmt.Errorf("path is required")
	}
	path := filepath.ToSlash(filepath.Clean(raw.Path))
	if path == "." || filepath.IsAbs(raw.Path) || path == ".." || startsWithDotDot(path) {
		return NestedDelegate{}, fmt.Errorf("path must be relative to the active root")
	}
	config := raw.Config
	if config == "" {
		config = Dir
	}
	if filepath.IsAbs(config) {
		return NestedDelegate{}, fmt.Errorf("config must be relative to path")
	}
	config = filepath.ToSlash(filepath.Clean(config))
	authority := map[AuthoritySubsystem]AuthorityRule{}
	for key, rawRule := range raw.Authority {
		subsystem, err := normAuthoritySubsystem(key)
		if err != nil {
			return NestedDelegate{}, err
		}
		rule, err := buildAuthorityRule(rawRule)
		if err != nil {
			return NestedDelegate{}, fmt.Errorf("%s: %w", key, err)
		}
		authority[subsystem] = rule
	}
	return NestedDelegate{Path: path, Config: config, Authority: authority}, nil
}

func buildAuthorityRule(raw rawAuthorityRuleOrText) (AuthorityRule, error) {
	if raw.Text != "" {
		policy, err := normAuthorityPolicy(raw.Text)
		if err != nil {
			return AuthorityRule{}, err
		}
		return AuthorityRule{Default: policy}, nil
	}
	rule := AuthorityRule{
		Kinds:    map[string]AuthorityPolicy{},
		Families: map[string]AuthorityPolicy{},
	}
	if raw.Default != "" {
		policy, err := normAuthorityPolicy(raw.Default)
		if err != nil {
			return AuthorityRule{}, fmt.Errorf("default: %w", err)
		}
		rule.Default = policy
	}
	for kind, text := range raw.Kinds {
		policy, err := normAuthorityPolicy(text)
		if err != nil {
			return AuthorityRule{}, fmt.Errorf("kinds.%s: %w", kind, err)
		}
		rule.Kinds[kind] = policy
	}
	for family, text := range raw.Families {
		policy, err := normAuthorityPolicy(text)
		if err != nil {
			return AuthorityRule{}, fmt.Errorf("families.%s: %w", family, err)
		}
		rule.Families[family] = policy
	}
	return rule, nil
}

func normNestedDiscovery(s string) (NestedConfigDiscovery, error) {
	switch NestedConfigDiscovery(s) {
	case "":
		return NestedDiscoveryNone, nil
	case NestedDiscoveryNone, NestedDiscoveryExplicit, NestedDiscoveryWalk:
		return NestedConfigDiscovery(s), nil
	default:
		return "", fmt.Errorf("nestedConfigs: unknown discovery %q (want none, explicit, or walk)", s)
	}
}

func normAuthorityPolicy(s string) (AuthorityPolicy, error) {
	switch AuthorityPolicy(s) {
	case "":
		return AuthorityRootNearest, nil
	case AuthorityRootNearest, AuthorityFileNearest, AuthorityCompose:
		return AuthorityPolicy(s), nil
	default:
		return "", fmt.Errorf("unknown authority policy %q (want root_nearest, file_nearest, or compose)", s)
	}
}

func normAuthoritySubsystem(s string) (AuthoritySubsystem, error) {
	switch AuthoritySubsystem(s) {
	case AuthorityFilesystemChecks, AuthorityCollections, AuthorityCollectionChecks, AuthoritySchemas, AuthorityFix:
		return AuthoritySubsystem(s), nil
	default:
		return "", fmt.Errorf("unknown authority subsystem %q", s)
	}
}

func startsWithDotDot(path string) bool {
	return path == ".." || len(path) > 3 && path[:3] == "../"
}
