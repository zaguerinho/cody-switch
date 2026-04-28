# RFC: Parallel Jira Ticket Processing with Intelligent Orchestration

**Status:** Draft
**Date:** 2026-04-02
**Target Phase:** Post-2.5.0 (requires stable worktree + save infrastructure)

## Summary

A `/jira-batch` skill that accepts multiple Jira ticket IDs, analyzes them for file overlap conflicts before spawning any work, plans execution in waves (parallel when safe, sequential when tickets touch the same code areas), and creates worktree-backed features for each ticket. A companion `/jira-status` command provides real-time monitoring of all active batch work, including stale detection and recovery.

The key insight: the orchestration layer **thinks before spawning**. Instead of blindly launching N agents that may step on each other, it reads each ticket, predicts which areas of the codebase each would touch, detects overlaps, and produces a wave plan that maximizes parallelism while avoiding merge conflicts.

## User Flow

1. User runs `/jira-batch PROJ-101 PROJ-102 PROJ-103`
2. The skill fetches all three tickets from Jira (title, description, acceptance criteria)
3. For each ticket, it analyzes the codebase to predict affected file areas (using the same investigation that `/init-feature` does, but lighter — just area prediction, not full AGENTS.md generation)
4. It builds a conflict matrix: which tickets overlap in predicted file areas
5. It presents a wave plan:
   ```
   Wave 1 (parallel): PROJ-101 + PROJ-102  — no overlap
   Wave 2 (after wave 1): PROJ-103          — overlaps with PROJ-101 on auth/
   ```
6. User approves (or edits the plan)
7. For each ticket in wave 1:
   - Creates a worktree feature: `cody-switch blank PROJ-101 --worktree`
   - Writes initial state to `.codex/batch/PROJ-101.json`
   - Tells the user which terminal to open and what to run
8. User opens terminals, runs `/jira-process PROJ-101` in each worktree
9. User runs `/jira-status` at any time to see progress across all tickets
10. When wave 1 tickets create PRs and merge, `/jira-batch --next` triggers wave 2

## Data & Dependencies

- **Jira API access** — already used by `/jira-process` and `/process-issues`
- **cody-switch worktrees** — already implemented (v2.4.0+)
- **cody-switch save** — just shipped (v2.5.0) for clean commits from worktrees
- **Codebase analysis** — reuse `/init-feature`'s investigation patterns for area prediction
- **State storage** — new `.codex/batch/` directory for tracking batch state

### State Schema (`.codex/batch/{ticket-id}.json`)

```json
{
  "ticket": "PROJ-101",
  "title": "Refactor auth middleware",
  "wave": 1,
  "status": "in_progress|queued|pr_created|merged|stale|failed",
  "worktree": ".codex/worktrees/PROJ-101",
  "feature": "PROJ-101",
  "branch": "worktree-PROJ-101",
  "session_id": "abc123",
  "pr_url": null,
  "started_at": "2026-04-02T10:30:00Z",
  "last_activity": "2026-04-02T11:15:00Z",
  "predicted_areas": ["auth/", "middleware/", "tests/auth/"],
  "conflicts_with": ["PROJ-103"],
  "error": null
}
```

## MVP Scope (v1)

**The orchestrator plans, the user executes.**

1. `/jira-batch <ticket-ids...>` — analyze, detect conflicts, produce wave plan
2. Wave plan creates worktree features automatically
3. `.codex/batch/` state tracking (JSON files per ticket)
4. `/jira-status` — read state files, show progress table, detect stale sessions
5. `/jira-batch --next` — check if current wave is done, prepare next wave
6. Manual terminal spawning — user opens terminals and runs `/jira-process` themselves
7. Status updates via git activity detection (last commit time in worktree branch)
8. `/jira-batch --cleanup` — archive merged worktree features, move state to done/
9. `/jira-batch --next` — auto-cleanup wave N, prepare wave N+1

**What MVP does NOT include:**
- Automated process spawning (no tmux/background)
- Automated PR creation coordination
- Cross-ticket dependency analysis beyond file overlap
- Automatic merge conflict resolution

## Future Enhancements

- **Automated spawning (v2):** Use tmux or background Codex sessions to launch agents automatically. `/jira-batch` becomes fully hands-off.
- **Live progress streaming:** A `/jira-watch` command that tails all active worktree sessions in a dashboard view.
- **Smart merge ordering:** After all wave PRs are ready, suggest the optimal merge order to minimize conflicts.
- **Cross-ticket dependencies:** Read Jira ticket links (blocks/is-blocked-by) and factor them into wave planning.
- **Adaptive re-planning:** If a ticket touches unexpected files during implementation, flag the conflict and pause queued tickets that overlap.
- **Batch templates:** Common patterns like "process all tickets in sprint X" or "process all bugs in component Y."
- **Integration with `/process-issues`:** Triage → ticket creation → batch processing as a single pipeline.

## Technical Approach

### Conflict Prediction

The hardest part. Two strategies, used together:

1. **Keyword-to-area mapping:** Parse ticket title/description for domain keywords (auth, payment, user, API) and map to known codebase areas. Fast but imprecise.

2. **AI codebase analysis:** For each ticket, ask Codex to predict which files/directories would need changes. Uses the same pattern as `/init-feature`'s investigation but outputs a list of paths instead of a full AGENTS.md. More accurate but slower — run in parallel per ticket.

Conservative approach: if prediction confidence is low, assume conflict and sequence. False sequencing is annoying but safe. False parallelism causes merge hell.

### Wave Execution Model

```
Batch = [Wave1, Wave2, ..., WaveN]
Wave  = [Ticket, Ticket, ...]  (all non-conflicting)
```

Graph coloring problem: build a conflict graph (edge = shared predicted files), then color it. Each color = one wave. Minimize wave count = maximize parallelism.

In practice, most batches will be 3-6 tickets, so brute force is fine.

### Cleanup Lifecycle

Each ticket follows: `create → work → pr_created → merged → cleanup`

Cleanup for a ticket means:
1. `cody-switch archive {ticket}` — syncs worktree to storage, removes worktree dir + branch, moves feature to archived/
2. Move `.codex/batch/{ticket}.json` to `.codex/batch/done/` (preserves history)

Cleanup triggers:
- **`/jira-batch --cleanup`** — explicitly check for merged PRs, archive those features
- **`/jira-batch --next`** — automatically cleans up merged wave N tickets before preparing wave N+1
- **`/jira-status`** — detects merged PRs and flags them as ready for cleanup (doesn't auto-clean)

The building blocks already exist: `cody-switch archive` handles worktree removal, and `gh pr view --json state` detects merged PRs. The batch skill just orchestrates the calls.

### Stale Detection

A session is "stale" if:
- `last_activity` (last commit in worktree branch) > 2 hours ago
- AND status is still `in_progress`

Recovery options:
- Resume the session: `cd $(cody-switch open PROJ-101) && codex resume <session_id>`
- Restart from scratch: delete worktree, recreate, re-run
- Skip and move to next wave

## Recommendation

**This is a high-value feature and the right time to build it.** The infrastructure is all in place — worktrees, save, session tracking, Jira integration. The orchestration layer is the natural capstone that turns individual tools into a pipeline.

**Build MVP (v1) first and use it for a sprint before automating.** The manual terminal spawning is actually a feature, not a limitation — it keeps the user in control while validating that the conflict prediction and wave planning work correctly. If the predictions are wrong, you want to catch that while you're watching, not while a background agent silently creates merge conflicts.

**The conflict prediction is the make-or-break piece.** Invest time in making it good. A bad predictor that either (a) sequences everything (no parallelism benefit) or (b) misses conflicts (merge hell) defeats the purpose. Start conservative (over-sequence), then loosen as confidence grows.

**Risk:** scope creep toward building a CI/CD system. Keep the boundary clear — this orchestrates Codex sessions working on tickets, it doesn't replace GitHub Actions or deployment pipelines. The output is PRs, not deployed code.

## Open Questions

1. **How to detect "done"?** A ticket is "done" when its PR is created — but should the orchestrator wait for PR merge before starting the next wave, or just PR creation? Merge depends on reviewers (external), PR creation is within the agent's control.

2. **What if a ticket is bigger than expected?** The agent might need multiple sessions. Should the orchestrator track session count per ticket and flag "complex" tickets?

3. **Should wave planning be editable after start?** If wave 1 ticket A turns out to not touch auth/ at all, could we promote wave 2 ticket C to run immediately?

4. **Naming convention:** Should worktree features use ticket IDs (`PROJ-101`) or slugs (`auth-middleware-refactor`)? IDs are unique but less readable. Slugs risk collision.
