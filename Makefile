.PHONY: all build test vet fmt tidy run clean docs-serve docs-build

BINARY := katalyst
HUGO := $(shell command -v hugo 2>/dev/null)
ifeq ($(HUGO),)
HUGO := go run github.com/gohugoio/hugo@latest
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

docs-serve:
	$(HUGO) server --buildDrafts --disableFastRender

docs-build:
	$(HUGO) --minify
