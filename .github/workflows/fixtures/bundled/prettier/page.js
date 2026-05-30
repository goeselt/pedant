// Intentionally malformed JavaScript file for pedant e2e testing.
//
// Expected finding:
//   prettier  -- "needs formatting" (double space before return value)

function foo() {
  return  1
}
console.log(foo())
