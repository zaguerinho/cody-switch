---
name: idea-capture
description: >
  Captures product ideas and feature concepts into RFC documents in docs/incoming/.
  Use when the user says "/capture", "capture this idea", "new idea", "RFC", or describes
  a feature concept they want documented. Reflects on the idea, adds structured feedback,
  and recommends an approach.
---

# Idea Capture

Capture a product idea or feature concept as a structured RFC in `docs/incoming/`.

## Process

### 1. Parse the Idea

Extract the core concept, user-facing behavior, and any specifics the user mentioned.

### 2. Sanity Check (before writing anything)

Before creating an RFC, do a quick validation:

1. **Check for duplicates** — scan `docs/incoming/` and `docs/incoming/done/` for existing RFCs on similar topics. If one exists, suggest updating it instead of creating a new one.
2. **Check for conflicts** — review the project's root documentation (e.g., `AGENTS.md`, `README.md`, or equivalent) and existing architecture. If the idea clearly contradicts an architectural decision or duplicates something already built, **flag it immediately** rather than writing a full RFC that will get rejected during processing.
3. **Check for completeness** — if the idea is so vague that you can't write even a basic user flow (e.g., "we should do something with AI"), ask the user to expand before capturing. A one-sentence idea is fine; a one-word idea isn't.

If any check fails, explain the issue and ask the user how they'd like to proceed. Don't silently create an RFC that's dead on arrival.

### 3. Generate Filename

Create a kebab-case filename from the idea (e.g., "AI venue recommendations" → `ai-venue-recommendations.md`). Verify no collision exists in `docs/incoming/`.

### 4. Write the RFC

Write the document to `docs/incoming/{filename}.md` with this structure:

```markdown
# RFC: {Title}

**Status:** Draft
**Date:** {YYYY-MM-DD}
**Target Phase:** {Best-fit phase from the project's plan, or "TBD"}

## Summary

{1-2 paragraph description of the idea in the user's voice, expanded with clarity}

## User Flow

{Numbered steps showing how a user would experience this feature}

## Data & Dependencies

{What data/systems are needed, what must exist first}

## MVP Scope

{What's the smallest useful version of this feature? Strip it down to the core
value proposition. Everything else goes in "Future Enhancements."}

## Future Enhancements

{Nice-to-haves, follow-up iterations, and features that can come later.
This is where the full vision lives — but it's explicitly marked as "not first."}

## Recommendation

{Your honest assessment: is this a good idea? What approach would you take?
Include tradeoffs, risks, and a suggested implementation strategy.
Be specific — recommend concrete technical choices, not vague options.
If the idea is risky or low-value, say so directly.}

## Open Questions

{Unresolved decisions or things to validate}

## Monetization Potential

{If applicable — how could this generate revenue or reduce costs?}
```

### 5. Adapt the Template

Skip sections that don't apply, add sections that do (e.g., "Technical Approach" for engineering-heavy ideas, "UX Considerations" for design-heavy ones). Don't force every section.

### 6. Be Opinionated

The **Recommendation** section is the most important part. The user wants your honest take — not "here are 3 options, you decide." Pick the best approach and explain why. Flag risks or reasons it might not work. If the idea is solving a non-problem or the effort outweighs the value, say that clearly.

### 7. Report Back

Provide a brief summary: the idea, your key recommendation, the MVP vs full scope distinction, and the file path.

## Important

- **Push back on bad ideas.** Being a good collaborator means flagging problems early, not just documenting everything uncritically. A rejected idea at capture is cheaper than a rejected RFC at processing.
- **Keep it concise** — aim for 1-2 pages, not a novel. Details get fleshed out during processing.
- **Reference existing project context** where relevant. Check root documentation for what's built, what's planned, and what the architecture looks like.
- **Preserve the user's intent.** Expand and structure, but don't reshape the idea into something different. If you think a different approach is better, say so in the Recommendation — don't silently substitute it.
- **The user's raw idea may be informal or incomplete** — that's fine. A few sentences is enough to capture. The interview step in idea-process will fill gaps later.
