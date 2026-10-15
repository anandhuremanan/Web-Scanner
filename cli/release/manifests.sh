#!/bin/sh
# Writes the package-manager manifests for a release of webscanner, from the
# files and checksums.txt the release workflow built:
#
#   webscanner.rb    Homebrew formula (macOS and Linux)
#   webscanner.json  Scoop manifest (Windows)
#
# Usage: manifests.sh <version> <owner/repo> <dist folder>
set -eu

version="$1"
repo="$2"
dist="$3"
base="https://github.com/$repo/releases/download/cli-v$version"

sum() {
	hash="$(grep " \*\{0,1\}webscanner_${version}_$1\$" "$dist/checksums.txt" | cut -d ' ' -f 1)"
	[ -n "$hash" ] || {
		echo "manifests.sh: no checksum for $1" >&2
		exit 1
	}
	echo "$hash"
}

cat >"$dist/webscanner.rb" <<EOF
# Homebrew formula for webscanner, written by cli/release/manifests.sh for
# each release. Install with: brew install ${repo%%/*}/tap/webscanner
class Webscanner < Formula
  desc "List a domain's subdomains and pages from public sources"
  homepage "https://webscanner.imanandhu.in/cli"
  version "$version"
  license "MIT"

  on_macos do
    on_arm do
      url "$base/webscanner_${version}_darwin_arm64.tar.gz"
      sha256 "$(sum darwin_arm64.tar.gz)"
    end
    on_intel do
      url "$base/webscanner_${version}_darwin_amd64.tar.gz"
      sha256 "$(sum darwin_amd64.tar.gz)"
    end
  end

  on_linux do
    on_arm do
      url "$base/webscanner_${version}_linux_arm64.tar.gz"
      sha256 "$(sum linux_arm64.tar.gz)"
    end
    on_intel do
      url "$base/webscanner_${version}_linux_amd64.tar.gz"
      sha256 "$(sum linux_amd64.tar.gz)"
    end
  end

  def install
    bin.install "webscanner"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/webscanner -version 2>&1")
  end
end
EOF

cat >"$dist/webscanner.json" <<EOF
{
    "version": "$version",
    "description": "List a domain's subdomains and pages from public sources",
    "homepage": "https://webscanner.imanandhu.in/cli",
    "license": "MIT",
    "architecture": {
        "64bit": {
            "url": "$base/webscanner_${version}_windows_amd64.zip",
            "hash": "$(sum windows_amd64.zip)"
        },
        "arm64": {
            "url": "$base/webscanner_${version}_windows_arm64.zip",
            "hash": "$(sum windows_arm64.zip)"
        }
    },
    "bin": "webscanner.exe"
}
EOF
echo "wrote $dist/webscanner.rb and $dist/webscanner.json"
