# CLAUDE.md

## Git

- Use conventional commit messages
- Reasonable units, reasonable timeline. Do not group multiple changes into one commit, and commit as you work.
- When merging a branch, use Git's default merge commit, but edit it to add your commit footer

## GitHub Actions

After adding or changing a `uses:` in a workflow, pin it to a commit SHA (with a version comment) using pinact — actions must be hash-pinned (zizmor's `unpinned-uses` audit enforces this in CI). Scope the pin to only the action you touched so unrelated actions aren't bumped:

- `mise exec -- pinact run --update --include '<action-name>'` — pins the matched action(s) to their latest SHA (e.g. `--include 'download-artifact'`).
- Plain `mise exec -- pinact run` pins a newly-added tag to that exact version's SHA without updating already-pinned actions.

If a scoped run isn't possible, run `mise exec -- pinact run --update` and `git checkout -p` (or revert) the hunks for actions you didn't intend to change. Then run `mise run lint:workflows` to confirm.

## Temporary files

If you need to write temporary files, use `.tmp/` inside the current directory. If you want to write outside this repo, ask for permission.
