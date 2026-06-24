package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/codec/markdownbodytext"
	"github.com/abegong/katalyst/internal/project"
	"gopkg.in/yaml.v3"
)

// parseItem reads and parses a markdown file's frontmatter and body.
func parseItem(path string) (*markdownbodytext.Document, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return markdownbodytext.Parse(src)
}

// validateItemWrite validates src against the checks for collection c,
// using the engine's schema-resolution precedence (--schema, inline
// "schema:" key, then the collection's object checks). It returns a
// multi-line error describing every violation, or nil when valid.
func validateItemWrite(e *engine, c project.Collection, path string, src []byte) error {
	doc, err := markdownbodytext.Parse(src)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if !doc.HasFrontmatter {
		return fmt.Errorf("%s: no frontmatter found", path)
	}

	checkList, err := e.checksFor(c, doc.Meta)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	instance := dropKey(doc.Meta, "schema")
	result := checks.RunAll(checks.Context{
		FilePath: path,
		Doc:      doc,
		Meta:     instance,
	}, checkList)

	// Only error-severity violations block a write; warnings are advisory.
	var lines []string
	for _, v := range result {
		if v.Severity == checks.SeverityWarning {
			continue
		}
		loc := v.Path
		if loc == "" {
			loc = "/"
		}
		if v.Line > 0 {
			lines = append(lines, fmt.Sprintf("%s:%d: %s: %s", path, v.Line, loc, v.Message))
		} else {
			lines = append(lines, fmt.Sprintf("%s: %s: %s", path, loc, v.Message))
		}
	}
	if len(lines) == 0 {
		return nil
	}
	return errors.New(strings.Join(lines, "\n"))
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
		return key, "", nil
	}
	if err := yaml.Unmarshal([]byte(raw), &value); err != nil {
		return "", nil, fmt.Errorf("invalid value for %q: %w", key, err)
	}
	return key, value, nil
}
