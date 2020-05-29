VERSION=$(shell git describe --tags --candidates=1 --dirty)
BUILD_FLAGS=-v -trimpath
SRC=$(shell find . -name '*.go')
INSTALL_DIR ?= ./build
.PHONY: build clean release install

build: build/bctx-linux-amd64 build/bctx-linux-arm64 build/bctx-darwin-amd64

test:
	go test -v ./...

clean:
	rm -f ./build/bctx ./build/bctx-*-* ./build/SHA256SUMS

release: build build/SHA256SUMS

build/bctx-darwin-amd64: $(SRC)
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/**/*.go

build/bctx-linux-amd64: $(SRC)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/**/*.go

build/bctx-linux-arm64: $(SRC)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o $@ cmd/**/*.go

build/SHA256SUMS: build
	shasum -a 256 build/bctx-darwin-amd64 build/bctx-linux-amd64 build/bctx-linux-arm64 > $@

