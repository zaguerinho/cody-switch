---
name: init-project
description: >
  Bootstrap a repository for Codex-friendly work. Use when the user wants to
  initialize or refresh project instructions, scaffold AGENTS files, or set up
  feature context storage for cody-switch.
---

# init-project

Use this skill when the user wants a Codex-ready project baseline.

## Goals

- Create or improve the root `AGENTS.md`
- Add scoped `AGENTS.md` files only where they provide real value
- Initialize cody-switch storage with `cody-switch init` when appropriate
- Avoid Claude-only conventions such as `.claude/commands/`

## Workflow

1. Inspect the repo layout and stack.
2. Read any existing `AGENTS.md` files first.
3. If the repo is not initialized for cody-switch, run:

```bash
cody-switch init
```

4. Improve the root `AGENTS.md` with:
   - stack summary
   - testing commands
   - key directories
   - project-specific conventions
5. Add nested `AGENTS.md` files only for directories with genuinely different conventions.
6. Summarize what was created or changed.

## Rules

- Prefer real `AGENTS.md` files in scoped directories over hidden metadata folders.
- Do not invent project-local slash-command systems.
- Keep instruction files short and practical.
