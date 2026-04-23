# cody-switch

Codex-native feature context switching built around `AGENTS.md`.

## Scope

- Root instructions live in `AGENTS.md`.
- Feature storage lives in `.codex/features/{name}/`.
- Active state markers are `.codex-current-feature`, `.codex-last-seen-feature`, and `.codex-with-refs`.
- User-level global instructions live in `~/AGENTS.md`.
- User-level lessons live in `~/.codex/lessons-global.md`.
- Installed skills go to `~/.codex/skills/`.

## Porting Rules

- Keep Codex-facing terminology accurate. Do not reintroduce Claude-only claims unless explicitly marked legacy.
- `cody-switch prompt ...` is the replacement for auto-installed Claude slash commands.
- `cody-switch current` is the replacement for the old Claude startup hook behavior.
- Session help must use `codex resume` and `codex fork`.

## Files To Keep In Sync

When behavior changes, update all relevant surfaces:
1. `cody-switch`
2. `README.md`
3. `completions/cody-switch.sh`
4. `global-agents.md`
5. `global-skills/switch/SKILL.md`
6. tests under `test/`

## Testing

Run:

```bash
make test
```

That should cover:
- `shellcheck --severity=error cody-switch`
- `bats test/`

## Notes

- The `video-tutorial` extra is still legacy and may require additional Codex-native rework.
- Prompt templates live in `global-commands/` for now even though they are no longer installed as slash commands.
- Internal helper names may still contain historical `claude` identifiers in a few places; user-facing behavior and docs should remain Codex-native.
