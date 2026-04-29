import js from '@eslint/js';
import svelte from 'eslint-plugin-svelte';
import svelteParser from 'svelte-eslint-parser';

export default [
  {
    ignores: ['dist/', 'node_modules/', 'wailsjs/', 'build/'],
  },

  js.configs.recommended,

  {
    files: ['**/*.svelte'],
    plugins: { svelte },
    languageOptions: {
      parser: svelteParser,
    },
    rules: {
      ...svelte.configs.recommended.rules,
    },
  },
];
