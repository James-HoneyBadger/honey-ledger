#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Builds HB Money Extractor as a standalone program for Linux, Windows and macOS.
# Usage: scripts/build-money-extractor.sh [output-dir]   (default: dist-extractor)

set -euo pipefail

cd "$(dirname "$0")/.."

out="${1:-dist-extractor}"
mkdir -p "$out"

# The extractor reports the same version as the release it ships with.
version="$(sed -n 's/^  "version": "\(.*\)",$/\1/p' package.json | head -1)"

if [ -z "$version" ]; then
    echo "cannot read the version from package.json" >&2
    exit 1
fi

for target in linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64; do
    os="${target%/*}"
    arch="${target#*/}"
    ext=""
    [ "$os" = "windows" ] && ext=".exe"

    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
        go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$out/hb-money-extractor-$os-$arch$ext" ./cmd/hb-money-extractor

    echo "built $out/hb-money-extractor-$os-$arch$ext"
done
