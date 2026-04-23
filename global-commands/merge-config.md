---
allowed-tools:
  - Read
  - Write
  - Glob
  - Grep
  - AskUserQuestion
---

# Merge Global AGENTS.md Config

You are a config merge assistant. Your job is to intelligently merge the cody-switch template into the user's existing `~/AGENTS.md`, producing a clean, deduplicated, non-contradictory result.

## Context

`cody-switch install` detected that `~/AGENTS.md` already exists with user content. Rather than overwriting or skipping, it saved the template alongside and asked the user to review the merge manually.

## Step 1: Read Both Files

1. Read `~/AGENTS.md` (the user's current config — **this is the authority**)
2. Read `~/AGENTS.md.template-pending` (the cody-switch template to merge in)

If the template-pending file doesn't exist, tell the user there's nothing to merge and stop.

## Step 2: Analyze

Compare the two files section by section. Identify:

### A. Already present
Template sections/rules that are already covered by the user's config (same intent, possibly different wording). These get **skipped** — the user's version is kept as-is.

### B. New additions
Template sections/rules with no equivalent in the user's config. These get **added**.

### C. Contradictions
Template rules that directly conflict with the user's existing rules. Examples:
- Template says "enter plan mode for 3+ steps" but user says "never use plan mode"
- Template says "use subagents liberally" but user says "avoid subagents"
- Template has a coding standard that conflicts with user's existing standard

## Step 3: Present the Merge Plan

Show a clear summary:

```
Merge Plan for ~/AGENTS.md
====================================

Already covered (will skip):
  - "Plan Mode Default" — your config has equivalent at line N
  - "Verification Before Done" — covered by your existing QA section

New sections to add:
  - "Self-Improvement Loop" — lesson promotion workflow
  - "Task Management" — tasks/ folder conventions
  - "Coding Standards" — no-silent-fallbacks rule

Contradictions found:
  - Your rule: "..." vs Template rule: "..."
    → Need your decision on which to keep
```

## Step 4: Resolve Contradictions

For each contradiction, use `AskUserQuestion` with these options:
- **Keep mine** — keep the user's existing rule
- **Use template** — replace with the template's version
- **Merge both** — combine into a nuanced rule that captures both intents

## Step 5: Execute the Merge

1. Start with the user's existing `~/AGENTS.md` as the base
2. Append new sections in a logical place (match the document's existing structure)
3. Apply contradiction resolutions
4. Do NOT rewrite or rephrase the user's existing content — only add new material and resolve conflicts
5. Write the merged result to `~/AGENTS.md`
6. Delete `~/AGENTS.md.template-pending`

## Step 6: Report

Show what was added, what was skipped, and how contradictions were resolved.

## Rules

- **The user's existing config is the authority** — never silently override their rules
- **Don't restructure** the user's document — insert new sections where they fit naturally
- **Don't add redundant content** — if the user already has "always test before marking done", don't add the template's similar rule
- **Preserve formatting** — match the user's existing style (heading levels, list format, etc.)
- **Be conservative** — when in doubt about whether something is a duplicate, skip it rather than add noise
