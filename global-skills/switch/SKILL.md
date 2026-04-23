---
name: switch
description: >
  Manage cody-switch feature contexts in Codex. Use for listing features,
  switching, creating, forking, archiving, deleting, saving context, checking
  current feature, worktree operations, and session pinning.
---

# cody-switch skill

Use `cody-switch` as the source of truth for feature context management.

## Core rules

- Prefer the human-readable command output unless you need to make a decision from structured data.
- Use `--json` when routing depends on the result.
- Worktree features should be opened in a fresh Codex session: `cd $(cody-switch open <name>) && codex`.
- After a classic switch, remind the user that `AGENTS.md` and `tasks/` changed on disk and that a fresh Codex session is recommended if they switched mid-session.
- Session restore guidance must use `codex resume <id>` and `codex fork <id>`.
- There is no Codex equivalent of the Claude slash-command install surface. If a bundled prompt is helpful, use `cody-switch prompt <name>`.

## Common commands

```bash
cody-switch list
cody-switch current
cody-switch new <name>
cody-switch blank <name> [--branch|--worktree] [--workflow]
cody-switch fork <source> <new-name> [--without-docs] [--without-tasks] [--branch|--worktree]
cody-switch peek <name>
cody-switch archive <name>
cody-switch unarchive <name>
cody-switch delete <name>
cody-switch merge <source> [into <target>] [--delete]
cody-switch save
cody-switch pin-session
cody-switch sync [name]
cody-switch open <name>
cody-switch doctor [--fix]
```

## Prompt templates

Use:

```bash
cody-switch prompt list
cody-switch prompt code-review
```

## JSON usage

Use `--json` for:
- `list`
- `current`
- `blank`
- `fork`
- `switch`
- `doctor`
- `open`

Example:

```bash
cody-switch list --json
```
