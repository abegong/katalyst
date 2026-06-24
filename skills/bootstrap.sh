#!/usr/bin/env bash
# Shared bootstrap for the katalyst skill family. Ensures the `katalyst` CLI is
# available, by fetching and unpacking the matching archive from the latest
# GitHub Release, falling back to `go install`. One copy of this script is
# bundled into every shipped .skill at package time, so there is a single source
# to maintain (skills/bootstrap.sh).
#
# Idempotent: if `katalyst` is already on PATH, it does nothing. No version pin —
# it tracks whatever the latest Release ships.
set -euo pipefail

REPO="abegong/katalyst"
INSTALL_DIR="${KATALYST_INSTALL_DIR:-$HOME/.local/bin}"

# 1. Reuse an already-installed CLI.
if command -v katalyst >/dev/null 2>&1; then
  echo "katalyst already installed: $(command -v katalyst)"
  exit 0
fi

# 2. Detect OS/arch and map to the GoReleaser archive naming.
os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Linux)  goos="linux" ;;
  Darwin) goos="darwin" ;;
  MINGW*|MSYS*|CYGWIN*) goos="windows" ;;
  *) goos="" ;;
esac
case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  aarch64|arm64) goarch="arm64" ;;
  *) goarch="" ;;
esac

go_install_fallback() {
  if command -v go >/dev/null 2>&1; then
    echo "Falling back to: go install github.com/${REPO}@latest"
    GOBIN="$INSTALL_DIR" go install "github.com/${REPO}@latest"
    return 0
  fi
  return 1
}

fetch_from_release() {
  [ -n "$goos" ] && [ -n "$goarch" ] || return 1
  command -v curl >/dev/null 2>&1 || return 1

  # Resolve the latest tag, then build the versioned archive name. GoReleaser
  # strips the leading "v": tag v0.1.0 -> katalyst_0.1.0_<os>_<arch>.<ext>.
  local tag version ext archive url tmp
  tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
  [ -n "$tag" ] || return 1
  version="${tag#v}"

  if [ "$goos" = "windows" ]; then ext="zip"; else ext="tar.gz"; fi
  archive="katalyst_${version}_${goos}_${goarch}.${ext}"
  url="https://github.com/${REPO}/releases/download/${tag}/${archive}"

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN
  echo "Downloading ${archive} ..."
  curl -fsSL "$url" -o "$tmp/$archive" || return 1

  mkdir -p "$INSTALL_DIR"
  if [ "$ext" = "zip" ]; then
    command -v unzip >/dev/null 2>&1 || return 1
    unzip -o -q "$tmp/$archive" -d "$tmp"
  else
    tar -xzf "$tmp/$archive" -C "$tmp"
  fi

  local bin
  bin="$(find "$tmp" -type f -name 'katalyst*' ! -name '*.tar.gz' ! -name '*.zip' | head -n1)"
  [ -n "$bin" ] || return 1
  install -m 0755 "$bin" "$INSTALL_DIR/katalyst"
  echo "Installed katalyst ${version} to $INSTALL_DIR/katalyst"
}

# 3. Try the Release archive; fall back to go install.
if ! fetch_from_release; then
  echo "Could not fetch a release archive for ${goos:-unknown}/${goarch:-unknown}." >&2
  if ! go_install_fallback; then
    echo "Failed to install katalyst: no release archive and no Go toolchain." >&2
    echo "Install manually from https://github.com/${REPO}/releases" >&2
    exit 1
  fi
fi

# 4. Confirm reachability.
if ! command -v katalyst >/dev/null 2>&1; then
  case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo "Note: add $INSTALL_DIR to your PATH to use 'katalyst' directly." >&2 ;;
  esac
fi
echo "Done."
