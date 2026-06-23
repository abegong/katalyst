package cmd_test

import (
	"errors"
	"strings"
	"testing"
)

func TestSchemaList_printsSortedNamesAndPaths(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)

	stdout, _, err := runRoot(t, "schema", "list")
	if err != nil {
		t.Fatalf("schema list: %v", err)
	}
	// The fixture pins the sorted names and their paths.
	snapshot(t, "schema/list.txt", stdout)
}

func TestSchemaList_noConfig(t *testing.T) {
	chdir(t, t.TempDir())

	_, _, err := runRoot(t, "schema", "list")
	if err == nil {
		t.Fatalf("expected error when no config found")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2 (usage), got: %v", err)
	}
}

func TestSchemaShow_printsSchemaContents(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)

	stdout, _, err := runRoot(t, "schema", "show", "book")
	if err != nil {
		t.Fatalf("schema show: %v", err)
	}
	snapshot(t, "schema/show-book.txt", stdout)
}

func TestSchemaShow_unknownName(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)

	_, _, err := runRoot(t, "schema", "show", "nonexistent")
	if err == nil {
		t.Fatalf("expected error for unknown schema")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention the missing name, got: %v", err)
	}
}
