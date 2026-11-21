#!/usr/bin/env node
// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

// Fails when a source file is missing the project's two-line SPDX header.

import { readdirSync, readFileSync } from 'node:fs';
import { dirname, extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');

// Generated output and dependencies are not ours to annotate.
const SKIP_DIRS = new Set(['.git', 'node_modules', 'dist', 'build', 'release', 'vendor', 'package']);

const COMMENT_STYLES = {
    '.go': ['// ', ''],
    '.ts': ['// ', ''],
    '.js': ['// ', ''],
    '.mjs': ['// ', ''],
    '.cjs': ['// ', ''],
    '.sh': ['# ', ''],
    '.ps1': ['# ', ''],
    '.bat': ['REM ', ''],
    '.vue': ['<!-- ', ' -->'],
};

const COPYRIGHT = 'SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe';
const IDENTIFIER = 'SPDX-License-Identifier: GPL-3.0-or-later';

function* walk(dir) {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
        if (entry.isDirectory()) {
            if (!SKIP_DIRS.has(entry.name)) {
                yield* walk(join(dir, entry.name));
            }
        } else if (entry.isFile()) {
            yield join(dir, entry.name);
        }
    }
}

const failures = [];
let checked = 0;

for (const path of walk(root)) {
    const style = COMMENT_STYLES[extname(path)];

    if (style === undefined) {
        continue;
    }

    const [open, close] = style;
    const lines = readFileSync(path, 'utf8').split('\n');
    // A shebang has to stay on line one, so the header follows it.
    const start = lines[0].startsWith('#!') ? 1 : 0;

    checked += 1;

    if (lines[start] !== `${open}${COPYRIGHT}${close}` || lines[start + 1] !== `${open}${IDENTIFIER}${close}`) {
        failures.push(relative(root, path));
    }
}

if (failures.length > 0) {
    console.error('License header check failed; add the two-line SPDX header to:');

    for (const file of failures) {
        console.error(`  - ${file}`);
    }

    process.exit(1);
}

console.log(`License header check passed: ${checked} source files.`);
