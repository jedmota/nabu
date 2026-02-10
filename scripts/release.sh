#!/usr/bin/env bash

set -e

VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="bin"

mkdir -p "$BUILD_DIR"

echo -n "Building version $VERSION to linux amd64... "
GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/proxy-linux-amd64 cmd/proxy-tui/main.go
echo "done!"

echo -n "Building version $VERSION to linux arm64... "
GOOS=linux GOARCH=arm64 go build -o $BUILD_DIR/proxy-linux-arm64 cmd/proxy-tui/main.go
echo "done!"

echo -n "Building version $VERSION to linux arm... "
GOOS=linux GOARCH=arm go build -o $BUILD_DIR/proxy-linux-arm cmd/proxy-tui/main.go
echo "done!"

echo -n "Building version $VERSION to darwin amd64... "
GOOS=darwin GOARCH=amd64 go build -o $BUILD_DIR/proxy-darwin-amd64 cmd/proxy-tui/main.go
echo "done!"

echo -n "Building version $VERSION to darwin arm64... "
GOOS=darwin GOARCH=arm64 go build -o $BUILD_DIR/proxy-darwin-arm64 cmd/proxy-tui/main.go
echo "done!"

cp LICENSE THIRD_PARTY_NOTICES "$BUILD_DIR/"

echo ""
find "$BUILD_DIR" -type f
