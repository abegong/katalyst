.PHONY: all build test vet fmt tidy run clean skills skill skills-link docs-deps docs-serve docs-build docs-pdf docs-gen docs-gen-check

BINARY := katalyst
DOCS_DIR := docs
DOCS_PDF_EXCLUDE ?=
DOCS_PDF_TONER_FRIENDLY ?= 0
DOCS_PDF_STANDARD := $(CURDIR)/$(DOCS_DIR)/public/katalyst-docs.pdf
DOCS_PDF_TONER := $(CURDIR)/$(DOCS_DIR)/public/katalyst-docs-toner-friendly.pdf
HUGO_BOOK_MODULE := github.com/alex-shpak/hugo-book
HUGO_LOCAL := $(shell command -v hugo 2>/dev/null)
HUGO_LOCAL_EXTENDED := $(shell hugo version 2>/dev/null | grep -q extended && echo 1 || echo 0)
ifeq ($(HUGO_LOCAL_EXTENDED),1)
HUGO := $(HUGO_LOCAL)
else
HUGO := go run -tags extended github.com/gohugoio/hugo@latest
endif

all: vet test build

build:
	go build -o bin/$(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

run:
	go run . $(ARGS)

clean:
	rm -rf bin

# Package the product skills under skills/ into bin/*.skill: one zip per
# shippable skill with SKILL.md at the archive root and the shared bootstrap
# bundled in. Skills marked `status: placeholder` are skipped. The .skill
# artifacts land in bin/, so `make clean` removes them. The release workflow
# runs this same target (see .goreleaser.yml) so local and CI packaging match.
skills:
	go run ./cmd/skillpack

# Package a single skill: make skill SKILL=katalyst-deploy
skill:
	@test -n "$(SKILL)" || { echo "usage: make skill SKILL=<name>" >&2; exit 2; }
	go run ./cmd/skillpack -skill $(SKILL)

# Symlink each skills/{name}/ into .claude/skills/ so they auto-load in a
# working copy. .gitignore already excludes all of .claude/, so the symlinks
# stay uncommitted.
skills-link:
	./scripts/link-product-skills.sh

# Docs are their own Hugo module under $(DOCS_DIR)/ (separate go.mod), so
# the application module's `go mod tidy` never strips the theme. Hugo
# invocations target that source root with -s, except `hugo mod get`, which
# forwards all trailing args straight to `go get` (so -s would reach go get
# and error) — it must run from within $(DOCS_DIR) instead.
# docs-gen regenerates the check-type reference from the checks registry.
docs-gen:
	go run ./cmd/gendocs

# docs-gen-check fails if the generated check-type or inspector reference, the
# embeddable worked-example snippets, or the mirrored governance pages, are out
# of date. Run in CI so a new check type or inspector, a behavior change that
# alters an example's output, or an edit to a root governance file can't ship
# without regenerating its docs.
docs-gen-check: docs-gen
	git diff --exit-code -- \
		docs/content/reference/check-types \
		docs/content/reference/inspectors \
		docs/generated/examples \
		docs/content/contributing/code-of-conduct.md \
		docs/content/contributing/security.md

docs-deps:
	cd $(DOCS_DIR) && $(HUGO) mod get -u $(HUGO_BOOK_MODULE)

docs-serve: docs-deps
	$(HUGO) server -s $(DOCS_DIR) --buildDrafts --disableFastRender

docs-build: docs-deps
	HUGO_DOCS_PDF_EXCLUDE="" HUGO_DOCS_PDF_TONER_FRIENDLY=0 $(HUGO) -s $(DOCS_DIR) --minify
	DOCS_PDF_OUTPUT="$(DOCS_PDF_STANDARD)" ./scripts/docs-pdf.sh
	HUGO_DOCS_PDF_EXCLUDE="" HUGO_DOCS_PDF_TONER_FRIENDLY=1 $(HUGO) -s $(DOCS_DIR) --minify --cleanDestinationDir=false
	DOCS_PDF_OUTPUT="$(DOCS_PDF_TONER)" ./scripts/docs-pdf.sh

# Export the whole docs site to PDF. DOCS_PDF_EXCLUDE is a comma-separated list
# of URL prefixes to omit. DOCS_PDF_TONER_FRIENDLY=1 prints code blocks with a
# white background instead of the theme's syntax-highlighted dark background.
# Example:
# make docs-pdf DOCS_PDF_EXCLUDE=/contributing/,/deep-dives/ DOCS_PDF_TONER_FRIENDLY=1
docs-pdf: docs-deps
	HUGO_DOCS_PDF_EXCLUDE="$(DOCS_PDF_EXCLUDE)" HUGO_DOCS_PDF_TONER_FRIENDLY="$(DOCS_PDF_TONER_FRIENDLY)" $(HUGO) -s $(DOCS_DIR) --baseURL / --minify
	./scripts/docs-pdf.sh
