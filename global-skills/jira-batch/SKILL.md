---
name: jira-batch
description: >
  Process multiple Jira tickets in parallel with conflict detection and wave
  orchestration. Use when the user says "/jira-batch", "batch tickets",
  "process multiple tickets", "parallel jira", or provides multiple ticket IDs
  to process simultaneously.
argument-hint: "<TICKET-KEY...> [--next|--cleanup|--plan-only]"
---

# Jira Batch: Parallel Ticket Orchestration

You are an orchestration specialist. Your job is to analyze multiple Jira tickets, detect file overlap conflicts, plan execution in waves that maximize parallelism, create isolated worktrees for each ticket, and track state so the developer can monitor progress.

**You plan and prepare. The developer runs each ticket's implementation in separate terminals.**

## Arguments

Parse `$ARGUMENTS` for:
- **Ticket keys** — one or more Jira ticket keys (e.g., `ELEM-101 ELEM-102 ELEM-103`)
- **`--next`** — advance to the next wave (cleanup current wave, prepare next)
- **`--cleanup`** — archive all merged batch worktrees
- **`--plan-only`** — analyze and show wave plan without creating worktrees

If `--next` or `--cleanup` is passed, skip to the corresponding section below.

## State Files

All batch state lives in `.codex/batch/` at the project root:
- `.codex/batch/plan.json` — wave structure, ticket list, current wave
- `.codex/batch/{TICKET-KEY}.json` — per-ticket state
- `.codex/batch/done/` — archived state files after cleanup

## Step 1: Check for Existing Batch

```bash
cat .codex/batch/plan.json 2>/dev/null
```

If a batch already exists:
- Show the current plan and state
- Ask the developer: **Continue existing batch**, **Reset and start fresh**, or **Add tickets to current batch**?
- If reset: `rm -rf .codex/batch/*.json` (keep done/)

If no batch exists, proceed.

## Step 2: Fetch All Tickets

For each ticket key in the arguments, fetch details from Jira:

```bash
php artisan tinker --execute="
    \$jira = app(\App\Services\JiraService::class);
    \$ticket = \$jira->getTicket('TICKET_KEY');
    echo json_encode(\$ticket, JSON_PRETTY_PRINT);
"
```

For each ticket, extract and note:
- **Summary** — what the ticket is about
- **Description / Acceptance Criteria** — what needs to be built
- **Status** — must be in a workable state (not "Done", not "In Review")
- **Story Points** — scope indicator

If a ticket is already "Done" or "In Review", warn and exclude it from the batch.

If fetching fails (Jira unavailable, ticket not found), report the error and exclude that ticket. If ALL tickets fail, stop.

## Step 3: Predict Affected Areas

This is the critical step. For each ticket, predict which areas of the codebase would need changes.

**Use Explore agents in parallel** — launch one per ticket (up to 3 at a time) with this prompt:

> Ticket: {TICKET_KEY} — {summary}
> Description: {description}
> Acceptance Criteria: {criteria}
>
> Analyze the codebase and predict which directories and files would need to be created or modified to implement this ticket. Consider:
> - Which models, controllers, services, or components are involved?
> - Which test files would be affected?
> - Which config or migration files?
>
> Return a JSON object with:
> ```json
> {
>   "ticket": "TICKET_KEY",
>   "predicted_areas": ["app/Services/Auth/", "app/Http/Middleware/", "tests/Feature/Auth/"],
>   "confidence": "high|medium|low",
>   "reasoning": "Brief explanation of why these areas"
> }
> ```
> Be conservative — if unsure, include the area. False positives (over-predicting) are safe. False negatives (missing an area) cause merge conflicts.

If more than 3 tickets, batch the Explore agents in groups of 3.

## Step 4: Build Conflict Matrix

Compare predicted areas pairwise. Two tickets conflict if ANY of their predicted areas share a common directory prefix.

Example:
- ELEM-101 predicts: `app/Services/Auth/`, `app/Http/Middleware/`
- ELEM-102 predicts: `app/Services/Payment/`
- ELEM-103 predicts: `app/Services/Auth/`, `app/Models/Token.php`

Conflict matrix:
- ELEM-101 ↔ ELEM-103: CONFLICT (both touch `app/Services/Auth/`)
- ELEM-101 ↔ ELEM-102: no conflict
- ELEM-102 ↔ ELEM-103: no conflict

Build an adjacency list of conflicts for wave planning.

## Step 5: Plan Waves

Use greedy graph coloring to assign tickets to waves:

1. Sort tickets by number of conflicts (most constrained first)
2. For each ticket, assign to the lowest-numbered wave where it has no conflicts
3. This minimizes the number of waves (maximizes parallelism)

If no conflicts exist, all tickets go in wave 1.

Present the wave plan:

```
Wave Plan ({N} tickets, {M} waves)
====================================

Wave 1 (parallel):
  ELEM-101: Auth middleware refactor        → app/Services/Auth/, app/Http/Middleware/
  ELEM-102: Payment validation rules        → app/Services/Payment/

Wave 2 (after wave 1 merges):
  ELEM-103: Auth token rotation             → app/Services/Auth/, app/Models/Token.php
  ⚠ Conflicts with: ELEM-101 (app/Services/Auth/)

Estimated parallelism: {wave1_count} / {total_count} tickets in first wave
```

If `--plan-only` was passed, stop here.

Use `AskUserQuestion` to get approval. Options:
- **Approve** — create worktrees and proceed
- **Edit waves** — manually adjust which tickets go in which wave
- **Cancel** — abort without creating anything

## Step 6: Create Worktree Features

For each ticket in wave 1:

```bash
cody-switch blank {TICKET-KEY} --worktree
```

Then create the state directory and write state files:

```bash
mkdir -p .codex/batch/done
```

Write `.codex/batch/{TICKET-KEY}.json` for EACH ticket (all waves, not just wave 1):

```json
{
  "ticket": "{TICKET-KEY}",
  "title": "{ticket summary}",
  "wave": {wave_number},
  "status": "ready",
  "worktree": ".codex/worktrees/{TICKET-KEY}",
  "feature": "{TICKET-KEY}",
  "branch": "worktree-{TICKET-KEY}",
  "predicted_areas": ["{area1}", "{area2}"],
  "conflicts_with": ["{other-ticket}"],
  "created_at": "{ISO-8601 timestamp}",
  "pr_url": null
}
```

For wave 1 tickets, set `"status": "ready"`.
For later wave tickets, set `"status": "queued"`.

Write `.codex/batch/plan.json`:

```json
{
  "tickets": ["{all ticket keys}"],
  "waves": [
    {"wave": 1, "tickets": ["{wave 1 keys}"]},
    {"wave": 2, "tickets": ["{wave 2 keys}"]}
  ],
  "current_wave": 1,
  "created_at": "{ISO-8601 timestamp}"
}
```

## Step 7: Show Instructions

Display clear instructions for the developer:

```
✅ Batch ready! {N} worktrees created for wave 1.

Open a terminal for each ticket:

  Terminal 1: ELEM-101 — Auth middleware refactor
    cd $(cody-switch open ELEM-101)
    codex
    # Then run the project ticket-processing prompt/skill for ELEM-101

  Terminal 2: ELEM-102 — Payment validation rules
    cd $(cody-switch open ELEM-102)
    codex
    # Then run the project ticket-processing prompt/skill for ELEM-102

Monitor progress:  /jira-status
Advance to wave 2: /jira-batch --next
Clean up merged:   /jira-batch --cleanup
```

---

## --next: Advance to Next Wave

When `$ARGUMENTS` contains `--next`:

1. **Read the plan:**
   ```bash
   cat .codex/batch/plan.json
   ```

2. **Check current wave completion** — for each ticket in the current wave:
   ```bash
   gh pr list --head worktree-{TICKET-KEY} --json number,state,url --limit 1
   ```
   A ticket is "done" if a PR exists (any state — open, merged, or closed).

3. **Report status:**
   - If all current wave tickets have PRs → proceed
   - If some are pending → show which ones, ask: **Wait**, **Force advance** (skip pending), or **Cancel**

4. **Cleanup completed tickets** — for each ticket with a merged PR:
   ```bash
   cody-switch archive {TICKET-KEY}
   mv .codex/batch/{TICKET-KEY}.json .codex/batch/done/
   ```
   For tickets with open (unmerged) PRs, leave the worktree intact.

5. **Advance wave counter:**
   Update `plan.json` → `"current_wave": {next_wave}`

6. **Check if there's a next wave:**
   - If yes → create worktree features for next wave tickets, show terminal instructions
   - If no more waves → report "Batch complete! All waves processed."

7. **Update state files** — set next wave tickets to `"status": "ready"`

---

## --cleanup: Archive Merged Work

When `$ARGUMENTS` contains `--cleanup`:

1. **Read all state files:**
   ```bash
   ls .codex/batch/*.json 2>/dev/null | grep -v plan.json
   ```

2. **For each ticket**, check if the PR is merged:
   ```bash
   gh pr list --head worktree-{TICKET-KEY} --state merged --json number --limit 1
   ```

3. **For merged tickets:**
   ```bash
   cody-switch archive {TICKET-KEY}
   mv .codex/batch/{TICKET-KEY}.json .codex/batch/done/
   ```

4. **Report:**
   ```
   Cleanup Summary
   ================
   Archived: ELEM-101 (PR #47 merged), ELEM-102 (PR #48 merged)
   Still open: ELEM-103 (PR #49 — awaiting review)
   Queued: (none)
   ```

5. **If all tickets are done**, clean up the plan file:
   ```bash
   mv .codex/batch/plan.json .codex/batch/done/
   ```
   Report: "Batch fully complete. All tickets processed and cleaned up."

---

## Rules

1. **Never create worktrees for tickets beyond the current wave** — they queue until earlier waves complete
2. **Never skip conflict analysis** — even for "obvious" non-conflicts, run the prediction
3. **Conservative on conflicts** — if prediction confidence is "low", assume conflict and sequence
4. **Always get plan approval** before creating worktrees
5. **Never force-cleanup unmerged work** — only archive tickets with merged PRs (unless developer explicitly requests)
6. **State files are the source of truth** — always read them before making decisions
