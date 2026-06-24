// Intentionally malformed JavaScript file for pedant e2e testing.
//
// Expected findings:
//   eslint no-var                                -- 'var' declarations (x and y)
//   eslint eqeqeq                                -- '==' instead of '==='
//   eslint no-console                            -- console.log calls
//   unicorn/no-negated-condition                 -- !flag with else branch
//   unicorn/prefer-logical-operator-over-ternary -- ternary expressible as ??

var x = 1
var y = 2

if (x == y) {
  console.log('equal')
}

const flag = true
if (!flag) {
  console.log('off')
} else {
  console.log('on')
}

const val = null
const out = val !== null ? val : 0
console.log(out)
