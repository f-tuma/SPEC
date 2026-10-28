#!/bin/sh
set -eu
SPEC_SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
if ! command -v go >/dev/null 2>&1; then
  echo "Source setup requires Go 1.26+. Alternatively download the native SPEC binary from GitHub Releases." >&2
  exit 2
fi
(cd "$SPEC_SCRIPT_DIR" && mkdir -p bin && CGO_ENABLED=0 go build -trimpath -o bin/spec .)
if [ "$#" -eq 0 ]; then
  exec "$SPEC_SCRIPT_DIR/bin/spec"
fi
exec "$SPEC_SCRIPT_DIR/bin/spec" install "$@"
