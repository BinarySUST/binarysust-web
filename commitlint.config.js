// Conventional Commits 1.0.0 (https://www.conventionalcommits.org): `type(scope): description`.
// lefthook runs it on every commit message; CI runs it on pull request titles (squash-merge subjects).
export default { extends: ["@commitlint/config-conventional"] };
