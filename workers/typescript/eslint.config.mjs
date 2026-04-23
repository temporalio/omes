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
      '**/lib/**',
      '**/*.js',
      '**/*.mjs',
      '**/*.cjs',
      'src/protos/*',
      'protogen.js',
      'omes-temp-*',
    ],
  },
  {
    files: ['src/**/*.ts'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended, prettierConfig],
    languageOptions: {
      parserOptions: { project: ['./tsconfig.json'] },
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
    files: ['src/workflows.ts', 'src/workflows-*.ts', 'src/workflows/*.ts'],
    rules: {
      'no-restricted-imports': ['error', ...restrictedBuiltins],
    },
  },
);
