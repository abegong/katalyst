package cmd_test

import _ "embed"

// Reusable test fixtures embedded from cmd/testdata/. See
// ../AGENTS.md ("Testing > Fixtures") for the inline-vs-fixture policy,
// and testdata/README.md for what each file is for.

//go:embed testdata/schemas/book.json
var bookSchemaFixture string

//go:embed testdata/schemas/person.json
var personSchemaFixture string

//go:embed testdata/schemas/strict-book.json
var strictBookSchemaFixture string

//go:embed testdata/configs/book-and-person.yaml
var bookAndPersonConfigFixture string

//go:embed testdata/configs/strict-book.yaml
var strictBookConfigFixture string
