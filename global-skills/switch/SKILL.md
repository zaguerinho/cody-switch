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
- After a classic switch, default to recommending a fresh `codex` session because `AGENTS.md` and `tasks/` changed on disk. Continue in the current session only when the user clearly wants to bridge prior context into the new feature.
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

## Dirty State Handling

For classic features, `cody-switch` buckets tracked dirty paths before switching:

- `active_feature_only` on a non-protected branch: runs `save`, stages `.codex/features/<active>/`, commits `chore(<active>): sync feature AGENTS.md`, then completes the switch.
- `active_feature_only` on a protected branch (`main`, `master`, `roles-deploy`, or `CODY_SWITCH_PROTECTED_BRANCHES`): aborts and asks for a feature branch commit.
- `cross_feature` or `mixed`: aborts with the leak recovery playbook because `.codex/features/<X>/...` is dirty while `<X>` is not active.
- `code_only`: aborts with the standard commit-or-stash message.

Use `cody-switch <name> --no-auto-commit` when the user explicitly wants to inspect active-feature context edits before committing them.

The installed per-repo pre-commit hook blocks staged `.codex/features/<X>/...` paths when `<X>` is not the active feature. Override only when intentional:

```bash
git commit --no-verify
CODY_SWITCH_SKIP_FEATURE_GUARD=1 git commit -m "..."
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
