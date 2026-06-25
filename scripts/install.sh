#!/bin/sh
# Install the latest katalyst release binary for this machine.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/abegong/katalyst/main/scripts/install.sh | sh
#
# Set KATALYST_INSTALL_DIR to choose the destination. Defaults to ~/.local/bin.
set -eu

repo="abegong/katalyst"
install_dir="${KATALYST_INSTALL_DIR:-$HOME/.local/bin}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "katalyst install: missing required command: $1" >&2
    exit 1
  fi
}

need curl
need sed
need grep
need mktemp
need find

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Linux) goos="linux" ;;
  Darwin) goos="darwin" ;;
  MINGW*|MSYS*|CYGWIN*) goos="windows" ;;
  *)
    echo "katalyst install: unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  aarch64|arm64) goarch="arm64" ;;
  *)
    echo "katalyst install: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

tag="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" \
  | sed -nE 's/.*"tag_name": *"([^"]+)".*/\1/p' \
  | head -n 1)"

if [ -z "$tag" ]; then
  echo "katalyst install: could not resolve latest release tag" >&2
  exit 1
fi

version="${tag#v}"
if [ "$goos" = "windows" ]; then
  ext="zip"
else
  ext="tar.gz"
fi

archive="katalyst_${version}_${goos}_${goarch}.${ext}"
base_url="https://github.com/${repo}/releases/download/${tag}"
tmp="$(mktemp -d)"

cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT HUP INT TERM

echo "Downloading ${archive}..."
curl -fsSL "${base_url}/${archive}" -o "${tmp}/${archive}"
curl -fsSL "${base_url}/checksums.txt" -o "${tmp}/checksums.txt"

expected="$(grep "  ${archive}\$" "${tmp}/checksums.txt" | sed -nE 's/^([0-9a-fA-F]+)  .*/\1/p')"
if [ -z "$expected" ]; then
  echo "katalyst install: checksum not found for ${archive}" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${tmp}/${archive}" | sed -nE 's/^([0-9a-fA-F]+)  .*/\1/p')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "${tmp}/${archive}" | sed -nE 's/^([0-9a-fA-F]+)  .*/\1/p')"
else
  echo "katalyst install: missing sha256sum or shasum for checksum verification" >&2
  exit 1
fi

if [ "$actual" != "$expected" ]; then
  echo "katalyst install: checksum mismatch for ${archive}" >&2
  exit 1
fi

if [ "$ext" = "zip" ]; then
  need unzip
  unzip -o -q "${tmp}/${archive}" -d "$tmp"
else
  need tar
  tar -xzf "${tmp}/${archive}" -C "$tmp"
fi

bin="$(find "$tmp" -type f -name 'katalyst*' ! -name '*.tar.gz' ! -name '*.zip' | head -n 1)"
if [ -z "$bin" ]; then
  echo "katalyst install: extracted archive did not contain a katalyst binary" >&2
  exit 1
fi

mkdir -p "$install_dir"
install -m 0755 "$bin" "${install_dir}/katalyst"

echo "Installed katalyst ${version} to ${install_dir}/katalyst"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "Add ${install_dir} to PATH to run 'katalyst' directly." >&2 ;;
esac
