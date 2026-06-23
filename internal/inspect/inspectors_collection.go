package inspect

// ObjectFields is the collection-layer data-dictionary inspector: it runs the
// object_fields primitive over a collection's item frontmatter. It subsumes the
// five former object_field_* inspectors as columns of one table.
type ObjectFields struct{}

func (ObjectFields) Name() string { return "object_fields" }

func (ObjectFields) Inspect(v CollectionView, _ Params) Evidence {
	return Evidence{
		Inspector: "object_fields",
		Scope:     v.collection.Name,
		N:         v.N(),
		Data:      objectFields(v.Frontmatter()),
	}
}

// MarkdownBody is the collection-layer body inspector: it runs the markdown_body
// primitive over a collection's item bodies. It subsumes the former markdown_*
// inspectors as facets of one walk.
type MarkdownBody struct{}

func (MarkdownBody) Name() string { return "markdown_body" }

func (MarkdownBody) Inspect(v CollectionView, _ Params) Evidence {
	return Evidence{
		Inspector: "markdown_body",
		Scope:     v.collection.Name,
		N:         v.N(),
		Data:      markdownBody(v.Bodies()),
	}
}
