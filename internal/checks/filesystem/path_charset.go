package filesystem

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// PathCharset constrains the characters of the collection-relative path.
// Exactly one of Allow / Deny is set (enforced at config load). Deny lists
// forbidden substrings; Allow lists the only permitted characters (the path
// separator is always allowed).
type PathCharset struct {
	Allow []string
	Deny  []string
}

func (c PathCharset) Run(ctx checks.Context) []checks.Violation {
	path := filepath.ToSlash(ctx.FilePath)
	if ctx.CollectionRoot != "" {
		if r, err := filepath.Rel(ctx.CollectionRoot, ctx.FilePath); err == nil {
			path = filepath.ToSlash(r)
		}
	}
	if len(c.Deny) > 0 {
		var out []checks.Violation
		for _, d := range c.Deny {
			if d != "" && strings.Contains(path, d) {
				out = append(out, checks.Violation{
					Path:    "/",
					Message: fmt.Sprintf("file path must not contain %q", d),
				})
			}
		}
		return out
	}
	allowed := map[rune]bool{'/': true}
	for _, a := range c.Allow {
		for _, r := range a {
			allowed[r] = true
		}
	}
	for _, r := range path {
		if !allowed[r] {
			return []checks.Violation{{
				Path:    "/",
				Message: fmt.Sprintf("file path contains disallowed character %q", string(r)),
			}}
		}
	}
	return nil
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckFilesystemPathCharset,
		Family:    "fileSystem",
		Slug:      "path-charset",
		Title:     "Path Charset",
		Summary:   "Constrain the characters allowed in the item's path.",
		Fields: []checks.Field{
			{Name: "deny", Required: false, Desc: "Forbidden substrings (e.g. a space). Use `deny` or `allow`, not both."},
			{Name: "allow", Required: false, Desc: "The only permitted characters; the path separator is always allowed."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_path_charset
        deny: [" "]`,
	}, func(ch config.CheckInstance) checks.Check {
		return PathCharset{Allow: ch.Allow, Deny: ch.Deny}
	}, nil)
}
