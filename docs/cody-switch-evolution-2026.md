# cody-switch Evolution Note

Date: 2026-04-28

This note ports the upstream `claude-switch` evolution direction into the
Codex-native `cody-switch` project.

## Why This Exists

The original idea remains valid: coding agents perform better when active
context is focused on the current feature instead of carrying every project
detail at once.

What changed is the platform shape. Modern agent environments are moving toward
smaller composable primitives:

- layered instruction files
- on-demand skills
- MCP-backed live tools and documentation
- native subagents with isolated context
- git worktrees for implementation isolation

`cody-switch` should evolve with that direction instead of becoming only a
better root `AGENTS.md` swapper.

## Current Model

Today, a classic `cody-switch` feature activates by replacing root files:

- root `AGENTS.md`
- root `tasks/`
- state markers such as `.codex-current-feature`

This is simple and effective, but it is context replacement.

## Target Model

The stronger model is context composition:

- keep root `AGENTS.md` short and durable
- store feature-specific instructions under `.codex/features/{name}/`
- keep task state feature-scoped
- use skills for multi-step procedures
- use prompt templates for reusable one-off instructions
- use MCP where live authoritative docs matter
- use worktrees and subagents for parallel or sidecar work

## Recommended Direction

### 1. Thin Root Instructions

Root `AGENTS.md` should contain stable project rules:

- architecture overview
- test/build commands
- non-negotiable conventions
- security and workflow constraints

Feature-specific detail should stay in feature storage, docs, or skills.

### 2. Feature Bundles

A feature should eventually behave like a bundle:

- instructions
- tasks
- docs
- related skills
- suggested validation commands
- optional worktree metadata
- optional live docs/MCP references

### 3. Skills-First Procedures

Procedural workflows should live in skills instead of bloating feature
instructions. Examples:

- `switch` for feature operations
- `sync-switch` for upstream porting from `claude-switch`
- `hub` for agent coordination
- review/audit/promotion workflows

### 4. Vendor-Aware Adapters

The source model can be vendor-neutral, but outputs should be explicit:

- Codex: `AGENTS.md`, `~/.codex/skills/`, `codex resume`, `codex fork`
- Claude legacy: `CLAUDE.md`, slash commands, `SessionStart`
- GitHub Copilot: repository and path-scoped instruction files

Do not blur these surfaces in user-facing docs.

### 5. Subagent And Worktree Awareness

Use worktrees for isolated implementation state and subagents for bounded
research/review/verification. Context switching should not be the only way to
keep the main session clean.

## Bottom Line

`cody-switch` should remain a practical tool first. The next evolution is not
more clever file swapping; it is a Codex-native context orchestration layer that
activates the right files, skills, docs, sessions, and workspaces for the task.
