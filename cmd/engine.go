package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	_ "github.com/abegong/katalyst/internal/checks/all" // register every check-type family
	"github.com/abegong/katalyst/internal/checks/structuredobject"
	"github.com/abegong/katalyst/internal/config"
	"github.com/abegong/katalyst/internal/project"
	"github.com/abegong/katalyst/internal/validator"
)

// engine resolves and compiles the checks for an item. It loads the
// project config once and caches compiled schemas across items.
type engine struct {
	proj *project.Project
	// forcedPath is the --schema override; when set, every item gets this
	// object schema regardless of inline key or collection config.
	forcedPath string
	cache      map[string]*validator.Schema
}

// newEngine loads the config from the working directory and validates the
// optional --schema override. A missing config is a usage error.
func newEngine(schemaFlag string) (*engine, error) {
	e := &engine{cache: map[string]*validator.Schema{}}
	if schemaFlag != "" {
		if _, err := os.Stat(schemaFlag); err != nil {
			return nil, usageErr(fmt.Sprintf("--schema: %v", err))
		}
		e.forcedPath = schemaFlag
	}
	cfg, err := loadConfigFromCWD()
	if err != nil {
		return nil, err
	}
	e.proj = project.New(cfg)
	return e, nil
}

func (e *engine) compile(path string) (*validator.Schema, error) {
	if cached, ok := e.cache[path]; ok {
		return cached, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open schema %s: %w", path, err)
	}
	defer f.Close()

	var s *validator.Schema
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		s, err = validator.LoadYAML(path, f)
	default:
		s, err = validator.Load(path, f)
	}
	if err != nil {
		return nil, err
	}
	e.cache[path] = s
	return s, nil
}

// checksFor builds the runnable check list for one item. Object-schema
// resolution precedence (highest first): --schema override, inline
// "schema:" key, the collection's configured object checks. Non-object
// checks always come from the collection.
//
// When the collection declares variants, the first variant whose `when`
// predicates the item's metadata satisfies contributes its checks on top of
// the base set (additively, through the same precedence). An item that matches
// no variant runs the base only, or, under useExhaustiveVariants, fails with
// "matches no variant".
func (e *engine) checksFor(c config.Collection, meta map[string]any) ([]checks.Check, error) {
	cfg := e.proj.Config()

	matched, routed, err := matchVariant(c, meta)
	if err != nil {
		return nil, err
	}
	effective := c.Checks
	if routed {
		effective = make([]config.CheckInstance, 0, len(c.Checks)+len(matched.Checks))
		effective = append(effective, c.Checks...)
		effective = append(effective, matched.Checks...)
	}

	checkList := make([]checks.Check, 0, len(effective))

	inlineSchema := ""
	if raw, ok := meta["schema"].(string); ok {
		inlineSchema = strings.TrimSpace(raw)
	}

	switch {
	case e.forcedPath != "":
		schema, err := e.compile(e.forcedPath)
		if err != nil {
			return nil, err
		}
		checkList = append(checkList, structuredobject.Object{Schema: schema})
	case inlineSchema != "":
		path := cfg.SchemaPath(inlineSchema)
		if path == "" {
			return nil, fmt.Errorf("inline schema %q is not defined under .katalyst/schemas/", inlineSchema)
		}
		schema, err := e.compile(path)
		if err != nil {
			return nil, err
		}
		checkList = append(checkList, structuredobject.Object{Schema: schema})
	default:
		for _, ch := range effective {
			if ch.Type != config.CheckObject {
				continue
			}
			path := cfg.SchemaPath(ch.Schema)
			if path == "" {
				return nil, fmt.Errorf("collection object schema %q is not defined under .katalyst/schemas/", ch.Schema)
			}
			schema, err := e.compile(path)
			if err != nil {
				return nil, err
			}
			checkList = append(checkList, structuredobject.Object{Schema: schema})
		}
	}

	// Every non-object, per-item check is built from its registry entry. The
	// object check is handled above (it needs a compiled schema); collection-
	// scoped checks have no per-item builder, so Build skips them here.
	for _, ch := range effective {
		if ch.Type == config.CheckObject {
			continue
		}
		if chk, ok := checks.Build(ch); ok {
			checkList = append(checkList, chk)
		}
	}

	// An item that matched no variant under useExhaustiveVariants fails. The
	// verdict rides through RunAll like any other check (so `check` and
	// `item list` report it identically).
	if !routed && c.UseExhaustiveVariants && len(c.Variants) > 0 {
		checkList = append(checkList, unroutedCheck{})
	}

	// A collection with variants is validated to carry some check config, and
	// an unrouted item under lenient mode legitimately runs nothing, so the
	// empty-list guard only applies to plain (variant-less) collections.
	if len(checkList) == 0 && !c.HasCollectionChecks() && len(c.Variants) == 0 {
		return nil, errors.New("no checks configured for collection")
	}
	return checkList, nil
}

// matchVariant returns the first variant whose `when` predicates the item's
// metadata all satisfy, and whether any matched. The collection's
// filterTypeMismatch governs an incomparable predicate (skip vs. error).
func matchVariant(c config.Collection, meta map[string]any) (config.CollectionVariant, bool, error) {
	for _, v := range c.Variants {
		all := true
		for _, p := range v.Where {
			ok, err := p.Matches(meta, c.Query.FilterTypeMismatch)
			if err != nil {
				return config.CollectionVariant{}, false, err
			}
			if !ok {
				all = false
				break
			}
		}
		if all {
			return v, true, nil
		}
	}
	return config.CollectionVariant{}, false, nil
}

// unroutedCheck reports a single violation for an item that matched no variant
// when the collection sets useExhaustiveVariants. It implements checks.Check so
// the verdict flows through RunAll uniformly for `check` and `item list`.
type unroutedCheck struct{}

func (unroutedCheck) Run(checks.Context) []checks.Violation {
	return []checks.Violation{{Message: "matches no variant"}}
}

// collectionChecksFor builds the collection-scoped checks configured for a
// collection. These run once per collection, after the per-item pass.
func (e *engine) collectionChecksFor(c config.Collection) []checks.CollectionCheck {
	var out []checks.CollectionCheck
	for _, ch := range c.Checks {
		if cc, ok := checks.BuildCollection(ch); ok {
			out = append(out, cc)
		}
	}
	return out
}

// projectFor wraps a loaded config in a project.
func projectFor(cfg *config.Config) *project.Project { return project.New(cfg) }

// resolveSelectors maps a *project.UsageError to a cmd usage error (exit
// 2) and passes other errors through unchanged.
func resolveSelectors(p *project.Project, selectors []string) (*project.Resolution, error) {
	res, err := p.Resolve(selectors)
	if err != nil {
		return nil, asUsageErr(err)
	}
	return res, nil
}

// asUsageErr converts project usage errors into the cmd exitError with
// code 2; other errors are wrapped as exit 2 as well, since selector/IO
// failures all surface as usage errors per the spec.
func asUsageErr(err error) error {
	var ue *project.UsageError
	if errors.As(err, &ue) {
		return usageErr(ue.Msg)
	}
	return usageErr(err.Error())
}
