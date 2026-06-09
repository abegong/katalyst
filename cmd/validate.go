package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/katabase-ai/katalyst/internal/checks"
	"github.com/katabase-ai/katalyst/internal/config"
	"github.com/katabase-ai/katalyst/internal/frontmatter"
	"github.com/katabase-ai/katalyst/internal/validator"
	"github.com/spf13/cobra"
)

// Exit codes for `validate`. Loosely modeled on shellcheck and on the
// `jv` CLI from santhosh-tekuri/jsonschema.
const (
	exitOK             = 0
	exitValidationFail = 1
	exitUsage          = 2
)

func newValidateCmd() *cobra.Command {
	var schemaFlag string

	c := &cobra.Command{
		Use:   "validate [paths...]",
		Short: "Run configured checks against markdown files.",
		Long: `Validate parses YAML frontmatter from each markdown file and
runs the checks configured in katalyst.yaml.

Object-schema resolution, highest precedence first:

  1. --schema <path>      (applies to every file in the invocation)
  2. inline "schema:" key in the file's frontmatter (a schema name from
     katalyst.yaml)
  3. object checks in the first matching rule in katalyst.yaml

Markdown and filesystem checks come from the first matching rule and run
even when --schema is used.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := newResolver(schemaFlag)
			if err != nil {
				return err
			}

			anyInvalid := false
			for _, path := range args {
				ok, err := validateFile(cmd.OutOrStdout(), cmd.ErrOrStderr(), r, path)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", path, err)
					anyInvalid = true
					continue
				}
				if !ok {
					anyInvalid = true
				}
			}

			if anyInvalid {
				return &exitError{code: exitValidationFail}
			}
			return nil
		},
	}

	c.Flags().StringVarP(&schemaFlag, "schema", "s", "",
		"Path to a JSON Schema file. Overrides config-based resolution for every input.")
	return c
}

// resolver builds checks for a given file path.
type resolver struct {
	// forcedPath, if non-empty, is the --schema override; every file
	// gets this object schema regardless of frontmatter or config.
	forcedPath string
	// cfg can be nil when only --schema is used and no config exists.
	cfg *config.Config
	// cache holds compiled object schemas by absolute file path.
	cache map[string]*validator.Schema
}

func newResolver(schemaFlag string) (*resolver, error) {
	r := &resolver{cache: map[string]*validator.Schema{}}
	if schemaFlag != "" {
		if _, err := os.Stat(schemaFlag); err != nil {
			return nil, usageErr(fmt.Sprintf("--schema: %v", err))
		}
		r.forcedPath = schemaFlag
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(wd)
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			if r.forcedPath != "" {
				return r, nil
			}
			return nil, usageErr("no --schema given and no katalyst.yaml found (run `katalyst init`)")
		}
		return nil, err
	}
	r.cfg = cfg
	return r, nil
}

func (r *resolver) compile(path string) (*validator.Schema, error) {
	if cached, ok := r.cache[path]; ok {
		return cached, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open schema %s: %w", path, err)
	}
	defer f.Close()
	s, err := validator.Load(path, f)
	if err != nil {
		return nil, err
	}
	r.cache[path] = s
	return s, nil
}

func (r *resolver) checksFor(filePath string, meta map[string]any) ([]checks.Check, error) {
	checkList := make([]checks.Check, 0)

	var matchedRule config.Rule
	hasMatchedRule := false
	if r.cfg != nil {
		matchedRule, hasMatchedRule = r.cfg.RuleFor(filePath)
	}

	inlineSchema := ""
	if raw, ok := meta["schema"].(string); ok {
		inlineSchema = strings.TrimSpace(raw)
	}

	// Object checks: forced schema, then inline schema, then rule object checks.
	switch {
	case r.forcedPath != "":
		schema, err := r.compile(r.forcedPath)
		if err != nil {
			return nil, err
		}
		checkList = append(checkList, checks.Object{Schema: schema})
	case inlineSchema != "":
		if r.cfg == nil {
			return nil, errors.New("inline schema requires katalyst.yaml to resolve names")
		}
		path := r.cfg.SchemaPath(inlineSchema)
		if path == "" {
			return nil, fmt.Errorf("inline schema %q is not registered in katalyst.yaml", inlineSchema)
		}
		schema, err := r.compile(path)
		if err != nil {
			return nil, err
		}
		checkList = append(checkList, checks.Object{Schema: schema})
	case hasMatchedRule:
		for _, cfgCheck := range matchedRule.Checks {
			if cfgCheck.Kind != config.CheckObject {
				continue
			}
			path := r.cfg.SchemaPath(cfgCheck.Schema)
			if path == "" {
				return nil, fmt.Errorf("rule object schema %q is not registered in katalyst.yaml", cfgCheck.Schema)
			}
			schema, err := r.compile(path)
			if err != nil {
				return nil, err
			}
			checkList = append(checkList, checks.Object{Schema: schema})
		}
	}

	// Non-object checks come from the matched rule regardless of --schema.
	if hasMatchedRule {
		for _, cfgCheck := range matchedRule.Checks {
			switch cfgCheck.Kind {
			case config.CheckMarkdownTitleMatchesH1:
				checkList = append(checkList, checks.MarkdownTitleMatchesH1{Field: cfgCheck.Field})
			case config.CheckFilesystemFilenameMatchesSlug:
				checkList = append(checkList, checks.FilenameMatchesSlug{Field: cfgCheck.Field})
			}
		}
	}

	if len(checkList) > 0 {
		return checkList, nil
	}
	if r.forcedPath == "" && inlineSchema == "" && !hasMatchedRule {
		return nil, errors.New("no checks matched (unmatched file)")
	}
	return nil, errors.New("no checks configured for file")
}

// validateFile reads one markdown file, resolves checks, runs them, and
// writes results to out/errOut. It returns (true, nil) if the file is
// valid, (false, nil) if it has validation errors, or (_, err) if the
// file couldn't be read/parsed or no checks apply.
func validateFile(out, errOut io.Writer, r *resolver, path string) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	doc, err := frontmatter.Parse(src)
	if err != nil {
		return false, err
	}

	if !doc.HasFrontmatter {
		fmt.Fprintf(errOut, "%s: no frontmatter found\n", path)
		return false, nil
	}

	checkList, err := r.checksFor(path, doc.Meta)
	if err != nil {
		return false, err
	}

	// The "schema" key is a katalyst directive, not user data. Strip
	// it before validating so user schemas with additionalProperties:
	// false don't reject documents that opt into themselves.
	instance := dropKey(doc.Meta, "schema")

	result := checks.RunAll(checks.Context{
		FilePath: path,
		Doc:      doc,
		Meta:     instance,
	}, checkList)
	if len(result) == 0 {
		fmt.Fprintf(out, "%s: OK\n", path)
		return true, nil
	}

	for _, e := range result {
		loc := e.Path
		if loc == "" {
			loc = "/"
		}
		if e.Line > 0 {
			fmt.Fprintf(errOut, "%s:%d: %s: %s\n", path, e.Line, loc, e.Message)
		} else {
			fmt.Fprintf(errOut, "%s: %s: %s\n", path, loc, e.Message)
		}
	}
	return false, nil
}

// lookupLine returns the source line for a JSON pointer path, walking
// up to ancestor paths if the exact one isn't known. "Missing required
// property" errors, for instance, are reported at the parent's path
// because the missing key has no source location of its own.
func lookupLine(lines map[string]int, ptr string) (int, bool) {
	for {
		if line, ok := lines[ptr]; ok {
			return line, true
		}
		if ptr == "" {
			return 0, false
		}
		i := strings.LastIndexByte(ptr, '/')
		if i < 0 {
			return 0, false
		}
		ptr = ptr[:i]
	}
}

// usageErr wraps an error so main can exit with code 2 (usage error).
func usageErr(msg string) error {
	return &exitError{code: exitUsage, msg: msg}
}

// exitError carries an explicit process exit code.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string {
	if e.msg == "" {
		return fmt.Sprintf("exit %d", e.code)
	}
	return e.msg
}

// Code returns the desired process exit code.
func (e *exitError) Code() int { return e.code }

// dropKey returns a shallow copy of m without the named key. The
// original map is not mutated, so the caller can keep using it for
// other purposes (logging, future reference resolution, etc.).
func dropKey(m map[string]any, key string) map[string]any {
	if _, present := m[key]; !present {
		return m
	}
	out := make(map[string]any, len(m)-1)
	for k, v := range m {
		if k == key {
			continue
		}
		out[k] = v
	}
	return out
}
