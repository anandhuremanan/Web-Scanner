#!/bin/sh
# Installs the latest webscanner command-line tool on Linux or macOS.
#
#   curl -fsSL https://raw.githubusercontent.com/anandhuremanan/website_mapper/main/cli/install.sh | sh
#
# Settings (optional, as environment variables):
#   WEBSCANNER_VERSION   a version such as 1.0.0 (default: the latest)
#   INSTALL_DIR          where to put the program (default: /usr/local/bin if
#                        writable, otherwise ~/.local/bin)
set -eu

repo="anandhuremanan/website_mapper"

fail() {
	echo "webscanner install: $*" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || fail "this needs '$1', which is not installed"
}
need curl
need tar
need uname

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) fail "unsupported system $(uname -s); download a release from https://github.com/$repo/releases" ;;
esac
case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) fail "unsupported processor $(uname -m)" ;;
esac

if [ -n "${WEBSCANNER_VERSION:-}" ]; then
	version="${WEBSCANNER_VERSION#v}"
else
	# The newest release whose tag starts with cli-v.
	tag="$(curl -fsSL "https://api.github.com/repos/$repo/releases?per_page=30" |
		grep -o '"tag_name": *"cli-v[^"]*"' | head -n 1 | sed 's/.*"cli-v//; s/"$//')" ||
		fail "could not look up the latest release"
	[ -n "$tag" ] || fail "no release of the command-line tool has been published yet"
	version="$tag"
fi

name="webscanner_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/cli-v$version"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT INT TERM

echo "Downloading webscanner $version for $os/$arch..."
curl -fsSL -o "$work/$name" "$base/$name" || fail "could not download $base/$name"
curl -fsSL -o "$work/checksums.txt" "$base/checksums.txt" || fail "could not download the checksums"

# Check the download against the published checksum before running anything.
expected="$(grep " $name\$" "$work/checksums.txt" | cut -d ' ' -f 1)"
[ -n "$expected" ] || fail "$name is not in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
	actual="$(sha256sum "$work/$name" | cut -d ' ' -f 1)"
elif command -v shasum >/dev/null 2>&1; then
	actual="$(shasum -a 256 "$work/$name" | cut -d ' ' -f 1)"
else
	fail "this needs 'sha256sum' or 'shasum' to check the download"
fi
[ "$expected" = "$actual" ] || fail "the download does not match its checksum; not installing it"

tar -xzf "$work/$name" -C "$work" webscanner

dir="${INSTALL_DIR:-}"
if [ -z "$dir" ]; then
	if [ -w /usr/local/bin ]; then
		dir=/usr/local/bin
	else
		dir="$HOME/.local/bin"
	fi
fi
mkdir -p "$dir"
mv "$work/webscanner" "$dir/webscanner"
chmod 755 "$dir/webscanner"

echo "Installed webscanner $version to $dir/webscanner"
case ":$PATH:" in
*":$dir:"*) ;;
*) echo "Add $dir to your PATH to run it as 'webscanner', or run $dir/webscanner" ;;
esac
echo "Try: webscanner -mode passive example.com"
