// Conventional Commits: <type>(<scope>): <subject>
// Example: feat(sendMessage): add SendMessage with send options
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      ['feat', 'fix', 'docs', 'style', 'refactor', 'perf', 'test', 'build', 'ci', 'chore', 'revert'],
    ],
    // Scopes mirror the feature branch names (feat/sendMessage -> sendMessage),
    // so camelCase is allowed.
    'scope-case': [2, 'always', ['camel-case', 'lower-case', 'kebab-case']],
    'header-max-length': [2, 'always', 100],
  },
};
