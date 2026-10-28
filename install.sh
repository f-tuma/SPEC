#!/bin/sh
# Download a release binary. No Go, Node or Python runtime is needed.
set -eu

spec_fail() { printf 'SPEC installer: %s\n' "$*" >&2; exit 1; }
spec_need() { command -v "$1" >/dev/null 2>&1 || spec_fail "Required command not found: $1"; }
spec_valid_version() { printf '%s\n' "$1" | LC_ALL=C grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; }
spec_https() {
  curl --fail --silent --show-error --location --retry 2 --connect-timeout 15 --max-time 180 --proto '=https' --tlsv1.2 --output "$2" "$1"
}
spec_cleanup() {
  SPEC_EXIT_CODE=$?
  trap - 0 HUP INT TERM
  set +e
  SPEC_RECOVERY_FAILED=0
  if [ "$SPEC_EXIT_CODE" -ne 0 ]; then
    for SPEC_RESTORE_NAME in ${SPEC_INSTALLED-}; do
      if [ -f "$SPEC_STAGE/old/$SPEC_RESTORE_NAME" ]; then
        mv -f -- "$SPEC_STAGE/old/$SPEC_RESTORE_NAME" "$SPEC_BIN_DIR/$SPEC_RESTORE_NAME" || SPEC_RECOVERY_FAILED=1
      else
        rm -f -- "$SPEC_BIN_DIR/$SPEC_RESTORE_NAME" || SPEC_RECOVERY_FAILED=1
      fi
    done
  fi
  if [ -n "${SPEC_STAGE-}" ]; then
    if [ "$SPEC_RECOVERY_FAILED" -eq 0 ]; then rm -rf -- "$SPEC_STAGE"
    else printf 'Recovery incomplete; backups retained at %s\n' "$SPEC_STAGE" >&2; fi
  fi
  if [ "${SPEC_LOCKED-0}" -eq 1 ]; then rmdir -- "$SPEC_BIN_DIR/.spec-install.lock"; fi
  if [ -n "${SPEC_TEMP-}" ]; then rm -rf -- "$SPEC_TEMP"; fi
  exit "$SPEC_EXIT_CODE"
}
spec_main() {
  SPEC_REPO=f-tuma/SPEC
  SPEC_VERSION=
  SPEC_FROM_DIR=
  SPEC_TRANSPORT=auto
  SPEC_BIN_DIR=${HOME:?HOME must identify your user directory}/.local/bin
  SPEC_TEMP=
  SPEC_STAGE=
  SPEC_INSTALLED=
  SPEC_LOCKED=0
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --version|--bin-dir|--from-dir|--transport)
        [ "$#" -ge 2 ] && [ -n "$2" ] || spec_fail "$1 requires a value"
        case "$1" in
          --version) SPEC_VERSION=$2 ;;
          --bin-dir) SPEC_BIN_DIR=$2 ;;
          --from-dir) SPEC_FROM_DIR=$2 ;;
          --transport) SPEC_TRANSPORT=$2 ;;
        esac
        shift 2 ;;
      -h|--help)
        cat <<'SPEC_HELP'
Install a prebuilt SPEC binary.
  sh install.sh [--version vX.Y.Z] [--bin-dir DIR]
  sh install.sh --from-dir DIR [--version vX.Y.Z] [--bin-dir DIR]
  --transport auto|gh|https   Authenticated GitHub CLI or public HTTPS
Default: latest stable release, ~/.local/bin. No shell startup files are changed.
Local mode selects one matching platform archive and requires SHA256SUMS beside it.
Project setup is separate: spec install --project DIR
SPEC_HELP
        return ;;
      *) spec_fail "Unknown option: $1 (use --help)" ;;
    esac
  done
  case "$SPEC_TRANSPORT" in auto|gh|https) ;; *) spec_fail "Invalid transport" ;; esac
  case "$(uname -s)" in Linux) SPEC_OS=linux ;; Darwin) SPEC_OS=darwin ;; *) spec_fail "Use install.ps1 on Windows; this system is unsupported" ;; esac
  case "$(uname -m)" in x86_64|amd64) SPEC_ARCH=amd64 ;; arm64|aarch64) SPEC_ARCH=arm64 ;; *) spec_fail "Supported architectures: amd64 and arm64" ;; esac
  if [ -n "$SPEC_VERSION" ]; then
    case "$SPEC_VERSION" in v*) ;; *) SPEC_VERSION=v$SPEC_VERSION ;; esac
    spec_valid_version "$SPEC_VERSION" || spec_fail "Use a stable version such as v0.3.0"
  fi
  if [ -n "$SPEC_FROM_DIR" ]; then
    [ -d "$SPEC_FROM_DIR" ] || spec_fail "Local package directory does not exist"
    SPEC_FROM_DIR=$(CDPATH= cd -- "$SPEC_FROM_DIR" && pwd -P)
    if [ -z "$SPEC_VERSION" ]; then
      SPEC_MATCHES=0
      for SPEC_CANDIDATE in "$SPEC_FROM_DIR"/SPEC_*_"$SPEC_OS"_"$SPEC_ARCH".tar.gz; do
        [ -f "$SPEC_CANDIDATE" ] || continue
        SPEC_CANDIDATE_NAME=${SPEC_CANDIDATE##*/}
        SPEC_CANDIDATE_VERSION=${SPEC_CANDIDATE_NAME#SPEC_}
        SPEC_CANDIDATE_VERSION=${SPEC_CANDIDATE_VERSION%_"$SPEC_OS"_"$SPEC_ARCH".tar.gz}
        if spec_valid_version "v$SPEC_CANDIDATE_VERSION"; then
          SPEC_VERSION=v$SPEC_CANDIDATE_VERSION
          SPEC_MATCHES=$((SPEC_MATCHES + 1))
        fi
      done
      [ "$SPEC_MATCHES" -eq 1 ] || spec_fail "Local mode needs exactly one matching version; otherwise use --version"
    fi
  else
    if [ "$SPEC_TRANSPORT" = auto ]; then
      SPEC_TRANSPORT=https
      if command -v gh >/dev/null 2>&1 && gh auth status --active --hostname github.com >/dev/null 2>&1; then SPEC_TRANSPORT=gh; fi
    fi
    if [ "$SPEC_TRANSPORT" = gh ]; then spec_need gh; else spec_need curl; fi
  fi
  for SPEC_COMMAND in tar awk mktemp; do spec_need "$SPEC_COMMAND"; done
  if command -v sha256sum >/dev/null 2>&1; then SPEC_HASH_TOOL=sha256sum
  elif command -v shasum >/dev/null 2>&1; then SPEC_HASH_TOOL=shasum
  else spec_fail "Need sha256sum (Linux) or shasum (macOS)"; fi
  SPEC_TEMP=$(mktemp -d)
  trap spec_cleanup 0
  trap 'exit 130' INT
  trap 'exit 143' HUP TERM
  if [ -z "$SPEC_VERSION" ]; then
    if [ "$SPEC_TRANSPORT" = gh ]; then
      SPEC_VERSION=$(gh release view --repo "$SPEC_REPO" --json tagName --jq .tagName) || spec_fail "No accessible release; check repository access and whether a release exists"
    else
      spec_https "https://api.github.com/repos/$SPEC_REPO/releases/latest" "$SPEC_TEMP/release.json" || spec_fail "No accessible release. Private repositories need authenticated gh or --from-dir"
      SPEC_VERSION=$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\(v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\)".*/\1/p' "$SPEC_TEMP/release.json")
    fi
  fi
  spec_valid_version "$SPEC_VERSION" || spec_fail "Release did not identify one stable version"
  SPEC_ASSET=SPEC_${SPEC_VERSION#v}_${SPEC_OS}_${SPEC_ARCH}.tar.gz
  if [ -n "$SPEC_FROM_DIR" ]; then
    cp -- "$SPEC_FROM_DIR/$SPEC_ASSET" "$SPEC_TEMP/$SPEC_ASSET"
    cp -- "$SPEC_FROM_DIR/SHA256SUMS" "$SPEC_TEMP/SHA256SUMS"
  elif [ "$SPEC_TRANSPORT" = gh ]; then
    gh release download "$SPEC_VERSION" --repo "$SPEC_REPO" --dir "$SPEC_TEMP" --pattern "$SPEC_ASSET" --pattern SHA256SUMS || spec_fail "Release assets could not be downloaded"
  else
    SPEC_RELEASE_URL=https://github.com/$SPEC_REPO/releases/download/$SPEC_VERSION
    spec_https "$SPEC_RELEASE_URL/SHA256SUMS" "$SPEC_TEMP/SHA256SUMS"
    spec_https "$SPEC_RELEASE_URL/$SPEC_ASSET" "$SPEC_TEMP/$SPEC_ASSET"
  fi
  SPEC_EXPECTED=$(awk -v name="$SPEC_ASSET" 'NF==2 && ($2==name || $2=="*"name) { if (length($1)!=64 || $1 ~ /[^0-9a-f]/) exit 1; value=$1; count++ } END { if (count!=1) exit 1; print value }' "$SPEC_TEMP/SHA256SUMS") || spec_fail "Checksum entry missing, duplicated or invalid"
  if [ "$SPEC_HASH_TOOL" = sha256sum ]; then SPEC_ACTUAL=$(sha256sum "$SPEC_TEMP/$SPEC_ASSET" | awk '{print $1}')
  else SPEC_ACTUAL=$(shasum -a 256 "$SPEC_TEMP/$SPEC_ASSET" | awk '{print $1}'); fi
  [ "$SPEC_EXPECTED" = "$SPEC_ACTUAL" ] || spec_fail "SHA-256 mismatch; existing installation was not changed"
  # Stream only fixed member names; never extract archive paths onto the filesystem.
  tar -tzf "$SPEC_TEMP/$SPEC_ASSET" > "$SPEC_TEMP/members"
  [ "$(wc -l < "$SPEC_TEMP/members" | tr -d ' ')" -eq 4 ] || spec_fail "Unexpected archive layout"
  for SPEC_MEMBER in spec LICENSE THIRD_PARTY_NOTICES.md README.md; do
    [ "$(grep -Fxc "$SPEC_MEMBER" "$SPEC_TEMP/members")" -eq 1 ] || spec_fail "Missing or duplicated archive member: $SPEC_MEMBER"
    tar -xzOf "$SPEC_TEMP/$SPEC_ASSET" "$SPEC_MEMBER" > "$SPEC_TEMP/$SPEC_MEMBER"
  done
  chmod 755 "$SPEC_TEMP/spec"
  SPEC_ACTUAL_VERSION=$("$SPEC_TEMP/spec" --version) || spec_fail "Downloaded executable cannot run on this system"
  [ "$SPEC_ACTUAL_VERSION" = "${SPEC_VERSION#v}" ] || spec_fail "Executable and requested version disagree"
  mkdir -p -- "$SPEC_BIN_DIR"
  SPEC_BIN_DIR=$(CDPATH= cd -- "$SPEC_BIN_DIR" && pwd -P)
  mkdir -- "$SPEC_BIN_DIR/.spec-install.lock" || spec_fail "Another install may be running; inspect $SPEC_BIN_DIR/.spec-install.lock before removing a stale lock"
  SPEC_LOCKED=1
  SPEC_STAGE=$(mktemp -d "$SPEC_BIN_DIR/.spec-install.XXXXXX")
  mkdir "$SPEC_STAGE/old" "$SPEC_STAGE/new"
  cp -- "$SPEC_TEMP/spec" "$SPEC_STAGE/new/spec"
  cp -- "$SPEC_TEMP/LICENSE" "$SPEC_STAGE/new/spec.LICENSE"
  cp -- "$SPEC_TEMP/THIRD_PARTY_NOTICES.md" "$SPEC_STAGE/new/spec.THIRD_PARTY_NOTICES.md"
  chmod 755 "$SPEC_STAGE/new/spec"
  for SPEC_NAME in spec.LICENSE spec.THIRD_PARTY_NOTICES.md spec; do
    SPEC_TARGET=$SPEC_BIN_DIR/$SPEC_NAME
    [ ! -L "$SPEC_TARGET" ] || spec_fail "Refusing to replace a symlink: $SPEC_TARGET"
    if [ -e "$SPEC_TARGET" ]; then
      [ -f "$SPEC_TARGET" ] || spec_fail "Target is not a regular file: $SPEC_TARGET"
      cp -p -- "$SPEC_TARGET" "$SPEC_STAGE/old/$SPEC_NAME"
    fi
  done
  for SPEC_NAME in spec.LICENSE spec.THIRD_PARTY_NOTICES.md spec; do
    SPEC_INSTALLED="$SPEC_NAME $SPEC_INSTALLED"
    mv -f -- "$SPEC_STAGE/new/$SPEC_NAME" "$SPEC_BIN_DIR/$SPEC_NAME"
  done
  SPEC_INSTALLED=
  printf 'Installed SPEC %s at %s/spec\n' "${SPEC_VERSION#v}" "$SPEC_BIN_DIR"
  case ":${PATH-}:" in *:"$SPEC_BIN_DIR":*) printf 'Run: spec\n' ;; *) printf 'Run: "%s/spec"\nAdd this directory to PATH if desired: %s\n' "$SPEC_BIN_DIR" "$SPEC_BIN_DIR" ;; esac
}
spec_main "$@"
