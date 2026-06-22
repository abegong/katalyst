package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/katabase-ai/katalyst/internal/checks"
	"github.com/katabase-ai/katalyst/internal/config"
	"github.com/katabase-ai/katalyst/internal/project"
	"github.com/katabase-ai/katalyst/internal/validator"
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
func (e *engine) checksFor(c config.Collection, meta map[string]any) ([]checks.Check, error) {
	cfg := e.proj.Config()
	checkList := make([]checks.Check, 0, len(c.Checks))

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
		checkList = append(checkList, checks.Object{Schema: schema})
	case inlineSchema != "":
		path := cfg.SchemaPath(inlineSchema)
		if path == "" {
			return nil, fmt.Errorf("inline schema %q is not defined under .katalyst/schemas/", inlineSchema)
		}
		schema, err := e.compile(path)
		if err != nil {
			return nil, err
		}
		checkList = append(checkList, checks.Object{Schema: schema})
	default:
		for _, ch := range c.Checks {
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
			checkList = append(checkList, checks.Object{Schema: schema})
		}
	}

	for _, ch := range c.Checks {
		switch ch.Type {
		case config.CheckObjectRequiredField:
			checkList = append(checkList, checks.ObjectRequiredField{Field: ch.Field})
		case config.CheckObjectFieldType:
			checkList = append(checkList, checks.ObjectFieldType{Field: ch.Field, Type: ch.FieldType})
		case config.CheckObjectFieldEnum:
			checkList = append(checkList, checks.ObjectFieldEnum{Field: ch.Field, Values: ch.Values})
		case config.CheckObjectNumberRange:
			checkList = append(checkList, checks.ObjectNumberRange{Field: ch.Field, Min: ch.Min, Max: ch.Max})
		case config.CheckObjectStringLength:
			checkList = append(checkList, checks.ObjectStringLength{
				Field:     ch.Field,
				MinLength: ch.MinLength,
				MaxLength: ch.MaxLength,
			})
		case config.CheckMarkdownTitleMatchesH1:
			checkList = append(checkList, checks.MarkdownTitleMatchesH1{Field: ch.Field})
		case config.CheckMarkdownRequiresH1:
			checkList = append(checkList, checks.MarkdownRequiresH1{})
		case config.CheckMarkdownSingleH1:
			checkList = append(checkList, checks.MarkdownSingleH1{})
		case config.CheckMarkdownNoHeadingLevelJumps:
			checkList = append(checkList, checks.MarkdownNoHeadingLevelJumps{})
		case config.CheckMarkdownRequiredSection:
			checkList = append(checkList, checks.MarkdownRequiredSection{Heading: ch.Heading})
		case config.CheckMarkdownCodeFenceHasLanguage:
			checkList = append(checkList, checks.MarkdownCodeFenceHasLanguage{})
		case config.CheckFilesystemExtensionIn:
			checkList = append(checkList, checks.FilesystemExtensionIn{Values: ch.Values})
		case config.CheckFilesystemParentDirIn:
			checkList = append(checkList, checks.FilesystemParentDirIn{Values: ch.Values})
		case config.CheckFilesystemNameCase:
			checkList = append(checkList, checks.NameCase{Style: ch.Style, Target: ch.Target})
		case config.CheckFilesystemNameMatchesField:
			checkList = append(checkList, checks.NameMatchesField{Field: ch.Field, Transform: ch.Transform, Target: ch.Target})
		case config.CheckFilesystemNameAffix:
			checkList = append(checkList, checks.NameAffix{Prefix: ch.Prefix, Suffix: ch.Suffix, Target: ch.Target})
		case config.CheckFilesystemPathCharset:
			checkList = append(checkList, checks.PathCharset{Allow: ch.Allow, Deny: ch.Deny})
		case config.CheckFilesystemNameRegex:
			checkList = append(checkList, checks.NameRegex{
				Re:      regexp.MustCompile(checks.AnchoredPattern(ch.Pattern)),
				Pattern: ch.Pattern,
				Target:  ch.Target,
			})
		case config.CheckFilesystemNameLength:
			checkList = append(checkList, checks.NameLength{Min: ch.MinInt, Max: ch.MaxInt, Target: ch.Target})
		case config.CheckFilesystemPathDepth:
			checkList = append(checkList, checks.PathDepth{Min: ch.MinInt, Max: ch.MaxInt})
		case config.CheckFilesystemParentDirMatchesFld:
			checkList = append(checkList, checks.ParentDirMatchesField{Field: ch.Field})
		case config.CheckFilesystemReferencedFiles:
			checkList = append(checkList, checks.ReferencedFilesExist{Fields: ch.Fields})
		}
	}

	if len(checkList) == 0 && !c.HasCollectionChecks() {
		return nil, errors.New("no checks configured for collection")
	}
	return checkList, nil
}

// collectionChecksFor builds the collection-scoped checks configured for a
// collection. These run once per collection, after the per-item pass.
func (e *engine) collectionChecksFor(c config.Collection) []checks.CollectionCheck {
	var out []checks.CollectionCheck
	for _, ch := range c.Checks {
		switch ch.Type {
		case config.CheckFilesystemUniqueFilename:
			out = append(out, checks.UniqueFilename{})
		case config.CheckFilesystemUniqueField:
			out = append(out, checks.UniqueField{Field: ch.Field})
		case config.CheckFilesystemIndexFileRequired:
			out = append(out, checks.IndexFileRequired{Name: ch.Name})
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
