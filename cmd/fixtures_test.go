package cmd_test

import (
	"embed"
	"testing"
)

// Reusable test fixtures embedded from cmd/testdata/. See
// ../AGENTS.md ("Testing > Fixtures") for the inline-vs-fixture policy,
// and testdata/README.md for what each file is for.

//go:embed testdata/schemas/book.json
var bookSchemaFixture string

//go:embed testdata/schemas/person.json
var personSchemaFixture string

//go:embed testdata/schemas/strict-book.json
var strictBookSchemaFixture string

//go:embed testdata/help/*.txt
var helpFixtures embed.FS

func mustHelpFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := helpFixtures.ReadFile("testdata/help/" + name)
	if err != nil {
		t.Fatalf("read help fixture %q: %v", name, err)
	}
	return string(b)
}
