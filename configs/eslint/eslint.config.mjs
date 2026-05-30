// Default ESLint flat config.
// Baked into the container image at /etc/pedant/eslint/eslint.config.mjs.
// Projects can override by placing their own eslint.config.{js,mjs,cjs} in the repository.

import js from '@eslint/js'
import unicorn from 'eslint-plugin-unicorn'
import globals from 'globals'

export default [
  {
    ignores: [
      '.cache/**',
      '.git/**',
      'build/**',
      'coverage/**',
      'dist/**',
      'node_modules/**',
      'out/**',
      'output/**',
      'vendor/**',
    ],
  },

  js.configs.recommended,

  {
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: {
        ...globals.node,
        ...globals.es2025,
      },
    },
    rules: {
      eqeqeq: ['error', 'always', { null: 'ignore' }],
      'no-constant-binary-expression': 'error',
      'no-constructor-return': 'error',
      'no-duplicate-imports': 'error',
      'no-eval': 'error',
      'no-extend-native': 'error',
      'no-implicit-coercion': ['error', { allow: ['!!'] }],
      'no-implied-eval': 'error',
      'no-lone-blocks': 'error',
      'no-new-wrappers': 'error',
      'no-param-reassign': 'warn',
      'no-return-assign': 'error',
      'no-self-compare': 'error',
      'no-sequences': 'error',
      'no-template-curly-in-string': 'warn',
      'no-throw-literal': 'error',
      'no-unmodified-loop-condition': 'error',
      'no-unneeded-ternary': 'error',
      'no-unused-expressions': 'error',
      'no-useless-call': 'error',
      'no-useless-computed-key': 'error',
      'no-useless-concat': 'error',
      'no-useless-rename': 'error',
      'no-useless-return': 'error',
      'no-var': 'error',
      'object-shorthand': ['error', 'always'],
      'prefer-arrow-callback': 'error',
      'prefer-const': 'error',
      'prefer-destructuring': ['warn', { object: true, array: false }],
      'prefer-rest-params': 'error',
      'prefer-spread': 'error',
      'prefer-template': 'error',
      radix: 'error',
      'require-await': 'warn',
      'symbol-description': 'error',
      'unicorn/no-for-loop': 'error',
      'unicorn/no-instanceof-array': 'error',
      'unicorn/no-useless-spread': 'error',
      'unicorn/no-useless-undefined': 'error',
      'unicorn/prefer-at': 'error',
      'unicorn/prefer-includes': 'error',
      'unicorn/prefer-node-protocol': 'error',
      'unicorn/prefer-number-properties': 'error',
      'unicorn/prefer-string-slice': 'error',
      'unicorn/throw-new-error': 'error',
    },
    plugins: { unicorn },
  },

  {
    files: ['**/*.cjs'],
    languageOptions: { sourceType: 'commonjs' },
  },
]
