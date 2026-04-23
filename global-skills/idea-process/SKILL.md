---
name: idea-process
description: >
  Processes RFC documents from docs/incoming/ through an evaluation pipeline.
  Use when the user says "/idea-process", "process ideas", "triage RFCs", or "plan ideas".
  Evaluates each RFC for clarity and viability, interviews the user when unclear,
  plans approved ideas into the project plan, and moves completed RFCs to done/.
---

# Idea Process Pipeline

Process unplanned RFC documents from `docs/incoming/` through a triage → prioritize → interview → plan → integrate pipeline.

## Process

### 1. Discover RFCs

List all `.md` files in `docs/incoming/` — **exclude** the `done/` and `parked/` subdirectories and any non-RFC files. Read each file's contents.

If no unprocessed RFCs exist, tell the user and stop.

**Check for stale RFCs:** If any RFC has a date older than 14 days and hasn't been processed, flag it in the triage summary so the user can decide to process, park, or discard it.

### 2. Triage Each RFC

For each RFC, classify it into one of three buckets:

| Verdict | Criteria | Action |
|---------|----------|--------|
| **Ready** | Clear problem, defined user flow, sound technical approach, no blocking open questions | Proceed to prioritization |
| **Needs Clarity** | Good core idea but has unresolved questions, vague user flow, missing technical details, or conflicting approaches | Interview the user |
| **Not Ready** | Fundamentally flawed, too vague to evaluate, duplicates existing functionality, or contradicts architectural decisions | Flag to user with reasons, suggest parking or discarding |

**Present the triage summary to the user as a table before proceeding.** Example:

```
| RFC | Verdict | Age | Reason |
|-----|---------|-----|--------|
| captcha-and-2fa.md | Ready | 2d | Clear scope, concrete technical choices |
| ai-recommendations.md | Needs Clarity | 5d | Depends on unbuilt system |
| half-baked-idea.md | Not Ready | 21d ⚠️ | No defined user flow, unclear value |
```

Wait for the user to confirm or override verdicts before continuing.

**For "Not Ready" RFCs**, ask the user:
- **Park it** → move to `docs/incoming/parked/` for future revisiting
- **Discard it** → delete the file (confirm first)
- **Keep it** → leave in `docs/incoming/` for next processing round

### 3. Prioritize (for "Ready" RFCs)

Before planning, help the user decide **what to plan now vs. later**. If there are multiple Ready RFCs:

1. Present them with a quick impact/effort assessment:

```
| RFC | Impact | Effort | Recommendation |
|-----|--------|--------|----------------|
| captcha-and-2fa.md | High (security) | Medium (~3 sessions) | Plan now |
| dark-mode.md | Low (cosmetic) | Small (~1 session) | Quick win, plan now |
| ai-recommendations.md | High (differentiator) | Large (~8 sessions) | Defer to next cycle |
```

2. Ask the user to confirm which RFCs to plan in this round
3. Unselected Ready RFCs stay in `docs/incoming/` for the next round — they don't need to be re-triaged

If there's only one Ready RFC, skip this step.

### 4. Interview (for "Needs Clarity" RFCs)

For each RFC that needs clarity, **one at a time**:

1. Read the **Open Questions** section and any ambiguous parts

2. Use the **gap analysis framework** — check for these common missing pieces and ask about any that are absent:
   - **Who is the user?** Is the target user clear, or could this serve multiple audiences differently?
   - **What triggers this?** What user action or system event kicks off this feature?
   - **What's the happy path?** Can you walk through the ideal flow step by step?
   - **What happens on failure?** Error states, edge cases, fallback behavior
   - **What's the MVP vs. the full vision?** Can this be shipped in a smaller first version?
   - **What are the hard dependencies?** Does this need something that doesn't exist yet?
   - **What data is needed?** New tables, external APIs, migrations to existing data?

3. Ask the user **specific, concrete questions** — not vague "what do you think?" prompts. Frame questions as choices where possible.

4. After getting answers, **update the RFC file** in-place with the clarified details

5. Re-evaluate: does it now qualify as "Ready"? If not, explain what's still missing and ask if the user wants to continue clarifying or park it for later.

### 5. Plan (for "Ready" RFCs selected in prioritization)

For each RFC being planned:

#### 5a. Determine Target Phase

Check the project's plan documentation (e.g., `docs/master-plan/`, `docs/plan/`, or equivalent) to find where this feature belongs. Consider:
- What does it depend on? (must come after its dependencies)
- What phase's scope does it fit? (don't shove everything into one phase)
- Does it need a new phase or sub-step? (rare, but possible)

#### 5b. Break Into Implementation Steps

Create concrete, estimable work items:
- Database changes (migrations, models, seeders)
- Backend logic (services, controllers, commands, jobs)
- Frontend (pages, components, API integration)
- Tests

Each step should be ~1 session of work (a few hours), not multi-day epics.

**Scope check:** If the feature produces more than 6 implementation steps, it's too big for a single plan entry. Split it:
- Identify the **MVP slice** (from the RFC's "MVP Scope" section) and plan that as the primary entry
- Create a separate follow-up entry for remaining steps, explicitly marked as a future iteration
- Tell the user about the split and why

#### 5c. Write the Plan

Add a new numbered section to the appropriate phase file. Follow the existing format. Include:
- Step title and brief description
- Sub-steps with enough detail to implement
- Dependencies on other steps
- Estimated scope (small/medium/large)
- If split from a larger feature, note what comes next

#### 5d. Update the Project Plan Dashboard

In the plan's index or README:
- Add the new step to the "What's Next" section if it's in the current or next phase
- Update any dependency notes if relevant

**Keep updates to one authoritative location.** If the project maintains planning info in multiple files (e.g., a master plan README *and* a root project doc), update the master plan as the source of truth. Only update secondary docs if the project explicitly uses them as dashboards — don't create sync burdens.

### 6. Move to Done

After an RFC has been successfully planned and integrated:

1. Move the file: `docs/incoming/{file}.md` → `docs/incoming/done/{file}.md`
2. Update the RFC's status from `Draft` to `Planned`
3. Add a note at the top: `**Planned into:** Phase X, Step X.Y — {title}`

### 7. Report

After processing all RFCs, provide a summary:

```
## Processing Complete

| RFC | Verdict | Result |
|-----|---------|--------|
| captcha-and-2fa.md | Ready | Planned → Phase 1.10, moved to done/ |
| ai-recommendations.md | Needs Clarity | Clarified → Ready, planned → Phase 3.2 (MVP only, full version deferred) |
| half-baked-idea.md | Not Ready | Parked — needs user flow definition |
| old-idea.md | Not Ready | Discarded by user |

Files modified:
- docs/master-plan/PHASE-1-FOUNDATION.md (added step 1.10)
- docs/master-plan/README.md (updated dashboard)
```

## Important

- **Never auto-plan a questionable idea.** If you're unsure, ask. A bad plan is worse than no plan.
- **Respect existing phase structure.** Don't renumber or reorganize phases — append new steps within existing phases.
- **Don't duplicate existing functionality.** Cross-reference the project's existing documentation to make sure the RFC isn't asking for something that already exists.
- **Check dependencies.** If an RFC depends on unbuilt features, note this in the plan. Don't block planning — just document the dependency.
- **Keep plans actionable.** Each step should be concrete enough that someone could start implementing without re-reading the RFC. Reference specific models, tables, controllers, and files.
- **One RFC at a time through the interview stage.** Don't batch-interview — resolve each RFC's questions fully before moving to the next.
- **Preserve the user's intent.** When writing plans, stay true to the RFC's vision. Add implementation details, don't change the feature.
- **Bias toward smaller scope.** When in doubt, plan the MVP and defer the rest. A shipped feature beats a perfect spec.
- **Keep the parking lot clean.** Parked RFCs aren't forgotten — suggest revisiting them periodically. But they shouldn't clutter the active incoming queue.
