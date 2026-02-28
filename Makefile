.PHONY: build test lint clean

BINARY := nabu
VERSION ?= dev
LDFLAGS := -X main.version=$(VERSION)

build:
	./scripts/release.sh
	# go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/nabu

run:
	./bin/nabu-linux-arm64

test:
	go test -race -count=1 ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
