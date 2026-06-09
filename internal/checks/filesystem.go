package checks

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// FilenameMatchesSlug checks that the markdown filename equals the slug value.
type FilenameMatchesSlug struct {
	Field string
}

func (f FilenameMatchesSlug) Run(ctx Context) []Violation {
	ptr := "/" + f.Field
	raw, ok := ctx.Meta[f.Field]
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing frontmatter field %q", f.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	slug, ok := raw.(string)
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("frontmatter field %q must be a string", f.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	fileName := filepath.Base(ctx.FilePath)
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if base == slug {
		return nil
	}
	return []Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("slug %q must match filename %q", slug, base),
		Line:    lookupLine(ctx.Doc.Lines, ptr),
	}}
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

var kebabCasePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// FilesystemFilenameKebabCase checks that basename is lowercase kebab-case.
type FilesystemFilenameKebabCase struct{}

func (f FilesystemFilenameKebabCase) Run(ctx Context) []Violation {
	base := strings.TrimSuffix(filepath.Base(ctx.FilePath), filepath.Ext(ctx.FilePath))
	if kebabCasePattern.MatchString(base) {
		return nil
	}
	return []Violation{{
		Path:    "/",
		Message: fmt.Sprintf("filename %q must be lowercase kebab-case", base),
	}}
}

// FilesystemNoSpacesInPath checks that full path has no spaces.
type FilesystemNoSpacesInPath struct{}

func (f FilesystemNoSpacesInPath) Run(ctx Context) []Violation {
	if !strings.Contains(ctx.FilePath, " ") {
		return nil
	}
	return []Violation{{
		Path:    "/",
		Message: "file path must not contain spaces",
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

// FilesystemFilenamePrefix checks that basename starts with a required prefix.
type FilesystemFilenamePrefix struct {
	Value string
}

func (f FilesystemFilenamePrefix) Run(ctx Context) []Violation {
	base := strings.TrimSuffix(filepath.Base(ctx.FilePath), filepath.Ext(ctx.FilePath))
	if strings.HasPrefix(base, f.Value) {
		return nil
	}
	return []Violation{{
		Path:    "/",
		Message: fmt.Sprintf("filename %q must start with prefix %q", base, f.Value),
	}}
}
