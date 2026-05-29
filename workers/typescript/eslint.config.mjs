import { builtinModules } from 'node:module';
import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import prettierConfig from 'eslint-config-prettier';

const ALLOWED_NODE_BUILTINS = new Set(['assert']);
const restrictedBuiltins = builtinModules
  .filter((m) => !ALLOWED_NODE_BUILTINS.has(m))
  .flatMap((m) => [m, `node:${m}`]);

export default tseslint.config(
  {
    ignores: [
      '**/node_modules/**',
      '**/dist/**',
      '**/dist-test/**',
      '**/lib/**',
      '**/*.js',
      '**/*.mjs',
      '**/*.cjs',
      'harness/api/**',
      'workerlib/kitchensink/protos/*',
      'protogen.js',
      'omes-temp-*',
    ],
  },
  {
    files: ['apps/**/*.ts', 'harness/**/*.ts', 'workerlib/**/*.ts'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended, prettierConfig],
    languageOptions: {
      parserOptions: { project: ['./tsconfig.json', './tsconfig.test.json'] },
    },
    rules: {
      '@typescript-eslint/no-deprecated': 'warn',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/no-unused-vars': [
        'warn',
        {
          argsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
        },
      ],
      'object-shorthand': ['error', 'always'],
    },
  },
  {
    files: ['workerlib/kitchensink/workflows/**/*.ts'],
    rules: {
      'no-restricted-imports': ['error', ...restrictedBuiltins],
    },
  },
);
