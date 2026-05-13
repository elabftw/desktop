import js from '@eslint/js';
import globals from 'globals';
import svelte from 'eslint-plugin-svelte';
import svelteParser from 'svelte-eslint-parser';
import ts from 'typescript-eslint';

export default [
  {
    ignores: ['dist/', 'node_modules/', 'wailsjs/', 'build/'],
  },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs['flat/recommended'],
    // prevent flagging legimitate runtime symbols like Go's 'window.'
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
  },

  {
    files: ['**/*.svelte'],
    languageOptions: {
      parser: svelteParser,
      parserOptions: {
        parser: ts.parser,
      },
      // rules: {
      //   ...svelte.configs.recommended.rules,
      //   'no-unused-vars': 'warn',
      // },
    },
  },
];
