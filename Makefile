BIN := edge-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build build-all install clean

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BIN) .

build-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BIN)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BIN)-linux-arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm    go build $(LDFLAGS) -o dist/$(BIN)-linux-arm   .

install: build
	cp $(BIN) /usr/local/bin/$(BIN)

clean:
	rm -f $(BIN)
	rm -rf dist/
