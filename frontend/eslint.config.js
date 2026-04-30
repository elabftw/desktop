import js from '@eslint/js';
import globals from 'globals';
import svelte from 'eslint-plugin-svelte';
import svelteParser from 'svelte-eslint-parser';

export default [
  {
    ignores: ['dist/', 'node_modules/', 'wailsjs/', 'build/'],
  },

  js.configs.recommended,

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
    plugins: { svelte },
    languageOptions: {
      parser: svelteParser,
    },
    rules: {
      ...svelte.configs.recommended.rules,
        "no-unused-vars": "warn",
        "no-undef": "warn", // some functions are used in other files but appear to eslint as never used.
    },
  },
];
