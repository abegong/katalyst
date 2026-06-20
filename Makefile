.PHONY: all build test vet fmt tidy run clean docs-deps docs-serve docs-build docs-gen docs-gen-check

BINARY := katalyst
DOCS_DIR := docs
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

# Docs are their own Hugo module under $(DOCS_DIR)/ (separate go.mod), so
# the application module's `go mod tidy` never strips the theme. Hugo
# invocations target that source root with -s, except `hugo mod get`, which
# forwards all trailing args straight to `go get` (so -s would reach go get
# and error) — it must run from within $(DOCS_DIR) instead.
# docs-gen regenerates the rule reference from the checks registry.
docs-gen:
	go run ./cmd/gendocs

# docs-gen-check fails if the generated rule reference is out of date.
# Run in CI so a new check can't ship without its generated page.
docs-gen-check: docs-gen
	git diff --exit-code -- docs/content/reference/rules

docs-deps:
	cd $(DOCS_DIR) && $(HUGO) mod get -u $(HUGO_BOOK_MODULE)

docs-serve: docs-deps
	$(HUGO) server -s $(DOCS_DIR) --buildDrafts --disableFastRender

docs-build: docs-deps
	$(HUGO) -s $(DOCS_DIR) --minify
