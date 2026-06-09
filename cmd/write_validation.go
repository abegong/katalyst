package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/katabase-ai/katalyst/internal/checks"
	"github.com/katabase-ai/katalyst/internal/frontmatter"
	"github.com/katabase-ai/katalyst/internal/validator"
	"gopkg.in/yaml.v3"
)

// validateWrite validates src as a markdown document for write-affecting
// commands (create/update). When strict is false, it returns nil immediately.
//
// Validation uses the same schema-resolution precedence as `validate`:
// --schema override, then inline schema key, then config rules.
func validateWrite(path string, src []byte, schemaFlag string, strict bool) error {
	if !strict {
		return nil
	}
	r, err := newResolver(schemaFlag)
	if err != nil {
		return err
	}

	doc, err := frontmatter.Parse(src)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if !doc.HasFrontmatter {
		return fmt.Errorf("%s: no frontmatter found", path)
	}

	checkList, err := r.checksFor(path, doc.Meta)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	instance := dropKey(doc.Meta, "schema")
	result := checks.RunAll(checks.Context{
		FilePath: path,
		Doc:      doc,
		Meta:     instance,
	}, checkList)
	if len(result) == 0 {
		return nil
	}

	var lines []string
	for _, e := range result {
		loc := e.Path
		if loc == "" {
			loc = "/"
		}
		if e.Line > 0 {
			lines = append(lines, fmt.Sprintf("%s:%d: %s: %s", path, e.Line, loc, e.Message))
		} else {
			lines = append(lines, fmt.Sprintf("%s: %s: %s", path, loc, e.Message))
		}
	}
	return errors.New(strings.Join(lines, "\n"))
}

// isMarkdownPath reports whether path appears to be a markdown file based
// on extension only. Validation hooks use this to avoid forcing frontmatter
// semantics on unrelated file types.
func isMarkdownPath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".md")
}

// composeMarkdown builds a markdown document from frontmatter metadata and
// body bytes. Body is preserved exactly as provided.
func composeMarkdown(meta map[string]any, body []byte) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(meta)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(yamlBytes)
	if !bytes.HasSuffix(yamlBytes, []byte("\n")) {
		out.WriteByte('\n')
	}
	out.WriteString("---\n")
	out.Write(body)
	return out.Bytes(), nil
}

// writeFileAtomic writes b to path via temp-file-and-rename in the same
// directory, minimizing the chance of partially-written files.
func writeFileAtomic(path string, b []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".katalyst-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// parseAssignment parses key=value and yaml-decodes the right-hand side so
// users can pass typed values (numbers, booleans, arrays, etc.) from CLI.
func parseAssignment(s string) (key string, value any, err error) {
	i := strings.IndexByte(s, '=')
	if i <= 0 {
		return "", nil, fmt.Errorf("invalid assignment %q (expected key=value)", s)
	}
	key = strings.TrimSpace(s[:i])
	raw := strings.TrimSpace(s[i+1:])
	if key == "" {
		return "", nil, fmt.Errorf("invalid assignment %q (empty key)", s)
	}
	if raw == "" {
		// Empty string is a legitimate scalar assignment.
		return key, "", nil
	}
	if err := yaml.Unmarshal([]byte(raw), &value); err != nil {
		return "", nil, fmt.Errorf("invalid value for %q: %w", key, err)
	}
	return key, value, nil
}

// flattenValidationErrors is used by callers that need structured errors.
func flattenValidationErrors(path string, errs []validator.Error, lines map[string]int) []string {
	out := make([]string, 0, len(errs))
	for _, e := range errs {
		loc := e.Path
		if loc == "" {
			loc = "/"
		}
		if line, ok := lookupLine(lines, e.Path); ok {
			out = append(out, fmt.Sprintf("%s:%d: %s: %s", path, line, loc, e.Message))
		} else {
			out = append(out, fmt.Sprintf("%s: %s: %s", path, loc, e.Message))
		}
	}
	return out
}
