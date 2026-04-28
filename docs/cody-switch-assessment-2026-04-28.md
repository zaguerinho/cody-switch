# cody-switch Assessment

Date: 2026-04-28

This note is the Codex-native counterpart to the upstream
`claude-switch` assessment added in `claude-switch` commit `395a637`.

## Summary

`cody-switch` is usable for its core purpose: feature-scoped Codex context
switching around `AGENTS.md`, per-feature `tasks/`, feature docs, session IDs,
and optional worktree-backed parallel features.

The project is best understood as three related surfaces:

1. A feature context switcher: `cody-switch`
2. A reusable workflow kit: prompt templates and installable Codex skills
3. Companion tooling: `agent-hub` and the legacy `video-tutorial` extra

The Codex port deliberately does not mirror Claude-only surfaces. The replacement
model is:

- `AGENTS.md` instead of `CLAUDE.md`
- `.codex/features/` instead of `.claude/features/`
- `cody-switch prompt ...` instead of installed Claude slash commands
- `cody-switch current` instead of a Claude `SessionStart` hook
- `codex resume` and `codex fork` for session continuation

## Current Strengths

- The core file/state model is simple enough to inspect and recover manually.
- Worktree-backed features support isolated parallel work without mutating root context.
- JSON output exists for automation and skill routing.
- `doctor` gives a practical health check for missing/empty feature context.
- The install path now has a user-level fallback when `/usr/local/bin` cannot be written non-interactively.
- Core skills install into `~/.codex/skills/`.

## Confirmed Gaps

### Full Shell Test Verification Depends On Local Tooling

The root `make test` target requires `shellcheck` and `bats`. If those tools are
missing, only partial local verification is possible. CI should remain the
authoritative cross-platform check.

### Optional Companion Binaries Are Not Fully Productized

The `hub` skill declares a companion `agent-hub` binary, but installation can
only download it when a matching GitHub release artifact exists. Until then,
users must build it from source with:

```bash
cd ~/scripts/cody-switch/agent-hub && make build
```

### `video-tutorial` Is Still Legacy

The `video-tutorial` extra remains opt-in and still contains inherited
Claude-coupled behavior. Treat it as legacy until it is explicitly reworked for
Codex.

### Feature Switching Still Mutates Root Context

Classic switching rewrites root `AGENTS.md` and `tasks/`. That is usable, but
the long-term direction should be context composition rather than only context
replacement.

## Product Direction

Keep the parts that are working:

- feature-scoped tasks
- feature-scoped docs
- worktree isolation
- session continuity
- recovery and diagnostics
- skills for repeated workflows

Reduce reliance on a single mutable root file over time. The stronger product
direction is a Codex-oriented context router that can compose:

- root durable `AGENTS.md`
- feature-specific instructions
- feature docs
- task state
- skills
- MCP-backed live docs
- subagent/worktree workflows

## Practical Readiness

For day-to-day use, `cody-switch` is ready for:

- `cody-switch init`
- `cody-switch blank/new/fork`
- `cody-switch list/current`
- classic feature switches
- worktree feature creation/open/sync
- save/doctor/session pinning
- prompt template lookup
- core skill installation

Before broader release, the highest-value hardening work is:

- keep CI green on macOS and Linux
- build/publish `agent-hub` release artifacts
- remove or clearly label remaining Claude-coupled legacy extra behavior
- continue moving reusable workflows into Codex skills
