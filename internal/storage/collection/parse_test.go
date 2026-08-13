package collection_test

import (
	"testing"

	"github.com/abegong/katalyst/internal/storage"
	"github.com/abegong/katalyst/internal/storage/collection"
)

func TestHasTextContent_filesystemCollectionExposesText(t *testing.T) {
	c := collection.Collection{Name: "notes", StorageType: string(storage.Filesystem)}
	if !c.HasTextContent() {
		t.Error("a filesystem collection's items are text by construction")
	}
}

func TestHasTextContent_sqliteWithContentColumnExposesText(t *testing.T) {
	c := collection.Collection{
		Name:          "notes",
		StorageType:   string(storage.SQLite),
		Table:         "notes",
		ContentColumn: "body",
	}
	if !c.HasTextContent() {
		t.Error("a sqlite collection mapping a content column has a body")
	}
}

func TestHasTextContent_sqliteWithoutContentColumnExposesNoText(t *testing.T) {
	c := collection.Collection{
		Name:        "notes",
		StorageType: string(storage.SQLite),
		Table:       "notes",
		Attributes:  map[string]collection.AttributeCapture{"title": {Column: "title"}},
	}
	if c.HasTextContent() {
		t.Error("a sqlite collection with attributes only has no body to rewrite")
	}
}
