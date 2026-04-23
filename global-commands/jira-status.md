---
allowed-tools:
  - Bash(git *)
  - Bash(gh *)
  - Bash(cody-switch *)
  - Bash(cat *)
  - Bash(ls *)
  - Bash(python3 *)
  - Bash(date *)
  - Read
---

# Jira Batch Status

Show the current state of all tickets in the active `/jira-batch` batch. This is a read-only status check.

## Step 1: Read Batch State

Check if a batch exists:

```bash
cat .codex/batch/plan.json 2>/dev/null
```

If no plan.json exists, say: "No active batch. Run `/jira-batch <ticket-keys...>` to start one." and stop.

Read the plan to get the wave structure and current wave number.

## Step 2: Gather Status for Each Ticket

For each `.codex/batch/*.json` file (excluding `plan.json`):

1. **Read the state file** to get ticket, title, wave, status, branch
2. **Check worktree exists:**
   ```bash
   [ -d .codex/worktrees/{TICKET-KEY} ] && echo "exists" || echo "missing"
   ```
3. **Count commits on the worktree branch:**
   ```bash
   git log --oneline worktree-{TICKET-KEY} --not main 2>/dev/null | wc -l
   ```
4. **Get last commit time:**
   ```bash
   git log -1 --format="%ci" worktree-{TICKET-KEY} 2>/dev/null
   ```
5. **Check for PR:**
   ```bash
   gh pr list --head worktree-{TICKET-KEY} --json number,state,url --limit 1
   ```

## Step 3: Derive Status

For each ticket, determine the current status from observations:

| Observed State | Derived Status |
|---|---|
| State file says "queued", no worktree | `queued` — waiting for earlier wave |
| Worktree exists, 0 commits beyond main | `ready` — worktree created, not started |
| Worktree exists, has commits, no PR | `in_progress` — work ongoing |
| PR exists, state is "OPEN" | `pr_created` — awaiting review |
| PR exists, state is "MERGED" | `merged` — ready for cleanup |
| PR exists, state is "CLOSED" | `closed` — investigate |
| `in_progress` + last commit > 2 hours ago | `stale` — may need attention |

## Step 4: Display Table

Format the output as a clear status table grouped by wave:

```
Jira Batch Status
==================
Plan: {total} tickets across {wave_count} waves

Wave 1 (current):
  ELEM-101  [in_progress]  5 commits, last 35m ago    Auth middleware refactor
  ELEM-102  [pr_created]   PR #47 — awaiting review   Payment validation rules

Wave 2 (queued):
  ELEM-103  [queued]       waiting on wave 1           Auth token rotation

Stale: none

Commands:
  /jira-batch --next      Advance to next wave
  /jira-batch --cleanup   Archive merged worktrees
```

If any tickets are stale, highlight them:

```
⚠ Stale sessions (no commits in >2 hours):
  ELEM-101  last commit 3h ago — resume with:
    cd $(cody-switch open ELEM-101) && codex resume <session-id>
```

## Step 5: Check Done Directory

Also report any recently completed tickets:

```bash
ls .codex/batch/done/*.json 2>/dev/null | wc -l
```

If there are done tickets, add a summary line:
```
Completed: {N} tickets archived in .codex/batch/done/
```

## Rules

- This command is **read-only** — never modify state files, worktrees, or git state
- If git or gh commands fail for a specific ticket, show "unknown" status, don't crash
- Keep the output concise — one line per ticket in the table
- Always show the available commands at the bottom
