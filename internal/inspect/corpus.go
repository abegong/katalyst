// Package inspect profiles a directory of markdown files. It measures their
// shape — frontmatter fields, body structure, filename conventions — and
// returns evidence, the descriptive dual of internal/checks. A check asserts a
// predicate; an inspector reports the distribution that predicate would be
// tested against. Inspectors never recommend; they report counts an agent or
// human judges.
//
// See product/specs/inspect-spec.md.
package inspect

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/katabase-ai/katalyst/internal/frontmatter"
)

// File is one markdown file in a Corpus: its path relative to the scope root,
// the parsed document, and any parse error. A file that failed to parse still
// appears in the corpus — a failure to parse is itself evidence.
type File struct {
	Rel      string
	Doc      *frontmatter.Document
	ParseErr error
}

// Corpus is the parsed set of markdown files under a scope. It is built once
// (see Load) and shared across inspectors, so each inspector reads parsed
// evidence rather than re-reading disk.
type Corpus struct {
	Scope string
	Files []File
}

// Load walks root for *.md files and parses each. A file that fails to read or
// parse is recorded with its error rather than dropped. Files are sorted by
// relative path so evidence is deterministic.
func Load(root string) (Corpus, error) {
	c := Corpus{Scope: root}
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		f := File{Rel: filepath.ToSlash(rel)}
		if src, readErr := os.ReadFile(path); readErr != nil {
			f.ParseErr = readErr
		} else {
			f.Doc, f.ParseErr = frontmatter.Parse(src)
		}
		c.Files = append(c.Files, f)
		return nil
	})
	if walkErr != nil {
		return Corpus{}, walkErr
	}
	sort.Slice(c.Files, func(i, j int) bool { return c.Files[i].Rel < c.Files[j].Rel })
	return c, nil
}

// meta returns a file's frontmatter map, or nil when the file has no
// frontmatter or failed to parse. Inspectors read keys through this so a
// missing or broken document contributes nothing rather than panicking.
func meta(f File) map[string]any {
	if f.Doc == nil || !f.Doc.HasFrontmatter {
		return nil
	}
	return f.Doc.Meta
}
