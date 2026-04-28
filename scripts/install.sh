#!/usr/bin/env sh
set -eu

repo="${AVM_REPO:-xz1220/Agent-VM}"
version="${AVM_VERSION:-latest}"
install_dir="${AVM_INSTALL_DIR:-$HOME/.local/bin}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'avm install: required command not found: %s\n' "$1" >&2
    exit 1
  fi
}

download() {
  url="$1"
  dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$dest"
  else
    printf 'avm install: curl or wget is required\n' >&2
    exit 1
  fi
}

sha256_check() {
  checksums="$1"
  artifact="$2"
  artifact_name="$(basename "$artifact")"
  expected="$(awk -v name="$artifact_name" '$2 == name { print $1 }' "$checksums")"
  if [ -z "$expected" ]; then
    printf 'avm install: checksum for %s not found\n' "$artifact_name" >&2
    exit 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$artifact" | awk '{ print $1 }')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$artifact" | awk '{ print $1 }')"
  else
    printf 'avm install: sha256sum or shasum is required\n' >&2
    exit 1
  fi
  if [ "$actual" != "$expected" ]; then
    printf 'avm install: checksum mismatch for %s\n' "$artifact_name" >&2
    exit 1
  fi
}

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    printf 'avm install: unsupported OS: %s\n' "$(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64) arch="x86_64" ;;
  *)
    printf 'avm install: unsupported architecture: %s\n' "$(uname -m)" >&2
    exit 1
    ;;
esac

need_cmd tar
tmp="${TMPDIR:-/tmp}/avm-install.$$"
mkdir -p "$tmp"
trap 'rm -rf "$tmp"' EXIT INT TERM

asset="avm_${os}_${arch}.tar.gz"
if [ "$version" = "latest" ]; then
  base="https://github.com/$repo/releases/latest/download"
else
  base="https://github.com/$repo/releases/download/$version"
fi

download "$base/$asset" "$tmp/$asset"
download "$base/checksums.txt" "$tmp/checksums.txt"
sha256_check "$tmp/checksums.txt" "$tmp/$asset"

tar -xzf "$tmp/$asset" -C "$tmp"
mkdir -p "$install_dir"
install "$tmp/avm" "$install_dir/avm"

printf 'installed avm to %s/avm\n' "$install_dir"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf 'add %s to PATH to run avm from any shell\n' "$install_dir" ;;
esac

if [ "${AVM_SKIP_INIT:-0}" != "1" ]; then
  "$install_dir/avm" init --yes >/dev/null
  printf 'initialized AVM home\n'
fi

printf 'next: avm create backend-coder\n'
