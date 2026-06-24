package checks

import (
	"sort"

	"github.com/abegong/katalyst/internal/project/config"
)

// This file holds the check-type registry: the types describing a check type
// for documentation, and the Register/Descriptors/Build machinery check types
// self-register with from their family subpackages.
//
// Each check type owns its Descriptor and registers it (along with a
// constructor) from an init() in its own file, so adding a check type touches
// one file instead of a central switch. cmd/engine builds the runnable check
// list by registry lookup (Build / BuildCollection); cmd/gendocs and
// `katalyst check-types` render the catalog from Descriptors() / Families().
// registry_test.go enforces parity with config.normalizeCheck so a check type
// cannot ship undocumented.

// Field describes one configuration key accepted by a check type. The json tags
// are the wire contract for `katalyst check-types list --json`; keep them
// snake_case (matching the config keys they describe) even if the Go field
// names change.
type Field struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Default  string `json:"default,omitempty"`
	Desc     string `json:"desc"`
}

// Descriptor is the machine-readable record for one check type. Its json tags
// are the wire contract for `katalyst check-types list --json`; see Field.
type Descriptor struct {
	// CheckType is the value used as `kind:` in a collection's checks.
	CheckType config.CheckType `json:"check_type"`
	// Library names the CheckLibrary that provides this check type
	// (Descriptor.Library == library.Name()), e.g. "json-schema". Empty until
	// a check type is migrated onto a library; resolved by LibraryFor. Library
	// is provenance, orthogonal to Family (source-data kind).
	Library string `json:"library,omitempty"`
	// Family groups the check type by source-data kind: "structuredObject",
	// "markdownBodyText", "fileSystem", or "plainText". Family and granularity
	// are orthogonal, a collection-scoped check is grouped by the data it
	// reads, not by its scope (e.g. unique_field is structuredObject).
	Family string `json:"family"`
	// Slug is the page basename under the family directory.
	Slug string `json:"slug"`
	// Title is the human-readable page title.
	Title string `json:"title"`
	// Summary is a one-line statement of what the check type enforces.
	Summary string `json:"summary"`
	// Fields documents the check type's configuration keys, if any. The
	// check-types command normalizes a nil slice to [] so consumers never see null.
	Fields []Field `json:"fields"`
	// ConfigExample is a complete config snippet (YAML, no fence)
	// showing the check in a collection.
	ConfigExample string `json:"config_example"`
	// Scope is "collection" for checks that run once per collection over all
	// its items; empty means an ordinary per-item check.
	Scope string `json:"scope,omitempty"`
	// Severity is "warning" for checks that emit advisory findings (never
	// failing the run); empty means the default, "error".
	Severity string `json:"severity,omitempty"`
}

// Family identifies a check-type family: its id (used in Descriptor.Family and
// `--family`), its docs-directory slug, and its intro copy.
type Family struct {
	ID    string
	Slug  string
	Title string
	Intro string
}

// Families returns the check-type families in display order. The order is
// significant: it fixes the section ordering in generated output and the
// family grouping in Descriptors().
func Families() []Family {
	return []Family{
		{
			ID:    "structuredObject",
			Slug:  "structured-object",
			Title: "Structured object check types",
			Intro: "Structured-object check types validate structured frontmatter fields using schema-backed checks.",
		},
		{
			ID:    "markdownBodyText",
			Slug:  "markdown-body-text",
			Title: "Markdown body text check types",
			Intro: "Markdown body-text check types validate relationships between frontmatter metadata and markdown body content.",
		},
		{
			ID:    "fileSystem",
			Slug:  "file-system",
			Title: "File system check types",
			Intro: "File-system check types validate filename and path conventions for items.",
		},
		{
			ID:    "plainText",
			Slug:  "plain-text",
			Title: "Plain text check types",
			Intro: "Plain-text check types validate body content as raw text, independent of markdown structure. They apply to plain-text items as well as markdown bodies.",
		},
	}
}

// Builder constructs a per-item Check from a normalized config instance. It is
// nil for check types that have no ordinary per-item form (collection-scoped
// checks, and the object check, which the engine builds specially because it
// needs a compiled schema).
type Builder func(config.CheckInstance) Check

// CollectionBuilder constructs a collection-scoped check. It is nil for
// per-item check types.
type CollectionBuilder func(config.CheckInstance) CollectionCheck

// registration is one check type's registry entry.
type registration struct {
	desc      Descriptor
	build     Builder
	buildColl CollectionBuilder
}

var (
	registrations []registration
	byKind        = map[config.CheckType]int{}
)

// Register records a check type: its Descriptor plus optional constructors. A
// check type calls this from an init() in its own file. build may be nil for a
// collection-scoped (or specially-built) check; buildColl may be nil for a
// per-item check. Duplicate kinds panic, a programming error caught at startup.
func Register(desc Descriptor, build Builder, buildColl CollectionBuilder) {
	if _, dup := byKind[desc.CheckType]; dup {
		panic("checks: duplicate registration for kind " + string(desc.CheckType))
	}
	byKind[desc.CheckType] = len(registrations)
	registrations = append(registrations, registration{desc, build, buildColl})
}

// Descriptors returns every registered check type, grouped by Families() order
// and, within a family, in registration order. The order is authored (it
// reflects code structure, not an alphabetical sort), so generated output is
// deterministic.
func Descriptors() []Descriptor {
	famPos := map[string]int{}
	for i, f := range Families() {
		famPos[f.ID] = i
	}
	ordered := make([]registration, len(registrations))
	copy(ordered, registrations)
	sort.SliceStable(ordered, func(i, j int) bool {
		return famPos[ordered[i].desc.Family] < famPos[ordered[j].desc.Family]
	})
	out := make([]Descriptor, len(ordered))
	for i, r := range ordered {
		out[i] = r.desc
	}
	return out
}

// Build constructs the per-item check for a configured instance. It returns
// (nil, false) for a kind with no per-item builder (collection-scoped, or the
// object check the engine builds itself), so callers skip it in the per-item
// loop.
func Build(ch config.CheckInstance) (Check, bool) {
	i, ok := byKind[ch.Type]
	if !ok || registrations[i].build == nil {
		return nil, false
	}
	return registrations[i].build(ch), true
}

// BuildCollection constructs the collection-scoped check for a configured
// instance, or returns (nil, false) for a per-item kind.
func BuildCollection(ch config.CheckInstance) (CollectionCheck, bool) {
	i, ok := byKind[ch.Type]
	if !ok || registrations[i].buildColl == nil {
		return nil, false
	}
	return registrations[i].buildColl(ch), true
}
