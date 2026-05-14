---
name: sync-switch
description: >
  Compare the upstream ~/scripts/claude-switch project against this cody-switch
  Codex port, identify new commits, classify each upstream change, port the
  applicable behavior/docs/tests in Codex-native form, skip Claude-only surfaces,
  and update the recorded last-checked upstream commit. Use when the user asks
  to pull, sync, port, review, or bring over changes from claude-switch to
  cody-switch.
---

# sync-switch

Use this skill to maintain `cody-switch` as a Codex-native port of
`~/scripts/claude-switch`.

For the user-facing trigger name, prefer the `bring` skill. `bring` delegates
to this workflow and exists so users can ask to "bring" or "transfer" upstream
changes without knowing the maintenance skill name.

## Defaults

- Upstream repo: `~/scripts/claude-switch`
- Target repo: `~/scripts/cody-switch`
- Sync state: `docs/upstream/claude-switch-sync.md`
- Primary branch: `main`

## Workflow

1. Verify both repos exist and read their current branch/status:

```bash
git -C ~/scripts/claude-switch status --short --branch
git -C ~/scripts/cody-switch status --short --branch
```

2. Read `docs/upstream/claude-switch-sync.md` if present and use its
   `last_checked_commit` as the comparison base.

3. Inspect upstream changes:

```bash
git -C ~/scripts/claude-switch log --oneline --reverse <last_checked_commit>..HEAD
git -C ~/scripts/claude-switch diff --name-status <last_checked_commit>..HEAD
```

If the recorded commit is missing or empty, inspect recent history manually with
`git log --oneline -20`.

4. Classify each upstream change:

- **Port directly** when it affects shared behavior, tests, CI, docs, or generic skill guidance.
- **Adapt** when it is useful but mentions Claude-specific filenames, commands, hooks, or slash-command behavior.
- **Skip** when it depends on Claude-only runtime surfaces that Codex does not expose.
- **Defer** when it is generated, binary, too large, or requires a separate product decision.

5. Apply Codex-native naming during ports:

- `claude-switch` -> `cody-switch`
- `CLAUDE.md` -> `AGENTS.md`
- `.claude/features` -> `.codex/features`
- `global-claude.md` -> `global-agents.md`
- `~/.claude/commands` -> bundled prompt templates via `cody-switch prompt`
- `claude --resume` -> `codex resume`
- `claude --resume ... --fork-session` -> `codex fork`
- `SessionStart hook` -> manual `cody-switch current`, unless explicitly documenting legacy Claude behavior

6. Do not reintroduce Claude-only claims into user-facing docs. If legacy
   material must stay, label it as legacy.

7. When behavior changes, update all relevant cody-switch surfaces:

- `cody-switch`
- `README.md`
- `completions/cody-switch.sh`
- `global-agents.md`
- `global-skills/*/SKILL.md`
- `test/`

8. Update `docs/upstream/claude-switch-sync.md` with:

- upstream path
- previous and new `last_checked_commit`
- date checked
- commit-by-commit decision list
- files changed in `cody-switch`
- verification commands run

9. Verify what is available locally:

```bash
bash -n cody-switch
make test
go test ./...
node test/unit.js
```

Run Go/Node checks from their component directories as needed. If local tools
such as `shellcheck` or `bats` are missing, report that explicitly.

## Porting Rules

- Prefer selective ports over raw copy.
- Keep the target repo clean and Codex-native.
- Preserve generated or binary artifacts only when they are intentionally
  source-controlled in `cody-switch`; otherwise keep editable sources and skip
  generated outputs.
- Do not overwrite unrelated local changes.
- After porting, commit only if the user asked for a durable sync or publish.
