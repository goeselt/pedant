// Intentionally malformed JavaScript file for pedant e2e testing.
//
// Expected findings:
//   eslint no-var    -- 'var' declarations (x and y)
//   eslint eqeqeq   -- '==' instead of '==='

var x = 1
var y = 2

if (x == y) {
  console.log('equal')
}
