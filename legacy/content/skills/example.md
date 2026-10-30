---
name: example
description: Sample skill of the smidja bundle template. Use when demonstrating or verifying how baked-in skills resolve from the embedded content.
---

# Example skill

This skill ships inside the bundle binary: it is embedded at build time and served to the harness from the bundle filesystem.

## When to use

- Proving that the embedded content pipeline works.
- Testing content resolution from the harness CLI.

## How it resolves

The harness reads skills from the bundle filesystem rooted at `content/`. This file lives at `skills/example.md` inside that root, so the harness catalogs it as the skill named `example` from this bundle's package.
