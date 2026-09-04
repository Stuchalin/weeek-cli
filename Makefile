BINARY := weeek
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test vet release-build check-plugin clean

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/weeek

test:
	go test ./...

vet:
	go vet ./...

release-build:
	mkdir -p dist
	@for os in darwin linux; do \
		for arch in amd64 arm64; do \
			echo "building $$os/$$arch"; \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build \
				-ldflags "$(LDFLAGS)" \
				-o "dist/weeek_$${os}_$${arch}" ./cmd/weeek || exit 1; \
		done; \
	done

check-plugin:
	@if [ -x scripts/check-plugin.sh ]; then \
		scripts/check-plugin.sh; \
	else \
		echo "check-plugin is not implemented yet"; \
	fi

clean:
	rm -rf $(BINARY) dist
