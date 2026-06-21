package checks

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Target names the slice of an item's path a name/path rule tests. The zero
// value ("") means TargetFilename.
const (
	TargetFilename     = "filename"      // basename without extension
	TargetFilenameExt  = "filename-ext"  // basename with extension
	TargetParentDir    = "parent-dir"    // immediate parent directory name
	TargetPathSegments = "path-segments" // every dir segment + the basename
)

// resolveTarget returns the path slice(s) a rule tests for the given target.
// path-segments is inclusive: every directory segment from the collection
// root down, plus the basename without extension. Other targets return a
// single value.
func resolveTarget(ctx Context, target string) []string {
	fileName := filepath.Base(ctx.FilePath)
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	switch target {
	case TargetFilenameExt:
		return []string{fileName}
	case TargetParentDir:
		return []string{filepath.Base(filepath.Dir(ctx.FilePath))}
	case TargetPathSegments:
		return pathSegments(ctx)
	default: // TargetFilename
		return []string{base}
	}
}

// pathSegments returns each directory segment of the collection-relative
// path plus the basename without extension. Segments are computed relative
// to CollectionRoot when set, otherwise relative to the file's own parent.
func pathSegments(ctx Context) []string {
	rel := ctx.FilePath
	if ctx.CollectionRoot != "" {
		if r, err := filepath.Rel(ctx.CollectionRoot, ctx.FilePath); err == nil {
			rel = r
		}
	}
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	out := make([]string, 0, len(parts))
	for i, p := range parts {
		if p == "" || p == "." || p == ".." {
			continue
		}
		if i == len(parts)-1 {
			p = strings.TrimSuffix(p, filepath.Ext(p))
		}
		out = append(out, p)
	}
	return out
}

// targetNoun is the human label for a target, used in messages.
func targetNoun(target string) string {
	switch target {
	case TargetFilenameExt:
		return "filename"
	case TargetParentDir:
		return "parent directory"
	case TargetPathSegments:
		return "path segment"
	default:
		return "filename"
	}
}

// caseStyle pairs a style's anchored pattern with its human label.
type caseStyle struct {
	pattern *regexp.Regexp
	label   string
}

var caseStyles = map[string]caseStyle{
	"kebab":           {regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`), "kebab-case"},
	"snake":           {regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`), "snake_case"},
	"screaming-snake": {regexp.MustCompile(`^[A-Z0-9]+(?:_[A-Z0-9]+)*$`), "SCREAMING_SNAKE_CASE"},
	"camel":           {regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`), "camelCase"},
	"pascal":          {regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`), "PascalCase"},
	"point":           {regexp.MustCompile(`^[a-z0-9]+(?:\.[a-z0-9]+)*$`), "point.case"},
	"lower":           {regexp.MustCompile(`^[^A-Z]*$`), "lowercase"},
}

// CaseStyles returns the set of supported style keys (used by config
// validation).
func CaseStyles() []string {
	return []string{"kebab", "snake", "screaming-snake", "camel", "pascal", "point", "lower"}
}

// NameCase checks that the target conforms to a case style.
type NameCase struct {
	Style  string
	Target string
}

func (c NameCase) Run(ctx Context) []Violation {
	style, ok := caseStyles[c.Style]
	if !ok {
		return nil // unknown style is rejected at config load
	}
	var out []Violation
	noun := targetNoun(c.Target)
	for _, v := range resolveTarget(ctx, c.Target) {
		if !style.pattern.MatchString(v) {
			out = append(out, Violation{
				Path:    "/",
				Message: fmt.Sprintf("%s %q must be %s", noun, v, style.label),
			})
		}
	}
	return out
}

// NameMatchesField checks that the target equals a frontmatter field,
// optionally after a transform.
type NameMatchesField struct {
	Field     string
	Transform string
	Target    string
}

func (c NameMatchesField) Run(ctx Context) []Violation {
	ptr := "/" + c.Field
	raw, ok := ctx.Meta[c.Field]
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing frontmatter field %q", c.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	want, ok := raw.(string)
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("frontmatter field %q must be a string", c.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	if c.Transform == "slugify" {
		want = slugify(want)
	}
	got := resolveTarget(ctx, c.Target)[0]
	if got == want {
		return nil
	}
	return []Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("%s %q must match field %q (%q)", targetNoun(c.Target), got, c.Field, want),
		Line:    lookupLine(ctx.Doc.Lines, ptr),
	}}
}

var nonSlugRun = regexp.MustCompile(`[^a-z0-9]+`)

// slugify lowercases and kebab-cases a string: runs of non-alphanumerics
// collapse to a single hyphen, and leading/trailing hyphens are trimmed.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonSlugRun.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// NameAffix checks that the target starts with Prefix and/or ends with Suffix.
type NameAffix struct {
	Prefix string
	Suffix string
	Target string
}

func (c NameAffix) Run(ctx Context) []Violation {
	v := resolveTarget(ctx, c.Target)[0]
	noun := targetNoun(c.Target)
	var out []Violation
	if c.Prefix != "" && !strings.HasPrefix(v, c.Prefix) {
		out = append(out, Violation{
			Path:    "/",
			Message: fmt.Sprintf("%s %q must start with prefix %q", noun, v, c.Prefix),
		})
	}
	if c.Suffix != "" && !strings.HasSuffix(v, c.Suffix) {
		out = append(out, Violation{
			Path:    "/",
			Message: fmt.Sprintf("%s %q must end with suffix %q", noun, v, c.Suffix),
		})
	}
	return out
}

// PathCharset constrains the characters of the collection-relative path.
// Exactly one of Allow / Deny is set (enforced at config load). Deny lists
// forbidden substrings; Allow lists the only permitted characters (the path
// separator is always allowed).
type PathCharset struct {
	Allow []string
	Deny  []string
}

func (c PathCharset) Run(ctx Context) []Violation {
	path := filepath.ToSlash(ctx.FilePath)
	if ctx.CollectionRoot != "" {
		if r, err := filepath.Rel(ctx.CollectionRoot, ctx.FilePath); err == nil {
			path = filepath.ToSlash(r)
		}
	}
	if len(c.Deny) > 0 {
		var out []Violation
		for _, d := range c.Deny {
			if d != "" && strings.Contains(path, d) {
				out = append(out, Violation{
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
			return []Violation{{
				Path:    "/",
				Message: fmt.Sprintf("file path contains disallowed character %q", string(r)),
			}}
		}
	}
	return nil
}

// FilesystemExtensionIn checks that extension is in an allowed set.
type FilesystemExtensionIn struct {
	Values []string
}

func (f FilesystemExtensionIn) Run(ctx Context) []Violation {
	ext := strings.ToLower(filepath.Ext(ctx.FilePath))
	for _, allowed := range f.Values {
		a := strings.ToLower(strings.TrimSpace(allowed))
		if !strings.HasPrefix(a, ".") {
			a = "." + a
		}
		if ext == a {
			return nil
		}
	}
	return []Violation{{
		Path:    "/",
		Message: fmt.Sprintf("file extension %q is not in allowed set", ext),
	}}
}

// FilesystemParentDirIn checks that parent directory name is in allowed values.
type FilesystemParentDirIn struct {
	Values []string
}

func (f FilesystemParentDirIn) Run(ctx Context) []Violation {
	parent := filepath.Base(filepath.Dir(ctx.FilePath))
	for _, allowed := range f.Values {
		if parent == allowed {
			return nil
		}
	}
	return []Violation{{
		Path:    "/",
		Message: fmt.Sprintf("parent directory %q is not in allowed set", parent),
	}}
}
