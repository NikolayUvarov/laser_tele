#!/bin/sh
# Builds laser-mtproxy and mtprobe for the usual platforms into ./build
set -e
cd "$(dirname "$0")"
VERSION=$(sed -n 's/^const Version = "\(.*\)"/\1/p' internal/app/app.go)
mkdir -p build
for target in linux/amd64 linux/arm64 linux/arm windows/amd64 darwin/arm64; do
	os=${target%/*}
	arch=${target#*/}
	ext=""
	[ "$os" = windows ] && ext=".exe"
	out="build/laser-mtproxy-$VERSION-$os-$arch"
	mkdir -p "$out"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags "-s -w" -o "$out/laser-mtproxy$ext" .
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags "-s -w" -o "$out/mtprobe$ext" ./cmd/mtprobe
	cp dist/laser-mtproxy.json dist/laser-mtproxy.service README.md "$out/"
	echo "built $out"
done
