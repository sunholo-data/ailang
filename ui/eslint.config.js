import js from '@eslint/js';
import globals from 'globals';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';
import reactPlugin from 'eslint-plugin-react';
import reactHooksPlugin from 'eslint-plugin-react-hooks';

export default [
  js.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    ignores: ['dist/**', 'node_modules/**'],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        ecmaFeatures: {
          jsx: true,
        },
      },
      globals: {
        ...globals.browser,
        ...globals.es2020,
        RequestInit: 'readonly',  // TypeScript DOM type
      },
    },
    plugins: {
      '@typescript-eslint': tseslint,
      'react': reactPlugin,
      'react-hooks': reactHooksPlugin,
    },
    settings: {
      react: {
        version: 'detect',
      },
    },
    rules: {
      // CRITICAL: Catch use-before-define errors that cause runtime crashes
      // This prevents the "can't access lexical declaration before initialization" errors
      '@typescript-eslint/no-use-before-define': ['error', {
        functions: false,  // Allow function hoisting
        classes: true,
        variables: true,   // CRITICAL: Catch variable use before definition
        allowNamedExports: false,
      }],

      // React hooks rules - ensure proper dependency arrays
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',

      // Relaxed rules for our codebase
      'react/react-in-jsx-scope': 'off',  // Not needed in React 17+
      'react/prop-types': 'off',  // We use TypeScript
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],

      // Allow any for now since we're retrofitting
      'no-unused-vars': 'off',  // TypeScript handles this
    },
  },
];
