#!/usr/bin/env sh
# One-line installer for the OpenShop release binaries (API + embedded storefront
# and admin console).
#
#   curl -fsSL https://raw.githubusercontent.com/holihur/openshop/main/scripts/install.sh | sh
#
# Environment overrides:
#   OPENSHOP_VERSION      release tag to install (default: latest)
#   OPENSHOP_BIN_DIR      where to put the binaries (default: /usr/local/bin or ~/.local/bin)
#   OPENSHOP_DATA_DIR     where to put migrations (default: ~/.local/share/openshop)
#   OPENSHOP_REPO         owner/repo (default: holihur/openshop)
set -eu

REPO="${OPENSHOP_REPO:-holihur/openshop}"
VERSION="${OPENSHOP_VERSION:-latest}"
DATA_DIR="${OPENSHOP_DATA_DIR:-$HOME/.local/share/openshop}"

err() { printf '\033[31m✗\033[0m %s\n' "$1" >&2; exit 1; }
info() { printf '\033[36m→\033[0m %s\n' "$1"; }
ok() { printf '\033[32m✓\033[0m %s\n' "$1"; }

need() { command -v "$1" >/dev/null 2>&1; }

download() {
  # download <url> <output>; fatal on failure.
  if need curl; then
    curl -fsSL "$1" -o "$2" || err "download failed: $1"
  elif need wget; then
    wget -qO "$2" "$1" || err "download failed: $1"
  else
    err "curl or wget is required"
  fi
}

try_download() {
  # try_download <url> <output>; non-fatal, returns non-zero on failure.
  if need curl; then
    curl -fsSL "$1" -o "$2" 2>/dev/null
  elif need wget; then
    wget -qO "$2" "$1" 2>/dev/null
  else
    return 1
  fi
}

fetch() {
  # fetch <url> -> stdout
  if need curl; then
    curl -fsSL "$1"
  elif need wget; then
    wget -qO- "$1"
  else
    err "curl or wget is required"
  fi
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo linux ;;
    Darwin) echo darwin ;;
    *) err "unsupported OS: $(uname -s) (download a release manually)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) echo amd64 ;;
    arm64 | aarch64) echo arm64 ;;
    *) err "unsupported architecture: $(uname -m)" ;;
  esac
}

resolve_bin_dir() {
  if [ -n "${OPENSHOP_BIN_DIR:-}" ]; then
    echo "$OPENSHOP_BIN_DIR"
  elif [ -w /usr/local/bin ]; then
    echo /usr/local/bin
  else
    echo "$HOME/.local/bin"
  fi
}

main() {
  os="$(detect_os)"
  arch="$(detect_arch)"

  if [ "$VERSION" = "latest" ]; then
    info "resolving the latest release of $REPO"
    tag="$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
    [ -n "$tag" ] || err "could not determine the latest release; set OPENSHOP_VERSION"
  else
    tag="$VERSION"
    case "$tag" in v*) ;; *) tag="v$tag" ;; esac
  fi
  ver="${tag#v}"

  archive="openshop_${ver}_${os}_${arch}.tar.gz"
  base="https://github.com/$REPO/releases/download/$tag"
  bin_dir="$(resolve_bin_dir)"

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT INT TERM

  info "downloading $archive"
  download "$base/$archive" "$tmp/$archive"

  info "verifying checksum"
  if try_download "$base/checksums.txt" "$tmp/checksums.txt"; then
    expected="$(sed -n "s/^\([0-9a-f]*\)[[:space:]]*$archive$/\1/p" "$tmp/checksums.txt")"
    if [ -n "$expected" ]; then
      if need sha256sum; then
        actual="$(sha256sum "$tmp/$archive" | awk '{print $1}')"
      elif need shasum; then
        actual="$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')"
      else
        actual=""
        info "no sha256 tool found; skipping verification"
      fi
      [ -z "$actual" ] || [ "$actual" = "$expected" ] || err "checksum mismatch for $archive"
    fi
  fi

  info "extracting"
  tar -xzf "$tmp/$archive" -C "$tmp"

  mkdir -p "$bin_dir"
  for bin in openshop openshop-ops openshop-migrate openshop-seed; do
    [ -f "$tmp/$bin" ] || err "expected binary $bin missing from archive"
    install -m 0755 "$tmp/$bin" "$bin_dir/$bin" 2>/dev/null || {
      cp "$tmp/$bin" "$bin_dir/$bin"
      chmod 0755 "$bin_dir/$bin"
    }
  done

  if [ -d "$tmp/migrations" ]; then
    mkdir -p "$DATA_DIR"
    rm -rf "$DATA_DIR/migrations"
    cp -R "$tmp/migrations" "$DATA_DIR/migrations"
  fi

  ok "installed openshop $ver to $bin_dir"
  cat <<EOF

Next steps:
  1. Point the server at PostgreSQL, Redis and NATS, e.g.:
       export POSTGRES_DSN='host=localhost port=5432 user=openshop password=openshop dbname=openshop sslmode=disable TimeZone=UTC'
       export REDIS_ADDR='localhost:6379'
       export NATS_URL='nats://localhost:4222'
       export JWT_SECRET="\$(head -c 32 /dev/urandom | base64)"
  2. Apply migrations:  $bin_dir/openshop-migrate -dir $DATA_DIR/migrations
  3. Seed demo data:    $bin_dir/openshop-seed        (optional)
  4. Run the storefront: $bin_dir/openshop
       storefront  http://localhost:8080/
       API docs    http://localhost:8080/docs
  5. Run the admin console (deploy on an internal network):
       OPS_ADDR=:8081 $bin_dir/openshop-ops
       console     http://localhost:8081/

EOF
  case ":$PATH:" in
    *":$bin_dir:"*) ;;
    *) printf '\033[33m!\033[0m %s is not on your PATH\n' "$bin_dir" ;;
  esac
}

main "$@"
