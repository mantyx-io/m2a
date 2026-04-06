#!/usr/bin/env bash
# Install m2a from GitHub releases.
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/mantyx-io/m2a/main/scripts/install.sh | bash
# Pin a release:
#   VERSION=v1.2.3 curl -fsSL ... | bash
# Install to a custom prefix (default: ~/.local/bin):
#   PREFIX=/usr/local curl -fsSL ... | bash

set -euo pipefail

REPO="${REPO:-mantyx-io/m2a}"
PREFIX="${PREFIX:-$HOME/.local/bin}"
VERSION="${VERSION:-}"

# Must be global so the EXIT trap can see it after main() returns (locals are gone then).
INSTALL_TMPDIR=""

cleanup_install_tmp() {
	[[ -z "${INSTALL_TMPDIR:-}" ]] && return 0
	rm -rf -- "$INSTALL_TMPDIR"
}

die() {
	echo "install.sh: $*" >&2
	exit 1
}

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

map_uname() {
	local os arch
	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	arch="$(uname -m)"
	case "$arch" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) die "unsupported CPU architecture: $arch (need amd64 or arm64)" ;;
	esac
	case "$os" in
	linux) ;;
	darwin) ;;
	*) die "unsupported OS: $os (need linux or darwin)" ;;
	esac
	printf '%s %s' "$os" "$arch"
}

read_tag() {
	if [[ -n "$VERSION" ]]; then
		[[ "$VERSION" == v* ]] || VERSION="v${VERSION#v}"
		printf '%s' "$VERSION"
		return
	fi
	need_cmd curl
	local json tag
	json="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")" || die "could not fetch latest release from GitHub"
	if command -v jq >/dev/null 2>&1; then
		tag="$(printf '%s' "$json" | jq -r .tag_name)"
	elif command -v python3 >/dev/null 2>&1; then
		tag="$(printf '%s' "$json" | python3 -c 'import sys,json; print(json.load(sys.stdin)["tag_name"])')"
	else
		die "need jq or python3 to parse the GitHub API response (or set VERSION=v1.2.3 explicitly)"
	fi
	[[ -n "$tag" && "$tag" != null ]] || die "could not parse latest release tag"
	printf '%s' "$tag"
}

main() {
	need_cmd curl
	need_cmd tar
	# Avoid `read` here: when this script is piped from curl, stdin is the script;
	# use word-splitting from map_uname instead (see README for bash -c alternative).
	local os_arch goos goarch
	os_arch="$(map_uname)"
	goos="${os_arch%% *}"
	goarch="${os_arch##* }"
	local tag
	tag="$(read_tag)"
	local base="m2a_${tag}_${goos}_${goarch}"
	local url="https://github.com/${REPO}/releases/download/${tag}/${base}.tar.gz"
	INSTALL_TMPDIR="$(mktemp -d)"
	trap cleanup_install_tmp EXIT

	echo "Downloading ${url}" >&2
	curl -fsSL "$url" -o "${INSTALL_TMPDIR}/bundle.tar.gz" || die "download failed (check tag exists and asset name matches install script)"

	(
		cd "$INSTALL_TMPDIR"
		tar -xzf bundle.tar.gz
		[[ -f m2a ]] || die "archive did not contain binary m2a"
		chmod +x m2a
		mkdir -p "$PREFIX"
		mv m2a "${PREFIX}/m2a"
	)

	echo "Installed m2a ${tag} -> ${PREFIX}/m2a" >&2
	if [[ ":$PATH:" != *":${PREFIX}:"* ]]; then
		echo "Add to PATH if needed: export PATH=\"${PREFIX}:\$PATH\"" >&2
	fi
}

main "$@"
