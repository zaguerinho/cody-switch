# cody-switch

`cody-switch` is a Codex-oriented port of `claude-switch`.

It preserves the useful core:
- feature-scoped `AGENTS.md`
- per-feature `tasks/` and `docs/`
- saved session checkpoints
- worktree-backed parallel features
- installable global skills under `~/.codex/skills`

It also explicitly avoids pretending Codex has Claude-only surfaces.

## Mapping

| Claude repo | Codex port |
|---|---|
| `CLAUDE.md` | `AGENTS.md` |
| `.claude/features/...` | `.codex/features/...` |
| `~/.claude/CLAUDE.md` | `~/AGENTS.md` |
| `claude --resume` | `codex resume` |
| `claude --resume ... --fork-session` | `codex fork ...` |
| Claude slash commands | Repo-local prompt templates via `cody-switch prompt ...` |
| `SessionStart` hook | No direct Codex equivalent; use `cody-switch current` |

## What Mirrors Cleanly

- Feature switching, save/restore, archive, fork, merge, delete
- `tasks/todo.md` and `tasks/lessons.md` scaffolding
- Feature docs under `docs/{feature}/`
- Worktree-backed features under `.codex/worktrees/{feature}/`
- Global/core skill install to `~/.codex/skills/`
- Session checkpoint and latest-session tracking using Codex session IDs
- JSON output for scripting

## What Does Not Mirror 1:1

- Codex does not expose the Claude `SessionStart` hook path used by `claude-switch`.
- Codex does not have the same `~/.claude/commands/` slash-command install model.
- Some inherited extras, especially `video-tutorial`, are still legacy and Claude-coupled.

## Alternatives Used Here

- `cody-switch current` is the manual context/status surface.
- `cody-switch prompt list` and `cody-switch prompt <name>` expose the bundled prompt templates.
- Stored session IDs are presented as `codex resume <id>` or `codex fork <id>`.
- Global user instructions are seeded into `~/AGENTS.md`.

## Install

```bash
git clone https://github.com/zaguerinho/cody-switch.git ~/scripts/cody-switch
~/scripts/cody-switch/cody-switch install
cody-switch help
```

`install`:
1. creates `/usr/local/bin/cody-switch`, or falls back to `~/.local/bin/cody-switch` when sudo is unavailable
2. seeds `~/AGENTS.md` from `global-agents.md` if needed
3. installs zsh completion for `cody-switch`
4. installs core skills to `~/.codex/skills/`
5. installs companion binaries to `~/.codex/bin/` when a skill declares one
6. optionally installs extra skills with `--extras` or `--skill`

If the installer uses the `~/.local/bin` fallback and that directory is not on `PATH`, it prints the shell line to add.

If `~/AGENTS.md` already exists and differs, the new template is written to `~/AGENTS.md.template-pending` for manual merge.

Companion binaries are downloaded from GitHub Releases when available. If no matching release exists and a local source tree is present, the installer falls back to `make build` and copies the result to `~/.codex/bin/`.

## Quick Start

```bash
cd my-project
cody-switch init

cody-switch new auth-system
cody-switch blank payment-flow
cody-switch fork auth-system auth-v2

cody-switch list
cody-switch auth-system
cody-switch auth-system --with payment-flow

cody-switch current
cody-switch save
```

For parallel work, prefer worktree-backed features:

```bash
cody-switch blank payment-flow --worktree
cd "$(cody-switch open payment-flow)" && codex
```

## Core Commands

```bash
cody-switch install
cody-switch install --force
cody-switch install --extras
cody-switch install --skill video-tutorial
cody-switch uninstall

cody-switch init [--template <stack>] [--force]

cody-switch list [--all]
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
cody-switch promote-lesson [--project|--user]
cody-switch promote-audit

cody-switch sync [name]
cody-switch open <name>
cody-switch doctor [--fix]

cody-switch prompt list
cody-switch prompt code-review
```

## Storage Layout

```text
project-root/
├── AGENTS.md
├── tasks/
├── docs/
├── .codex-current-feature
├── .codex-last-seen-feature
├── .codex-with-refs
└── .codex/
    ├── features/
    │   ├── {feature}/
    │   │   ├── AGENTS.md
    │   │   ├── tasks/
    │   │   ├── session
    │   │   ├── session-latest
    │   │   ├── session-summary
    │   │   └── worktree
    │   ├── archived/{feature}/
    │   └── lessons-global.md
    └── worktrees/{feature}/
```

User-level data:
- `~/AGENTS.md`
- `~/.codex/skills/`
- `~/.codex/lessons-global.md`
- `~/.codex/bin/`

## Session Tracking

Each feature can store:
- `session`: the checkpoint session
- `session-latest`: the latest known Codex session
- `session-summary`: optional one-line description of the checkpoint

When a feature has saved sessions, `cody-switch` prints:
- `codex fork <checkpoint-id>` for a clean continuation
- `codex resume <latest-id>` for direct continuation

## Prompt Templates

The old `global-commands/` directory is kept as a prompt-template library.

Use:

```bash
cody-switch prompt list
cody-switch prompt code-review
cody-switch prompt security-scan
cody-switch prompt handoff
```

These are not auto-installed into Codex because Codex does not expose the same slash-command install surface as Claude.

## Skills

Core skills in `global-skills/` install to `~/.codex/skills/`.

Optional extras in `global-skills-extra/` are opt-in. Some extras are still legacy and may require additional Codex-specific rework before they feel native.

The `sync-switch` core skill documents the maintenance workflow for checking `~/scripts/claude-switch`, deciding which upstream changes apply, porting them into `cody-switch`, and updating `docs/upstream/claude-switch-sync.md`.

`agent-hub` is treated as a shared, agent-agnostic companion tool. The Codex-facing integration is the `hub` skill; upstream dashboard/server changes should not be duplicated here unless this repo intentionally owns that shared component.

## Development

```bash
make bootstrap-dev
make test
```

That runs:
- `make check-dev-deps`
- `shellcheck --severity=error cody-switch`
- `bats test/`
