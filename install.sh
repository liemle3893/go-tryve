#!/bin/sh
# Tryve installer — downloads the latest release from GitHub.
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/liemle3893/e2e-runner/main/install.sh | sh
#   curl -fsSL ... | sh -s -- --dir /custom/path
#   curl -fsSL ... | sh -s -- --version v1.2.3

set -eu

REPO="liemle3893/e2e-runner"
BINARY="tryve"
INSTALL_DIR="/usr/local/bin"
VERSION=""

# --- Argument parsing ---
while [ $# -gt 0 ]; do
  case "$1" in
    --dir)     INSTALL_DIR="$2"; shift 2 ;;
    --version) VERSION="$2";     shift 2 ;;
    -h|--help)
      echo "Usage: install.sh [--dir DIR] [--version VERSION]"
      echo "  --dir      Install directory (default: /usr/local/bin)"
      echo "  --version  Specific version tag (default: latest)"
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# --- Detect OS and architecture ---
detect_platform() {
  OS="$(uname -s)"
  ARCH="$(uname -m)"

  case "$OS" in
    Linux*)  OS="linux" ;;
    Darwin*) OS="darwin" ;;
    *)       echo "Error: unsupported OS '$OS'" >&2; exit 1 ;;
  esac

  case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    *)              echo "Error: unsupported architecture '$ARCH'" >&2; exit 1 ;;
  esac
}

# --- Resolve version ---
resolve_version() {
  if [ -z "$VERSION" ]; then
    echo "Fetching latest release..."
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name"' \
      | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"

    if [ -z "$VERSION" ]; then
      echo "Error: could not determine latest version" >&2
      exit 1
    fi
  fi
}

# --- Download and install ---
install() {
  # Strip leading 'v' for the archive filename (goreleaser uses raw version)
  RAW_VERSION="${VERSION#v}"
  ARCHIVE="${BINARY}_${RAW_VERSION}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

  TMPDIR="$(mktemp -d)"
  trap 'rm -rf "$TMPDIR"' EXIT

  echo "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}..."
  curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "$URL"

  echo "Extracting..."
  tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

  # Ensure install directory exists
  if [ ! -d "$INSTALL_DIR" ]; then
    mkdir -p "$INSTALL_DIR" 2>/dev/null || {
      echo "Error: cannot create ${INSTALL_DIR}. Try with sudo or use --dir." >&2
      exit 1
    }
  fi

  # Install binary
  if [ -w "$INSTALL_DIR" ]; then
    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    echo "Need elevated permissions to install to ${INSTALL_DIR}"
    sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  fi

  chmod +x "${INSTALL_DIR}/${BINARY}"

  echo ""
  echo "Successfully installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"
  echo "Run '${BINARY} --help' to get started."
}

detect_platform
resolve_version
install
