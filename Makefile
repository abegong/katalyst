.PHONY: all build test vet fmt tidy run clean docs-deps docs-serve docs-build

BINARY := katalyst
DOCS_DIR := docs
HUGO_BOOK_MODULE := github.com/alex-shpak/hugo-book
HUGO_LOCAL := $(shell command -v hugo 2>/dev/null)
HUGO_LOCAL_EXTENDED := $(shell if [ -n "$(HUGO_LOCAL)" ]; then case "$$($(HUGO_LOCAL) version 2>/dev/null)" in *extended*) echo 1 ;; *) echo 0 ;; esac; else echo 0; fi)
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
# the application module's `go mod tidy` never strips the theme. All Hugo
# invocations target that source root with -s.
docs-deps:
	$(HUGO) mod get -u $(HUGO_BOOK_MODULE) -s $(DOCS_DIR)

docs-serve: docs-deps
	$(HUGO) server -s $(DOCS_DIR) --buildDrafts --disableFastRender

docs-build: docs-deps
	$(HUGO) -s $(DOCS_DIR) --minify
