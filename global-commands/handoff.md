# Handoff Manager

Manage multi-topic, multi-agent handoff folders for Codex projects. Use this
prompt to check status, read messages, post substantive updates, archive tracks,
or bootstrap a new handoff topic.

## Arguments

`$ARGUMENTS` may contain:
- empty or `--check`: show current status across all topics
- `--status <topic>`: show manifesto progress for a specific topic
- `--check-in <topic>`: read latest unread messages and summarize changes
- `--post <topic>`: write a new handoff message after a substance gate
- `--archive <topic>`: move resolved conversation files to `archive/`
- `--topics`: list all active handoff topics
- `--setup <topic>`: create a new handoff topic structure

## Storage

Default location:

```text
.codex/handoff/<topic>/
  STATUS.md
  PROTOCOL.md
  QUALITY_MANIFESTO.md
  archive/
```

Also support a flat layout when it already exists:

```text
.codex/handoff/STATUS.md
```

Do not write to `.claude/handoff/` in this port unless the user explicitly asks
to inspect or migrate a legacy Claude handoff folder.

## Command: --check

1. Discover handoff folders under `.codex/handoff/`.
2. Read each `STATUS.md`.
3. Extract agent status, open action items, and last activity.
4. Show the most actionable items first.

Suggested output:

```text
=== Handoff Status ===

topic: release-readiness
location: .codex/handoff/release-readiness/
agents: Cody active, Reviewer pending
manifesto: 12/18 complete
action: Reviewer needs to check q007
```

## Command: --status <topic>

1. Read the topic's `QUALITY_MANIFESTO.md`.
2. Extract progress tables or checkbox counts.
3. Summarize green/yellow/red areas and the next concrete action.

Never modify `QUALITY_MANIFESTO.md` without explicit user request.

## Command: --check-in <topic>

1. Read `STATUS.md` for the last reviewed timestamp or marker.
2. Find message files newer than that marker.
3. Summarize each new message and mark what changed.
4. Ask before updating `STATUS.md` unless the user clearly requested state updates.

## Command: --post <topic>

First, gate on substance:

> Is this a substantive handoff, or is it a trivial follow-up that is obvious
> from the diff or commit?

Post only for substantive coordination:
- Multi-issue review fix-pass
- Design or RFC round
- Phase-gate unblock
- Specific question another agent asked
- Non-obvious implementation decision that needs asynchronous review

Do not post for ceremony:
- One-line obvious fixes
- Restating a commit message
- Reviewer already approved in principle
- Routine status that is better captured by the PR or commit

If posting:
1. Determine the next message number by scanning existing files.
2. Ask for sender, recipient, subject, and message body.
3. Write a concise q-numbered message file.
4. Update `STATUS.md` only with the new action item and timestamp.

## Command: --archive <topic>

1. Show active message files outside `archive/`.
2. Ask whether to archive all or a named track.
3. Create `archive/<track>/`.
4. Move selected files.
5. Report the count and leave `STATUS.md` consistent.

## Command: --topics

List each topic with:
- Topic name
- Location
- Agent names from `PROTOCOL.md`
- Manifesto score or summary
- Last activity
- Current blocker or "none"

## Command: --setup <topic>

Create:

```text
.codex/handoff/<topic>/STATUS.md
.codex/handoff/<topic>/PROTOCOL.md
.codex/handoff/<topic>/QUALITY_MANIFESTO.md
.codex/handoff/<topic>/archive/
```

Before writing, ask for:
- Topic purpose
- Agent names and roles
- Initial owner
- What counts as done

After setup, tell the user the next useful action: write onboarding notes,
define manifesto dimensions, or post the first substantive question.

## Rules

- Only write handoff files unless the user explicitly asks otherwise.
- Prefer concise summaries over long transcripts.
- Keep output scannable for frequent operator checks.
- Preserve existing protocol conventions if a topic already defines them.
- Do not require a reviewer-ask closer globally. Use one only when the topic's
  `PROTOCOL.md` explicitly defines that convention.
- When in doubt about whether to post, do not post. Summarize locally and ask.
