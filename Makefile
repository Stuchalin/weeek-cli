BINARY := weeek
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)
TARGET_OSES ?= darwin linux
TARGET_ARCHES ?= amd64 arm64

.PHONY: build test vet release-build check-plugin clean

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/weeek

test:
	go test ./...

vet:
	go vet ./...

release-build:
	mkdir -p dist
	@for os in $(TARGET_OSES); do \
		for arch in $(TARGET_ARCHES); do \
			echo "building $$os/$$arch"; \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build \
				-ldflags "$(LDFLAGS)" \
				-o "dist/weeek_$${os}_$${arch}" ./cmd/weeek || exit 1; \
		done; \
	done

check-plugin:
	scripts/check-plugin.sh
	scripts/check-plugin_test.sh

clean:
	rm -rf $(BINARY) dist
