#!/bin/sh
# Native integration: package the already-built CLI, then install without Go.
set -eu
SPEC_TEST_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SPEC_TEST_BINARY=${1:-$SPEC_TEST_ROOT/bin/SPEC}
SPEC_TEST_VERSION=$("$SPEC_TEST_BINARY" --version)
case "$(uname -s)" in Linux) SPEC_TEST_OS=linux ;; Darwin) SPEC_TEST_OS=darwin ;; *) exit 1 ;; esac
case "$(uname -m)" in x86_64|amd64) SPEC_TEST_ARCH=amd64 ;; arm64|aarch64) SPEC_TEST_ARCH=arm64 ;; *) exit 1 ;; esac
SPEC_TEST_TEMP=$(mktemp -d)
trap 'rm -rf -- "$SPEC_TEST_TEMP"' 0
mkdir "$SPEC_TEST_TEMP/packages" "$SPEC_TEST_TEMP/payload"
cp -- "$SPEC_TEST_BINARY" "$SPEC_TEST_TEMP/payload/spec"
cp -- "$SPEC_TEST_ROOT/LICENSE" "$SPEC_TEST_ROOT/README.md" "$SPEC_TEST_ROOT/THIRD_PARTY_NOTICES.md" "$SPEC_TEST_TEMP/payload/"
SPEC_TEST_ASSET=SPEC_${SPEC_TEST_VERSION}_${SPEC_TEST_OS}_${SPEC_TEST_ARCH}.tar.gz
tar -C "$SPEC_TEST_TEMP/payload" -czf "$SPEC_TEST_TEMP/packages/$SPEC_TEST_ASSET" spec LICENSE THIRD_PARTY_NOTICES.md README.md
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$SPEC_TEST_TEMP/packages" && sha256sum "$SPEC_TEST_ASSET" > SHA256SUMS)
else
  (cd "$SPEC_TEST_TEMP/packages" && shasum -a 256 "$SPEC_TEST_ASSET" > SHA256SUMS)
fi
(cd "$SPEC_TEST_TEMP" && sh "$SPEC_TEST_ROOT/install.sh" --from-dir packages --bin-dir "installed bin")
test "$("$SPEC_TEST_TEMP/installed bin/spec" --version)" = "$SPEC_TEST_VERSION"
test -f "$SPEC_TEST_TEMP/installed bin/spec.LICENSE"
test -f "$SPEC_TEST_TEMP/installed bin/spec.THIRD_PARTY_NOTICES.md"
echo 'Native POSIX binary installer and caller-relative paths: PASS'
