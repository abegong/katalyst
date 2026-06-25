package structuredobject

import (
	"fmt"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
)

// uniqueFieldArgs is filesystem_unique_field's own config shape.
type uniqueFieldArgs struct {
	Field string `yaml:"field"`
}

// UniqueField requires that no two items share a value for Field. It is
// collection-scoped (it reasons across siblings) but belongs to the
// structuredObject family because it reads a frontmatter field; scope and
// family are orthogonal. Its kind keeps the historical filesystem_ prefix.
type UniqueField struct {
	Field string
}

func (c UniqueField) RunCollection(ctx checks.CollectionContext) []checks.Violation {
	groups := map[string][]string{}
	for _, it := range ctx.Items {
		raw, ok := it.Meta[c.Field]
		if !ok {
			continue
		}
		s, ok := raw.(string)
		if !ok {
			continue
		}
		groups[s] = append(groups[s], it.FilePath)
	}
	return checks.CollisionViolations(groups, fmt.Sprintf("%s value", c.Field))
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckFilesystemUniqueField,
		Family:    "structuredObject",
		Slug:      "unique-field",
		Title:     "Unique field",
		Summary:   "Require that no two items share a value for a frontmatter field.",
		Scope:     "collection",
		Fields: []checks.Field{
			{Name: "field", Required: true, Desc: "Frontmatter key whose value must be unique across the collection."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_unique_field
        field: slug`,
	}, checks.ParseInto(func(a uniqueFieldArgs) error {
		return argcheck.RequireString("filesystem_unique_field", "field", a.Field)
	}), nil, func(a any) checks.CollectionCheck {
		return UniqueField{Field: a.(uniqueFieldArgs).Field}
	})
}
