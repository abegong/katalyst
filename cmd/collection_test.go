package cmd_test

import (
	"errors"
	"path/filepath"
	"testing"
)

// The list/get text contracts are pinned as snapshots; the exit-code behavior
// stays a property test. See cmd/AGENTS.md ("Testing the CLI").

func TestCollectionList_showsNamePathCountSchema(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)
	// One item in the books collection; people stays empty.
	mustWrite(t, filepath.Join(dir, "notes/books/dune.md"), "---\ntitle: Dune\nyear: 1965\n---\n# Dune\n")

	stdout, _, err := runRoot(t, "collection", "list")
	if err != nil {
		t.Fatalf("collection list: %v", err)
	}
	snapshot(t, "collection/list.txt", stdout)
}

func TestCollectionGet_showsDetail(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "notes/books/dune.md"), "---\ntitle: Dune\nyear: 1965\n---\n# Dune\n")

	stdout, _, err := runRoot(t, "collection", "get", "books")
	if err != nil {
		t.Fatalf("collection get: %v", err)
	}
	snapshot(t, "collection/get.txt", stdout)
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
