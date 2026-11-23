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

for target in linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64; do
    os="${target%/*}"
    arch="${target#*/}"
    ext=""
    [ "$os" = "windows" ] && ext=".exe"

    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
        go build -trimpath -ldflags "-s -w" -o "$out/hb-money-extractor-$os-$arch$ext" ./cmd/hb-money-extractor

    echo "built $out/hb-money-extractor-$os-$arch$ext"
done
