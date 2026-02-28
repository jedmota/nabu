#!/usr/bin/env bash

set -e

VERSION=${VERSION:-"dev"}
BUILD_DIR="bin"
LDFLAGS="-X main.version=${VERSION}"

mkdir -p "$BUILD_DIR"

echo -n "Building version $VERSION to linux amd64... "
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o $BUILD_DIR/nabu-linux-amd64 cmd/nabu/main.go
echo "done!"

echo -n "Building version $VERSION to linux arm64... "
GOOS=linux GOARCH=arm64 go build -ldflags "$LDFLAGS" -o $BUILD_DIR/nabu-linux-arm64 cmd/nabu/main.go
echo "done!"

echo -n "Building version $VERSION to linux arm... "
GOOS=linux GOARCH=arm go build -ldflags "$LDFLAGS" -o $BUILD_DIR/nabu-linux-arm cmd/nabu/main.go
echo "done!"

echo -n "Building version $VERSION to darwin amd64... "
GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o $BUILD_DIR/nabu-darwin-amd64 cmd/nabu/main.go
echo "done!"

echo -n "Building version $VERSION to darwin arm64... "
GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o $BUILD_DIR/nabu-darwin-arm64 cmd/nabu/main.go
echo "done!"

cp LICENSE THIRD_PARTY_NOTICES "$BUILD_DIR/"

echo ""
find "$BUILD_DIR" -type f
