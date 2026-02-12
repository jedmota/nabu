.PHONY: build test lint clean

BINARY := proxy-tui

build:
	./scripts/release.sh
	# go build -o $(BINARY) ./cmd/proxy-tui

run:
	./bin/proxy-linux-arm64

test:
	go test -race -count=1 ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
