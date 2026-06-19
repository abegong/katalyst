package cmd_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectionList_showsNamePathCountSchema(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)
	// One item in the books collection.
	mustWrite(t, filepath.Join(dir, "notes/books/dune.md"), "---\ntitle: Dune\nyear: 1965\n---\n# Dune\n")

	stdout, _, err := runRoot(t, "collection", "list")
	if err != nil {
		t.Fatalf("collection list: %v", err)
	}
	if !strings.Contains(stdout, "books") || !strings.Contains(stdout, "people") {
		t.Errorf("expected both collections listed, got: %q", stdout)
	}
	if !strings.Contains(stdout, "notes/books") {
		t.Errorf("expected directory column, got: %q", stdout)
	}
	if !strings.Contains(stdout, "book") {
		t.Errorf("expected schema column, got: %q", stdout)
	}
}

func TestCollectionGet_showsDetail(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "notes/books/dune.md"), "---\ntitle: Dune\nyear: 1965\n---\n# Dune\n")

	stdout, _, err := runRoot(t, "collection", "get", "books")
	if err != nil {
		t.Fatalf("collection get: %v", err)
	}
	for _, want := range []string{"books", "notes/books", "*.md", "book", "items:   1"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in output, got: %q", want, stdout)
		}
	}
}

func TestCollectionGet_unknown_exit2(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)
	_, _, err := runRoot(t, "collection", "get", "ghosts")
	if err == nil {
		t.Fatalf("expected error for unknown collection")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestCollectionGet_wrongDepth_exit2(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)
	_, _, err := runRoot(t, "collection", "get", "books/dune")
	if err == nil {
		t.Fatalf("expected wrong-depth usage error")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}
