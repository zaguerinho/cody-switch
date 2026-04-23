---
allowed-tools:
  - Bash(git *)
  - Bash(cody-switch *)
  - Bash(date *)
  - Read
  - Write
  - TaskCreate
  - TaskUpdate
---

# Stopping Point: Save Full Context for Feature Resume

When the user says "/stopping-point", "save context", "parking this", or is about to switch features — save everything needed to resume this exact context in a future session.

## Arguments

`$ARGUMENTS` may contain:
- No arguments — save a stopping point for the current feature
- `resume` — read and apply the existing stopping point to resume work
- `show` — display the current stopping point without acting on it

## Save Flow

1. **Identify the current feature**:
   ```bash
   cody-switch current
   ```
   If no active feature, warn and save to `tasks/STOPPING_POINT.md` anyway.

2. **Gather repo state**:
   ```bash
   git branch --show-current
   git log --oneline -3
   git status --short
   ```

3. **Read current task state**:
   - Read `tasks/todo.md` for progress context
   - Read `tasks/lessons.md` for session learnings

4. **Create `tasks/STOPPING_POINT.md`** using this template — omit any section that would be empty:

```markdown
# Stopping Point — [Feature Name]

**Saved**: YYYY-MM-DD HH:MM
**Feature**: [cody-switch feature name]
**Branch**: [branch name]
**Last commit**: [short hash] — [message]
**Uncommitted**: [file list or "clean"]

## Where We Left Off

[2-3 sentences: what was being worked on, what the last action was, what the next action should be]

## Session Progress

- [What was accomplished — include PR numbers, test results, key changes]

## Decisions Made

- [Any decisions that matter for context but aren't in code or docs yet]

## Blocking Items

- [What's blocked and by whom/what — omit section if nothing blocked]

## Files to Read on Resume

1. `tasks/STOPPING_POINT.md` (this file)
2. `tasks/todo.md`
3. [Most critical source file to re-read]
4. [...]

---

## Resume Checklist

- [ ] Read this file
- [ ] Read `tasks/todo.md` and `tasks/lessons.md`
- [ ] Check uncommitted changes with `git status`
- [ ] Confirm understanding with user before taking action
```

5. **Update `tasks/todo.md`** to reflect where things stand.

6. **Persist state with cody-switch**:
   ```bash
   cody-switch save
   cody-switch pin-session
   ```
   When prompted for a pin summary, use the "Where We Left Off" text.

7. **Write any pending memory updates** — if context would be lost without memory, save it now.

8. **Confirm**: "Stopping point saved and session pinned. To resume: `cody-switch <feature>` then `/stopping-point resume`"

## Resume Flow

When `$ARGUMENTS` is `resume`:

1. Read `tasks/STOPPING_POINT.md`
2. Read `tasks/todo.md` and `tasks/lessons.md`
3. Read the files listed in "Files to Read on Resume"
4. Check current repo state against the saved state:
   ```bash
   git log --oneline -3
   git status --short
   ```
5. Summarize what changed since the stopping point (if anything)
6. Present the resume context to the user and ask: "Ready to continue from where we left off?"

## Show Flow

When `$ARGUMENTS` is `show`:

1. Read and display `tasks/STOPPING_POINT.md`
2. No other action

## Rules

- **READ-ONLY on source files** — only write to `tasks/` files
- Be thorough but scannable — a fresh session should resume in under 60 seconds
- The stopping point file must be **self-contained** — don't assume memory or prior context
- Omit empty sections — a clean stopping point is a useful stopping point
- If multiple repos are involved, add a "Secondary Repos" section with branch/commit/status for each
