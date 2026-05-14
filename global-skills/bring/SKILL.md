---
name: bring
description: >
  Bring the latest applicable upstream changes from ~/scripts/claude-switch
  into this Codex-native cody-switch port. Use when the user says bring,
  transfer, sync upstream, port claude-switch changes, or asks for parity with
  claude-switch.
---

# bring

Use this as the user-friendly entry point for upstream parity work. It delegates
to the `sync-switch` workflow and keeps the public trigger short.

## Workflow

1. Read `global-skills/sync-switch/SKILL.md` or the installed `sync-switch`
   skill before acting.
2. Use `docs/upstream/claude-switch-sync.md` as the state ledger. Its
   `Last checked commit` is the comparison base.
3. Compare `~/scripts/claude-switch` against the recorded commit, classify each
   upstream change, and port only the parts that make sense for Codex.
4. Adapt names and behavior:

- `claude-switch` -> `cody-switch`
- `CLAUDE.md` -> `AGENTS.md`
- `.claude/features` -> `.codex/features`
- `global-claude.md` -> `global-agents.md`
- Claude slash commands -> `cody-switch prompt ...`
- `claude --resume` / `--fork-session` -> `codex resume` / `codex fork`

5. Update all relevant surfaces when behavior changes:

- `cody-switch`
- `README.md`
- `completions/cody-switch.sh`
- `global-agents.md`
- `global-skills/*/SKILL.md`
- `test/`

6. Update `docs/upstream/claude-switch-sync.md` with the previous and new
   upstream commit, decision notes, files changed, and verification commands.
7. Run the relevant checks, normally `bash -n cody-switch` and `make test`.

## Guardrails

- Prefer selective ports over raw copies.
- Keep user-facing docs Codex-native.
- Skip Claude-only runtime surfaces unless they can be expressed as Codex
  prompts or skills.
- Do not overwrite unrelated local changes.
