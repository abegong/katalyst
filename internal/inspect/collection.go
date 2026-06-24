package inspect

import (
	"os"

	"github.com/abegong/katalyst/internal/project"
	"github.com/abegong/katalyst/internal/storage/collection"
	"github.com/abegong/katalyst/internal/storage/collection/document"
)

// CollectionView is the collection layer's addressing surface: a resolved
// collection and its items, parsed once. Items are addressed by domain identity
// (their Item.ID), and the bytes are reached through the project's
// CollectionDefinition, collection inspectors never see a raw path. Parsing
// here is a thin local adapter over document.Parse; it deliberately does not
// reach into internal/checks.
type CollectionView struct {
	collection project.Collection
	items      []collection.Item
	// docs is aligned with items; an entry is nil when the item could not be
	// read or parsed, so a broken item contributes nothing rather than panicking.
	docs []*document.Document
}

// NewCollectionView resolves a collection's items via the project and parses
// each once.
func NewCollectionView(proj *project.Project, c project.Collection) (CollectionView, error) {
	items, err := proj.Items(c)
	if err != nil {
		return CollectionView{}, err
	}
	docs := make([]*document.Document, len(items))
	for i, it := range items {
		src, err := os.ReadFile(it.Path)
		if err != nil {
			continue
		}
		if doc, err := document.Parse(src); err == nil {
			docs[i] = doc
		}
	}
	return CollectionView{collection: c, items: items, docs: docs}, nil
}

// Collection returns the collection this view describes.
func (v CollectionView) Collection() project.Collection { return v.collection }

// N is the number of items in the collection (the evidence denominator).
func (v CollectionView) N() int { return len(v.items) }

// IDs returns the item identifiers (collection-relative), in resolution order.
func (v CollectionView) IDs() []string {
	ids := make([]string, len(v.items))
	for i, it := range v.items {
		ids[i] = it.ID
	}
	return ids
}

// Frontmatter returns the frontmatter map of every item that carries one, the
// input to the object_fields primitive.
func (v CollectionView) Frontmatter() []map[string]any {
	var out []map[string]any
	for _, d := range v.docs {
		if d != nil && d.HasFrontmatter {
			out = append(out, d.Meta)
		}
	}
	return out
}

// Bodies returns the body and title of every parsed item, the input to the
// markdown_body primitive.
func (v CollectionView) Bodies() []mdInput {
	var out []mdInput
	for _, d := range v.docs {
		if d == nil {
			continue
		}
		title, _ := d.Meta["title"].(string)
		out = append(out, mdInput{Body: d.Body, Title: title})
	}
	return out
}
