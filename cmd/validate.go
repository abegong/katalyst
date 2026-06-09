package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
		Short: "Validate markdown frontmatter against a JSON Schema.",
		Long: `Validate parses YAML frontmatter from each markdown file and
checks it against a JSON Schema. Schema resolution, highest precedence first:

  1. --schema <path>      (applies to every file in the invocation)
  2. inline "schema:" key in the file's frontmatter (a schema name from
     katalyst.yaml)
  3. the first matching rule in katalyst.yaml

Files that don't resolve to any schema are reported as errors.`,
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

// resolver picks a schema for a given file path. It hides the precedence
// rules from the validate loop and caches compiled schemas so repeated
// hits on the same schema file don't re-parse.
type resolver struct {
	// forcedPath, if non-empty, is the --schema override; every file
	// gets this schema regardless of frontmatter or config.
	forcedPath string
	// cfg is nil when --schema is set (config isn't loaded at all in
	// that mode).
	cfg *config.Config
	// cache holds compiled schemas by absolute file path.
	cache map[string]*validator.Schema
}

func newResolver(schemaFlag string) (*resolver, error) {
	r := &resolver{cache: map[string]*validator.Schema{}}
	if schemaFlag != "" {
		if _, err := os.Stat(schemaFlag); err != nil {
			return nil, usageErr(fmt.Sprintf("--schema: %v", err))
		}
		r.forcedPath = schemaFlag
		return r, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(wd)
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			return nil, usageErr("no --schema given and no katalyst.yaml found (run `katalyst init`)")
		}
		return nil, err
	}
	r.cfg = cfg
	return r, nil
}

// schemaFor returns the compiled schema for filePath, the human-readable
// label describing which source determined that choice, and an error if
// no schema could be resolved.
func (r *resolver) schemaFor(filePath string, meta map[string]any) (*validator.Schema, string, error) {
	// 1. --schema wins.
	if r.forcedPath != "" {
		s, err := r.compile(r.forcedPath)
		return s, "--schema " + r.forcedPath, err
	}

	// 2. inline "schema:" key in frontmatter.
	if name, ok := meta["schema"].(string); ok && name != "" {
		path := r.cfg.SchemaPath(name)
		if path == "" {
			return nil, "", fmt.Errorf("inline schema %q is not registered in katalyst.yaml", name)
		}
		s, err := r.compile(path)
		return s, "inline schema: " + name, err
	}

	// 3. config rules.
	if name, ok := r.cfg.Match(filePath); ok {
		path := r.cfg.SchemaPath(name)
		s, err := r.compile(path)
		return s, "rule -> " + name, err
	}

	return nil, "", errors.New("no schema matched (unmatched file)")
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

// validateFile reads one markdown file, picks a schema, validates the
// frontmatter, and writes results to out/errOut. It returns (true, nil)
// if the file is valid, (false, nil) if it has validation errors, or
// (_, err) if the file couldn't be read/parsed or no schema applies.
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

	schema, _, err := r.schemaFor(path, doc.Meta)
	if err != nil {
		return false, err
	}

	// The "schema" key is a katalyst directive, not user data. Strip
	// it before validating so user schemas with additionalProperties:
	// false don't reject documents that opt into themselves.
	instance := dropKey(doc.Meta, "schema")

	result := schema.Validate(instance)
	if result.Valid {
		fmt.Fprintf(out, "%s: OK\n", path)
		return true, nil
	}

	for _, e := range result.Errors {
		loc := e.Path
		if loc == "" {
			loc = "/"
		}
		if line, ok := lookupLine(doc.Lines, e.Path); ok {
			fmt.Fprintf(errOut, "%s:%d: %s: %s\n", path, line, loc, e.Message)
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
