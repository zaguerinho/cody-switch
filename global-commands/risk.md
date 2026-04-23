---
allowed-tools:
  - Bash(git *)
  - Read
  - Grep
  - Glob
  - Agent
---

# Risk Assessment

Analyze current changes and report a confidence percentage that they won't cause regressions or break existing features. Think of this as a pre-commit gut check from a senior engineer.

## Arguments

`$ARGUMENTS` may contain:
- `staged` — only assess staged changes
- `branch` — assess all commits on current branch vs main
- `<file>` — assess changes to a specific file
- No arguments — assess all uncommitted changes (staged + unstaged)

## Step 1: Gather Context

Based on arguments:

- **No args:** `git diff HEAD` (all uncommitted changes)
- **staged:** `git diff --cached`
- **branch:** `git diff main...HEAD`
- **file:** `git diff HEAD -- <file>`

If the diff is empty, say "No changes to assess" and stop.

Also gather:
```bash
git diff --stat HEAD  # (or appropriate variant)
git log --oneline -10
```

## Step 2: Spawn Risk Analyst

Launch a subagent with the following prompt:

> You are a principal engineer performing a risk assessment on code changes. Your job is to estimate how likely these changes are to cause regressions, break existing features, or introduce subtle bugs.
>
> First, read the project's root AGENTS.md and any relevant nested AGENTS.md files to understand the stack, architecture, and conventions. Then read the full content of every changed file (not just the diff) to understand the surrounding context.
>
> **Evaluate these risk dimensions:**
>
> ### 1. Blast Radius
> - How many files changed? How many lines?
> - Are changes isolated to one module/feature, or scattered across the codebase?
> - Do changed files have many dependents? (Check for imports/requires of changed files)
> - Score: Contained (low risk) → Widespread (high risk)
>
> ### 2. Change Type
> - **Low risk:** Documentation, comments, formatting, test-only changes, adding new files
> - **Medium risk:** New features with no existing code modified, config changes, dependency updates
> - **High risk:** Refactoring existing logic, changing function signatures, modifying shared utilities, database migrations, auth/security changes
> - Score based on the highest-risk change type present
>
> ### 3. Test Coverage
> - Do the changes include new/updated tests?
> - Are the tests testing the actual changed behavior (not just boilerplate)?
> - Is there existing test coverage for the modified code? (Check for test files that import changed modules)
> - Score: Well-tested (low risk) → No tests for changed logic (high risk)
>
> ### 4. Critical Path Exposure
> - Do changes touch: authentication, authorization, payment processing, data persistence, API contracts, database schema, encryption, session management?
> - Are there changes to error handling that could mask failures?
> - Score: No critical paths (low risk) → Core security/data paths (high risk)
>
> ### 5. Interface Stability
> - Do any function/method signatures change? (parameters added/removed/reordered)
> - Do API response shapes change?
> - Are database columns/tables added, modified, or removed?
> - Are configuration keys renamed or removed?
> - Score: No interface changes (low risk) → Breaking changes (high risk)
>
> ### 6. Side Effect Potential
> - Do changes modify shared state, global variables, or singleton patterns?
> - Are there changes to initialization order or lifecycle hooks?
> - Could the changes affect behavior in code paths not directly modified?
> - Score: Pure/isolated (low risk) → Global side effects (high risk)
>
> ### 7. Rollback Difficulty
> - If these changes cause problems in production, how hard is it to revert?
> - Database migrations, data transformations, and external API changes are hard to roll back
> - Pure code changes with no state changes are easy to roll back
> - Score: Easy revert (low risk) → Irreversible (high risk)
>
> **Scoring:**
>
> For each dimension, assign a risk level: LOW, MEDIUM, or HIGH.
>
> Then compute an overall **Safety Confidence %** — your estimate that these changes will NOT cause any regression or breakage. Use this rough calibration:
>
> - **95-100%**: Trivial changes — docs, comments, formatting, isolated additions with tests
> - **85-94%**: Low risk — new features that don't touch existing code, well-tested refactors
> - **70-84%**: Moderate risk — refactoring existing logic, changes to shared code with decent tests
> - **50-69%**: Elevated risk — touching critical paths, incomplete test coverage, interface changes
> - **30-49%**: High risk — widespread changes to core logic, no tests, breaking interfaces
> - **0-29%**: Dangerous — untested changes to auth/data/payments, irreversible migrations
>
> **Output format:**
>
> ```
> Risk Assessment
> ===============
>
> Safety Confidence: XX%
> [A one-sentence verdict: what makes this safe or risky]
>
> Dimension Breakdown:
>   Blast Radius:        LOW/MEDIUM/HIGH — [brief reason]
>   Change Type:         LOW/MEDIUM/HIGH — [brief reason]
>   Test Coverage:       LOW/MEDIUM/HIGH — [brief reason]
>   Critical Paths:      LOW/MEDIUM/HIGH — [brief reason]
>   Interface Stability: LOW/MEDIUM/HIGH — [brief reason]
>   Side Effects:        LOW/MEDIUM/HIGH — [brief reason]
>   Rollback Difficulty: LOW/MEDIUM/HIGH — [brief reason]
>
> Key Risks:
> - [Most important risk factor, with specific file:line references]
> - [Second most important, if applicable]
> - [...]
>
> Mitigations Present:
> - [What's already in place that reduces risk — tests, type safety, etc.]
>
> Recommendations:
> - [What would increase confidence — specific tests to add, things to verify]
> ```
>
> Be honest and calibrated. Don't inflate confidence to be nice. A senior engineer shipping to production is relying on this assessment.

The subagent should read the full changed files for context, not just the diff lines.

## Step 3: Present Results

Show the risk assessment to the user. If confidence is below 70%, highlight the key risks prominently and emphasize the recommendations.

Add a visual indicator:

- **95-100%**: "Ship it."
- **85-94%**: "Looks good. Minor items to consider."
- **70-84%**: "Proceed with caution. Review the recommendations."
- **50-69%**: "Significant risk. Address the key risks before merging."
- **Below 50%**: "Hold up. These changes need more work before they're safe to ship."

## Safety Rules

- This command is READ-ONLY — it never modifies files
- If the diff is very large (50+ files), warn the user and suggest assessing a subset
- Be calibrated: don't give 95% to untested changes just because they "look simple"
