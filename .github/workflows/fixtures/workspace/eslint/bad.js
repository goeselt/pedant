// Custom ESLint scenario: no-console rule added by custom config.
// This file is clean under the bundled config but fails the custom one.
//
// Expected findings with custom config:
//   eslint no-console  -- console.log is forbidden
// No findings with bundled config.

const name = 'World'
console.log(`Hello, ${name}`)
