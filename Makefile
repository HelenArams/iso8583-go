GOLANGCI_VERSION := v1.59.1
TOOLS := $(CURDIR)/.tools/bin

.PHONY: all test cover bench fuzz lint fmt build clean

all: fmt lint test

build:
	go build -o bin/ ./cmd/...

test:
	go test -race -cover ./...

cover:
	go test -coverprofile=cover.out ./...
	go tool cover -func=cover.out | tail -1

bench:
	go test -run '^$$' -bench . -benchmem .

fuzz:
	go test -run '^$$' -fuzz FuzzUnpack -fuzztime 60s .

fmt:
	gofmt -l -w .

lint: $(TOOLS)/golangci-lint
	$(TOOLS)/golangci-lint run

$(TOOLS)/golangci-lint:
	GOBIN=$(TOOLS) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION)

clean:
	rm -rf bin cover.out
