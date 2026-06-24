package cmd_test

import "embed"

// Reusable test fixtures embedded from cmd/testdata/. See
// ../AGENTS.md ("Testing > Fixtures") for the inline-vs-fixture policy,
// and testdata/AGENTS.md for what each file is for.

//go:embed testdata/schemas/book.json
var bookSchemaFixture string

//go:embed testdata/schemas/person.json
var personSchemaFixture string

//go:embed testdata/schemas/strict-book.json
var strictBookSchemaFixture string

// snapshotFixtures embeds the whole golden-fixture tree. Reads go through the
// embed FS (not os.ReadFile) so they survive the per-test chdir into a temp
// dir; the snapshot harness in snapshot_test.go is the only consumer.
//
//go:embed testdata/snapshots
var snapshotFixtures embed.FS
