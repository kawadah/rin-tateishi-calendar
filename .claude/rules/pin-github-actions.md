---
paths:
  - ".github/workflows/*.{yml,yaml}"
  - "**/.github/workflows/*.{yml,yaml}"
---

# Pinning GitHub Actions

After adding or changing any `uses:` ref, run:

```sh
mise run pin:actions:update
```

Do this before committing, and do not wait for Dependabot.

## Why

A `uses:` ref written from memory is usually stale. Pinning alone does not fix
that — it preserves it:

| Command | `actions/checkout@v4` becomes |
| --- | --- |
| `mise run pin:actions` | `11d5960…` **# v4.4.0** — still the old major |
| `mise run pin:actions:update` | `3d3c42e…` **# v7.0.1** — current |

The first result is the dangerous one. A full SHA with a version comment reads
as deliberate and security-conscious, so it passes review unquestioned while
being years out of date.

Dependabot would eventually propose the major bump, but as a separate PR days
later, by which point the stale ref has already been reviewed as correct.

## Review the diff

`pin:actions:update` rewrites **every** ref in the file, not only the one just
added. If a ref is deliberately held at an older version, restore it after
running.

To see what would change without writing to the file:

```sh
pinact run --update --diff
```
