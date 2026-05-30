// Custom ESLint config: adds no-console (not present in the bundled default).
// Deliberately self-contained -- avoids `import` of npm packages because Node
// ESM does not honour NODE_PATH and the workspace has no node_modules.
export default [
  {
    rules: {
      'no-console': 'error',
    },
  },
]
