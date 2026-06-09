.PHONY: all build test vet fmt tidy run clean docs-deps docs-serve docs-build

BINARY := katalyst
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

docs-deps:
	$(HUGO) mod get -u $(HUGO_BOOK_MODULE)

docs-serve: docs-deps
	$(HUGO) server --buildDrafts --disableFastRender

docs-build: docs-deps
	$(HUGO) --minify
