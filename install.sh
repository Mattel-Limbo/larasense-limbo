#!/bin/sh
# Install script for larasense-limbo
# Usage: curl -fsSL https://raw.githubusercontent.com/Mattel-Limbo/larasense-limbo/main/install.sh | sh
#
# Options (via environment variables):
#   VERSION=0.5.1          Install a specific version (default: latest)
#   INSTALL_DIR=~/.local/bin   Custom install directory (default: auto-detect)

set -e

REPO="Mattel-Limbo/larasense-limbo"
BINARY_NAME="larasense-limbo"

# ─── Colors ───────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info() {
  printf "${CYAN}  ▸ %s${RESET}\n" "$1"
}

success() {
  printf "${GREEN}  ✓ %s${RESET}\n" "$1"
}

warn() {
  printf "${YELLOW}  ⚠ %s${RESET}\n" "$1"
}

error() {
  printf "${RED}  ✗ %s${RESET}\n" "$1" >&2
}

# ─── Platform Detection ──────────────────────────────────────────────────────

detect_platform() {
  platform="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "${platform}" in
    linux)   platform="linux" ;;
    darwin)  platform="darwin" ;;
    mingw*|msys*|cygwin*) platform="windows" ;;
    *)
      error "Unsupported operating system: ${platform}"
      error "Supported: linux, darwin (macOS), windows (Git Bash/MSYS2)"
      exit 1
      ;;
  esac
  printf '%s' "${platform}"
}

detect_arch() {
  arch="$(uname -m)"
  case "${arch}" in
    x86_64|amd64)    arch="amd64" ;;
    aarch64|arm64)   arch="arm64" ;;
    *)
      error "Unsupported architecture: ${arch}"
      error "Supported: x86_64 (amd64), aarch64 (arm64)"
      exit 1
      ;;
  esac
  printf '%s' "${arch}"
}

# ─── Version Detection ───────────────────────────────────────────────────────

get_latest_version() {
  # Try GitHub API first
  if command -v curl > /dev/null 2>&1; then
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | \
      grep '"tag_name"' | sed -E 's/.*"tag_name": *"v?([^"]+)".*/\1/')
  elif command -v wget > /dev/null 2>&1; then
    version=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | \
      grep '"tag_name"' | sed -E 's/.*"tag_name": *"v?([^"]+)".*/\1/')
  fi

  if [ -z "${version}" ]; then
    error "Could not determine latest version."
    error "Set VERSION=x.y.z manually or check: https://github.com/${REPO}/releases"
    exit 1
  fi

  printf '%s' "${version}"
}

# ─── Install Directory ────────────────────────────────────────────────────────

detect_install_dir() {
  # User override
  if [ -n "${INSTALL_DIR}" ]; then
    printf '%s' "${INSTALL_DIR}"
    return
  fi

  # Prefer ~/.local/bin (no sudo needed, XDG standard)
  if [ -d "${HOME}/.local/bin" ]; then
    printf '%s' "${HOME}/.local/bin"
    return
  fi

  # /usr/local/bin (needs sudo on most systems)
  if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    printf '%s' "/usr/local/bin"
    return
  fi

  # Fallback: create ~/.local/bin
  printf '%s' "${HOME}/.local/bin"
}

# ─── Download ─────────────────────────────────────────────────────────────────

download() {
  url="$1"
  dest="$2"

  if command -v curl > /dev/null 2>&1; then
    curl -fsSL -o "${dest}" "${url}"
  elif command -v wget > /dev/null 2>&1; then
    wget -qO "${dest}" "${url}"
  else
    error "Neither curl nor wget found. Please install one of them."
    exit 1
  fi
}

# ─── Main ─────────────────────────────────────────────────────────────────────

main() {
  printf "\n"
  printf "${BOLD}  Larasense Limbo Installer${RESET}\n"
  printf "  ─────────────────────────\n\n"

  # Detect environment
  platform=$(detect_platform)
  arch=$(detect_arch)
  info "Platform: ${platform}-${arch}"

  # Determine version
  if [ -n "${VERSION}" ]; then
    version="${VERSION}"
    info "Version: v${version} (specified)"
  else
    info "Fetching latest version..."
    version=$(get_latest_version)
    info "Version: v${version} (latest)"
  fi

  # Build download URL (matches GoReleaser naming)
  ext="tar.gz"
  if [ "${platform}" = "windows" ]; then
    ext="zip"
  fi
  filename="${BINARY_NAME}_${version}_${platform}_${arch}.${ext}"
  url="https://github.com/${REPO}/releases/download/v${version}/${filename}"

  info "Downloading: ${filename}"

  # Download to temp
  tmpdir=$(mktemp -d)
  trap 'rm -rf "${tmpdir}"' EXIT
  archive="${tmpdir}/${filename}"

  if ! download "${url}" "${archive}"; then
    error "Download failed: ${url}"
    error ""
    error "Possible causes:"
    error "  - Version v${version} does not exist"
    error "  - No binary for ${platform}-${arch}"
    error ""
    error "Check available releases:"
    error "  https://github.com/${REPO}/releases"
    exit 1
  fi

  success "Downloaded"

  # Extract
  info "Extracting..."
  if [ "${ext}" = "zip" ]; then
    if command -v unzip > /dev/null 2>&1; then
      unzip -qo "${archive}" -d "${tmpdir}/extract"
    else
      error "unzip not found. Please install unzip."
      exit 1
    fi
  else
    mkdir -p "${tmpdir}/extract"
    tar -xzf "${archive}" -C "${tmpdir}/extract"
  fi

  # Find binary
  binary_name="${BINARY_NAME}"
  if [ "${platform}" = "windows" ]; then
    binary_name="${BINARY_NAME}.exe"
  fi

  extracted_binary=$(find "${tmpdir}/extract" -name "${binary_name}" -type f 2>/dev/null | head -1)
  if [ -z "${extracted_binary}" ]; then
    error "Binary '${binary_name}' not found in archive"
    exit 1
  fi

  success "Extracted"

  # Install
  install_dir=$(detect_install_dir)
  info "Installing to: ${install_dir}"

  # Create directory if needed
  if [ ! -d "${install_dir}" ]; then
    mkdir -p "${install_dir}"
    warn "Created ${install_dir}"
  fi

  dest="${install_dir}/${binary_name}"

  # Check if we need sudo
  if [ -d "${install_dir}" ] && [ ! -w "${install_dir}" ]; then
    warn "Need elevated permissions for ${install_dir}"
    sudo cp "${extracted_binary}" "${dest}"
    sudo chmod +x "${dest}"
  else
    cp "${extracted_binary}" "${dest}"
    chmod +x "${dest}"
  fi

  success "Installed: ${dest}"

  # Verify
  if "${dest}" version > /dev/null 2>&1; then
    version_output=$("${dest}" version 2>&1 | head -1)
    success "Verified: ${version_output}"
  else
    warn "Binary installed but could not verify (may need to restart shell)"
  fi

  # Check PATH
  case ":${PATH}:" in
    *":${install_dir}:"*)
      # Already in PATH
      ;;
    *)
      printf "\n"
      warn "${install_dir} is not in your PATH"
      printf "\n"
      printf "  Add it to your shell profile:\n"
      printf "\n"
      printf "    ${CYAN}# bash (~/.bashrc)${RESET}\n"
      printf "    ${BOLD}export PATH=\"\$HOME/.local/bin:\$PATH\"${RESET}\n"
      printf "\n"
      printf "    ${CYAN}# zsh (~/.zshrc)${RESET}\n"
      printf "    ${BOLD}export PATH=\"\$HOME/.local/bin:\$PATH\"${RESET}\n"
      printf "\n"
      printf "    ${CYAN}# fish (~/.config/fish/config.fish)${RESET}\n"
      printf "    ${BOLD}set -gx PATH \$HOME/.local/bin \$PATH${RESET}\n"
      printf "\n"
      printf "  Then restart your terminal or run: ${BOLD}source ~/.bashrc${RESET}\n"
      ;;
  esac

  printf "\n"
  printf "  ${GREEN}${BOLD}larasense-limbo v${version} installed successfully!${RESET}\n"
  printf "\n"
  printf "  Get started:\n"
  printf "    ${BOLD}cd /path/to/laravel-project${RESET}\n"
  printf "    ${BOLD}larasense-limbo init${RESET}\n"
  printf "    ${BOLD}larasense-limbo scan${RESET}\n"
  printf "\n"
  printf "  Docs: ${CYAN}https://github.com/${REPO}${RESET}\n"
  printf "\n"
}

main
