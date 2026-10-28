#!/bin/sh
set -eu
SPEC_BUILD_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$SPEC_BUILD_ROOT"
SPEC_BUILD_VERSION=$(go run . --version)
mkdir -p dist
for SPEC_BUILD_OS in linux darwin windows; do
  for SPEC_BUILD_ARCH in amd64 arm64; do
    SPEC_BUILD_NAME="SPEC_${SPEC_BUILD_VERSION}_${SPEC_BUILD_OS}_${SPEC_BUILD_ARCH}"
    SPEC_BUILD_STAGE=$(mktemp -d)
    SPEC_BUILD_EXECUTABLE=spec
    if [ "$SPEC_BUILD_OS" = windows ]; then SPEC_BUILD_EXECUTABLE=spec.exe; fi
    CGO_ENABLED=0 GOOS="$SPEC_BUILD_OS" GOARCH="$SPEC_BUILD_ARCH" go build -trimpath -ldflags='-s -w -buildid=' -o "$SPEC_BUILD_STAGE/$SPEC_BUILD_EXECUTABLE" .
    cp LICENSE README.md THIRD_PARTY_NOTICES.md "$SPEC_BUILD_STAGE/"
    if [ "$SPEC_BUILD_OS" = windows ]; then
      (cd "$SPEC_BUILD_STAGE" && zip -q "$SPEC_BUILD_ROOT/dist/$SPEC_BUILD_NAME.zip" "$SPEC_BUILD_EXECUTABLE" LICENSE README.md THIRD_PARTY_NOTICES.md)
    else
      tar -C "$SPEC_BUILD_STAGE" -czf "dist/$SPEC_BUILD_NAME.tar.gz" "$SPEC_BUILD_EXECUTABLE" LICENSE README.md THIRD_PARTY_NOTICES.md
    fi
    rm -rf "$SPEC_BUILD_STAGE"
    echo "Built $SPEC_BUILD_NAME"
  done
done
cp install.sh install.ps1 dist/
(cd dist && sha256sum SPEC_"$SPEC_BUILD_VERSION"_*.tar.gz SPEC_"$SPEC_BUILD_VERSION"_*.zip install.sh install.ps1 > SHA256SUMS)
