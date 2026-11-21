// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

import pluginVue from 'eslint-plugin-vue';
import vueTsEslintConfig from '@vue/eslint-config-typescript';

export default [
    ...pluginVue.configs['flat/essential'],
    ...vueTsEslintConfig(),
    {
        languageOptions: {
            parserOptions: {
                projectService: true,
                tsconfigRootDir: import.meta.dirname,
            }
        },
    },
    {
        ignores: [
            'dist/**',
            'src/models/generated/**',
            '**/*.{js,jsx,cjs,mjs}'
        ]
    },
    {
        files: [
            '**/*.{vue,ts,tsx,mts,js,jsx,cjs,mjs}'
        ],
        rules: {
            'vue/valid-v-slot': ['error', {
                allowModifiers: true
            }]
        }
    },
];
