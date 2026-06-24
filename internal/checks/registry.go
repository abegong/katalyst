package checks

import (
	"errors"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// This file holds the check-type registry: the types describing a check type for
// documentation, and the Register/Parse/Build machinery check types self-register
// with from their family subpackages.
//
// Each check type owns its Descriptor, parser, and constructor and registers them
// from an init() in its own file, so adding a check type touches one file. The
// loader (config) parses each configured check through Parse at load time; the
// engine builds the runnable check via Build/BuildCollection; cmd/gendocs and
// `katalyst check-types` render the catalog from Descriptors() / Families().

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
	CheckType CheckType `json:"check_type"`
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

// ConfiguredCheck is one check as it sits on a collection after loading: its
// kind and the validated args its parser produced (parse, don't validate — by
// the time a ConfiguredCheck exists, its args are valid). The object check
// carries its Schema name instead; the engine builds it from a compiled schema.
type ConfiguredCheck struct {
	Kind   CheckType
	Args   any
	Schema string
}

// Parser decodes and validates a check's own arguments from its raw config node
// (the deferred yaml.Node for one `checks:` entry), returning the validated args
// as an opaque value the build functions consume. It is how a check type owns
// its config parsing.
type Parser func(*yaml.Node) (any, error)

// NoArgs is a Parser for a check type that takes no configuration: it ignores
// the node and yields an empty value the builder discards.
func NoArgs(*yaml.Node) (any, error) { return struct{}{}, nil }

// ParseInto builds a Parser that decodes the raw node into a fresh T and runs
// validate (nil for none). A nil node (no keys) yields the zero T, so validate
// sees the same empty value the loader would. The returned args are consumed by
// an ArgsBuilder/CollectionArgsBuilder that type-asserts T.
func ParseInto[T any](validate func(T) error) Parser {
	return func(n *yaml.Node) (any, error) {
		var a T
		if n != nil {
			if err := n.Decode(&a); err != nil {
				return nil, err
			}
		}
		if validate != nil {
			if err := validate(a); err != nil {
				return nil, err
			}
		}
		return a, nil
	}
}

// ArgsBuilder constructs a per-item Check from a Parser's validated args.
type ArgsBuilder func(any) Check

// CollectionArgsBuilder constructs a collection-scoped check from validated args.
type CollectionArgsBuilder func(any) CollectionCheck

// registration is one check type's registry entry: its Descriptor plus, for a
// configurable check, its parser and args-builders. A Descriptor-only entry (the
// object check) has nil parse/builders; the engine builds it specially.
type registration struct {
	desc          Descriptor
	parse         Parser
	buildArgs     ArgsBuilder
	buildCollArgs CollectionArgsBuilder
}

var (
	registrations []registration
	byKind        = map[CheckType]int{}
)

// RegisterParsed records a check type that owns its config parsing: parse
// decodes+validates the raw node into args, and buildArgs/buildCollArgs turn
// those args into the runnable check(s). One decode feeds both builders, so a
// dual (item + collection-scoped) check validates once. Either builder may be
// nil.
func RegisterParsed(desc Descriptor, parse Parser, buildArgs ArgsBuilder, buildCollArgs CollectionArgsBuilder) {
	register(registration{desc: desc, parse: parse, buildArgs: buildArgs, buildCollArgs: buildCollArgs})
}

// RegisterDescriptor records a check type that has only a Descriptor (no parser
// or builder): the object check, which the engine builds from a compiled schema.
// It still appears in the catalog and is a Known kind.
func RegisterDescriptor(desc Descriptor) { register(registration{desc: desc}) }

func register(r registration) {
	if _, dup := byKind[r.desc.CheckType]; dup {
		panic("checks: duplicate registration for kind " + string(r.desc.CheckType))
	}
	byKind[r.desc.CheckType] = len(registrations)
	registrations = append(registrations, r)
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

// Known reports whether kind is a registered check type.
func Known(kind CheckType) bool { _, ok := byKind[kind]; return ok }

// CollectionScoped reports whether kind runs once per collection (vs. per item).
func CollectionScoped(kind CheckType) bool {
	i, ok := byKind[kind]
	return ok && registrations[i].desc.Scope == "collection"
}

// Parse decodes and validates one configured check's arguments through the
// check type's own parser. It is called at load time, so a parse error fails
// config.Load. An unknown kind, or one with no parser (the object check, which
// the loader handles separately), is an error here.
func Parse(kind CheckType, node *yaml.Node) (any, error) {
	i, ok := byKind[kind]
	if !ok {
		if kind == "" {
			return nil, errors.New("check type is required")
		}
		return nil, fmt.Errorf("unknown check type %q", kind)
	}
	if registrations[i].parse == nil {
		return nil, fmt.Errorf("check type %q takes no configurable arguments here", kind)
	}
	return registrations[i].parse(node)
}

// Build constructs the per-item check from already-validated args (from Parse).
// ok is false for a kind with no per-item form (collection-scoped, or the object
// check the engine builds itself).
func Build(kind CheckType, args any) (Check, bool) {
	i, ok := byKind[kind]
	if !ok || registrations[i].buildArgs == nil {
		return nil, false
	}
	return registrations[i].buildArgs(args), true
}

// BuildCollection constructs the collection-scoped check from validated args, or
// returns (nil, false) for a per-item kind.
func BuildCollection(kind CheckType, args any) (CollectionCheck, bool) {
	i, ok := byKind[kind]
	if !ok || registrations[i].buildCollArgs == nil {
		return nil, false
	}
	return registrations[i].buildCollArgs(args), true
}
