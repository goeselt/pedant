// Custom prettier scenario: stricter printWidth (40 vs the bundled 120).
// The line below is 64 chars -- bundled keeps it as-is; custom must wrap it.
//
// Expected finding with custom config:
//   prettier  -- "needs formatting"
// No findings with bundled config.

const x = 'long string here that fits 120 but not 40 columns easily'
console.log(x)
