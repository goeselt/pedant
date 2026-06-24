// Intentionally malformed TypeScript file for pedant e2e testing.
//
// The explicit type annotations require the TypeScript parser; with eslint's
// default espree parser this file would fail to parse instead of yielding lint
// findings. It therefore exercises the bundled flat-config TypeScript support.
//
// Expected findings under the bundled config:
//   eslint no-var  -- 'var' declaration
//   eslint eqeqeq  -- '==' instead of '==='

var count: number = 1
const limit: number = 2

if (count == limit) {
  // count reached limit
}
