# claude-switch Upstream Sync State

This file records the last inspected upstream `claude-switch` commit for the
Codex-native `cody-switch` port.

## State

- Upstream repo: `/Users/zaguerinho/scripts/claude-switch`
- Target repo: `/Users/zaguerinho/scripts/cody-switch`
- Last checked date: 2026-04-28
- Last checked commit: `d46a755acab35c0afe479050d8a9d71f166fc697`
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

## Next Sync

Use the `sync-switch` skill and compare:

```bash
git -C /Users/zaguerinho/scripts/claude-switch log --oneline --reverse d46a755acab35c0afe479050d8a9d71f166fc697..HEAD
git -C /Users/zaguerinho/scripts/claude-switch diff --name-status d46a755acab35c0afe479050d8a9d71f166fc697..HEAD
```
