#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
# SPDX-License-Identifier: GPL-3.0-or-later
"""Check the shared Money engine against a reviewed manifest or a peer checkout."""
import argparse
import hashlib
import json
from pathlib import Path
import sys

ENGINE = Path("pkg/converters/msmoney")
MANIFEST = Path("scripts/money-engine-manifest.json")
ADAPTERS = {
    "microsoft_money.go",
    "microsoft_money_transaction_data_file_importer.go",
    "microsoft_money_transaction_data_file_importer_test.go",
    "microsoft_money_transaction_data_table.go",
    "microsoft_money_transaction_data_table_test.go",
}
MODULES = (b"github.com/James-HoneyBadger/hb-sunrise/", b"github.com/James-HoneyBadger/honey-ledger/")

def snapshot(root):
    engine = root / ENGINE
    if not engine.is_dir():
        raise ValueError(f"Engine directory missing: {engine}")
    result = {}
    for path in sorted(engine.rglob("*.go")):
        name = path.relative_to(engine).as_posix()
        if name in ADAPTERS:
            continue
        data = path.read_bytes().replace(b"\r\n", b"\n")
        for module in MODULES:
            data = data.replace(b'"' + module, b'"shared-money-engine/')
        result[name] = hashlib.sha256(data).hexdigest()
    if not result:
        raise ValueError("No shared engine sources found")
    return result

def compare(actual, expected, label):
    differences = sorted(k for k in actual.keys() | expected.keys()
                         if actual.get(k) != expected.get(k))
    if differences:
        for key in differences:
            print(f"{label}: {key}: {'missing' if key not in actual else 'added' if key not in expected else 'changed'}", file=sys.stderr)
        return False
    print(f"{label}: {len(actual)} shared files match")
    return True

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parent.parent)
    parser.add_argument("--peer", type=Path)
    parser.add_argument("--update", action="store_true", help="explicitly regenerate reviewed baseline")
    args = parser.parse_args()
    if args.update and args.peer:
        parser.error("--update and --peer are mutually exclusive")
    try:
        current = snapshot(args.root)
        manifest = args.root / MANIFEST
        if args.update:
            manifest.write_text(json.dumps(current, indent=2, sort_keys=True) + "\n")
            print(f"Recorded {len(current)} shared files")
            return 0
        passed = compare(current, json.loads(manifest.read_text()), "manifest")
        if args.peer:
            passed = compare(current, snapshot(args.peer), "peer") and passed
        return 0 if passed else 1
    except (OSError, ValueError) as exc:
        print(f"Parity check failed: {exc}", file=sys.stderr)
        return 1

if __name__ == "__main__":
    sys.exit(main())
