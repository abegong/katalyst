.PHONY: all build test vet fmt tidy run clean

BINARY := katalyst

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
