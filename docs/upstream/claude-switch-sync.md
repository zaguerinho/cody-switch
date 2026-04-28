# claude-switch Upstream Sync State

This file records the last inspected upstream `claude-switch` commit for the
Codex-native `cody-switch` port.

## State

- Upstream repo: `/Users/zaguerinho/scripts/claude-switch`
- Target repo: `/Users/zaguerinho/scripts/cody-switch`
- Last checked date: 2026-04-28
- Last checked commit: `ab94f97655effcca1e88d9c00387f2c4a446f50e`
- Previous known baseline: initial `cody-switch` import plus installer fallback commit `0d52278`

## 2026-04-28 Review

Reviewed upstream commits through:

```text
d46a755 docs: add agent-hub demo tutorial materials
395a637 docs: add claude-switch assessment and evolution notes
39446a8 docs(switch): prefer /clear over /compact for post-switch reset
e824336 fix: evict stale slash commands shadowed by skills during install
3b6561f chore: remove agent-hub release workflow (manual releases preferred)
```

## Decisions

- `d46a755`: Ported editable agent-hub tutorial markdown and print CSS in
  Codex-native form. Skipped generated PDF because it is Claude-branded output
  and can be regenerated from the adapted sources later.
- `395a637`: Ported the assessment/evolution intent as
  `docs/cody-switch-assessment-2026-04-28.md` and
  `docs/cody-switch-evolution-2026.md`.
- `39446a8`: Adapted the switch-skill guidance to recommend a fresh Codex
  session after classic switches, with current-session continuation reserved
  for intentional bridging.
- `e824336`: Skipped. The change evicts stale Claude slash commands shadowing
  skills. `cody-switch` does not install Claude slash commands.
- `3b6561f`: Already aligned. `cody-switch` does not ship an agent-hub release
  workflow.

## Files Updated In cody-switch

- `docs/hub-tutorial/agent-hub-demo-tutorial.md`
- `docs/hub-tutorial/print-style.css`
- `docs/cody-switch-assessment-2026-04-28.md`
- `docs/cody-switch-evolution-2026.md`
- `docs/upstream/claude-switch-sync.md`
- `global-skills/switch/SKILL.md`
- `global-skills/sync-switch/SKILL.md`
- `global-skills/sync-switch/agents/openai.yaml`
- `README.md`

## 2026-04-28 Follow-Up Review

Reviewed upstream commits through:

```text
ab94f97 feat(agent-hub): colorize dashboard dimensions
f3aaa9b feat(agent-hub): add dashboard message search and paging
4260cae fix(agent-hub): keep dashboard docs open
7f9b18c feat(agent-hub): default to bound room and rename agents
3c0ee06 feat(agent-hub): improve diagnostics and room bootstrap
4ceebbe feat(commands): adopt handoff.md as a managed global command
```

## Follow-Up Decisions

- `3c0ee06`, `7f9b18c`, `4260cae`, `f3aaa9b`, `ab94f97`: Did not copy
  `agent-hub` Go/dashboard implementation changes into `cody-switch`. Treat
  `agent-hub` as an agnostic shared Go component instead of a change stream to
  duplicate blindly. Updated only the Codex-facing `hub` skill so it knows about
  newer upstream CLI concepts: `doctor`, deep `health`, project room binding,
  `room bootstrap`, optional bound-room arguments, `rename`, and dashboard
  improvements.
- `4ceebbe`: Ported as a Codex-native prompt template instead of copying the
  upstream command. The cody-switch version uses `.codex/handoff`, avoids
  Claude slash-command assumptions, and only applies reviewer-ask closers when
  a topic protocol explicitly defines that convention.

## Follow-Up Files Updated In cody-switch

- `global-skills/hub/SKILL.md`
- `global-commands/handoff.md`
- `docs/upstream/claude-switch-sync.md`

## Next Sync

Use the `sync-switch` skill and compare:

```bash
git -C /Users/zaguerinho/scripts/claude-switch log --oneline --reverse ab94f97655effcca1e88d9c00387f2c4a446f50e..HEAD
git -C /Users/zaguerinho/scripts/claude-switch diff --name-status ab94f97655effcca1e88d9c00387f2c4a446f50e..HEAD
```
