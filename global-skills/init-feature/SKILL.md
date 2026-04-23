---
name: init-feature
description: >
  Bootstraps a cody-switch feature with deep codebase investigation. Use when the user
  says "/init-feature", "bootstrap feature", "initialize feature", or wants to populate a
  new feature's AGENTS.md with real context from codebase analysis. Takes a feature description
  as argument and produces focused AGENTS.md, research docs, and an initial implementation plan.
---

# Feature Initialization

Bootstrap an active cody-switch feature by investigating the codebase and writing focused working context.

**Input:** `/init-feature <description of what the feature does>`

## Process

### 1. Parse Input and Validate

Extract the feature description from the argument after `/init-feature`.

If no description is provided, ask: **"What does this feature do? Describe it in 1-2 sentences."**

Detect the active feature:

1. Find the project root (walk up from `$PWD` to find `.git`)
2. Read `.codex-current-feature` at the project root
3. If the file doesn't exist or is empty, **stop** with:
   > No active feature. Create one first with `cody-switch blank <name>`, then run `/init-feature` again.
4. Store the feature name for use in file paths

Verify `docs/{feature}/` exists. If not, create it.

### 2. Check for Existing Content

Read the project root `AGENTS.md` and check if it has content beyond the minimal scaffold:

```
# Feature: {name}

<!-- Add feature-specific context here -->
```

**How to detect:** If the file has more than 5 non-empty lines or more than 150 characters of actual content (excluding the scaffold pattern), it has custom content.

If custom content exists:
- Show the user the first ~20 lines
- Ask: **"AGENTS.md already has content. Overwrite with investigation results, or keep it and only write research docs?"**
- If they choose to keep it, skip AGENTS.md writing in Step 5 but still produce `research.md` and `tasks/todo.md`

Also check if `docs/{feature}/research.md` already exists. If so, note it and ask before overwriting.

### 3. Reconnaissance (Quick Scan)

Before spawning subagents, do a fast scan to understand the project:

1. **Root-level files** — read whichever exist: `README.md`, `package.json`, `Cargo.toml`, `pyproject.toml`, `go.mod`, `Makefile`, `composer.json`, `build.gradle`, `pom.xml`, `CMakeLists.txt`
2. **Project structure** — list top-level directories to understand organization
3. **Existing project AGENTS.md** — check the project's root `AGENTS.md` and any nested `AGENTS.md` files for conventions
4. **Tech stack** — identify language(s), framework(s), build system, test framework

This gives enough context to design targeted investigation tracks.

### 4. Deep Investigation

Investigate the codebase areas relevant to the described feature. **Adapt depth to project size:**

- **Small project** (< 30 files): investigate inline, no subagents needed
- **Medium/large project**: spawn 2-3 Explore subagents in parallel

#### Investigation Tracks

**Track A: Architecture & Patterns**
- Project's architectural patterns (MVC, service layer, event-driven, etc.)
- How similar features are structured — look for examples in the codebase
- Module/package organization relevant to the feature
- Conventions: naming, file organization, error handling, logging

**Track B: Relevant Code & Data Flows**
- Files, functions, and modules directly related to the feature description
- Data flows through relevant areas (input → transform → persist)
- Interfaces and contracts the feature will interact with
- Configuration, environment variables, or feature flags in the area

**Track C: Testing & Dependencies**
- Test framework, patterns, and existing tests near the feature area
- Test utilities, fixtures, or factories that would be relevant
- What the feature area depends on (libraries, services, other modules)
- What depends on areas the feature will touch (blast radius)
- Potential constraints: database schemas, API contracts, shared state

Provide each subagent with:
- The feature description
- Tech stack findings from recon
- Specific investigation focus and what to look for
- Instruction to return file paths, function names, and brief explanations

### 5. Synthesize and Write

After investigation completes, write three outputs:

#### Output 1: `docs/{feature}/research.md`

```markdown
# Research: {feature-name}

**Date:** {YYYY-MM-DD}
**Description:** {user's feature description}

## Codebase Overview

{Tech stack, build system, project structure — only what's relevant to the feature.}

## Relevant Architecture

{Patterns, modules, data flows that the feature intersects.
Include specific file paths and brief explanations.}

## Existing Related Code

{Files and functions the feature will interact with, modify, or extend.
Group by area/module. Include file paths and line references.}

## Testing Landscape

{Test framework, patterns, relevant existing tests.
Available test utilities and fixtures.}

## Dependencies & Constraints

{What the feature depends on. What depends on areas it will touch.
Database schemas, API contracts, shared state — anything constraining.}

## Open Questions

{Things the investigation couldn't resolve. Decisions needing user input.}
```

#### Output 2: Root `AGENTS.md`

Only write this if the user approved (Step 2). Follow the workflow.md template structure, filled with real findings:

```markdown
# Feature: {feature-name}

> `tasks/` is auto-managed by `cody-switch`. When you switch features, your current
> `tasks/` is saved and the target feature's is restored. A fresh scaffold is created
> automatically for new features.

## Context

{1-2 paragraphs: What problem does this feature solve? What is the goal?
Derived from user's description + what the investigation revealed about current state.
Be specific about what exists today and what needs to change.}

## Architecture

{Design decisions based on investigation:
- Patterns to follow (reference existing code that exemplifies them)
- How this feature fits into the existing architecture
- Trade-offs and constraints discovered
- Conventions to maintain (naming, error handling, etc.)}

## Key Files

{Files this feature will modify or create. For each:
- `path/to/file.ext` — what changes are needed and why
This is the most actionable section — a developer reading this
should know exactly where to start.}

## Notes

{Edge cases, open questions, related features.
Reference: See `docs/{feature}/research.md` for full investigation details.}
```

#### Output 3: `tasks/todo.md`

Rewrite the existing scaffold with a draft implementation plan:

```markdown
# TODO

## Plan

> Draft plan from `/init-feature` investigation. Review and adjust before starting.

- [ ] {First step — most foundational change}
- [ ] {Second step — builds on first}
- [ ] {Third step}
- [ ] {Tests for the above}
- [ ] {Integration/cleanup if needed}

## Review

_(filled after implementation)_
```

Keep to **4-8 items**. Each item should:
- Be concrete enough to act on (reference specific files or modules)
- Be ordered by dependency (foundational first, then what builds on it, tests last)
- Include test items alongside or after the code they cover

### 6. Report Summary

Present a concise summary:

```
## Feature Initialized: {feature-name}

**Files written:**
- `AGENTS.md` — feature context and working instructions
- `docs/{feature}/research.md` — detailed investigation findings
- `tasks/todo.md` — draft implementation plan (N items)

**Key findings:**
- {Most important architectural insight}
- {Key constraint or dependency}
- {Notable pattern to follow}

**Open questions:**
- {Unresolved questions from investigation}

Review `tasks/todo.md` and adjust the plan before starting.
```

## Important

- **Project-agnostic.** Never assume specific languages, frameworks, or tools. Discover everything from the codebase. This works for any project.
- **Focused investigation.** Only investigate areas relevant to the described feature. The goal is working context, not a codebase encyclopedia.
- **Concrete over abstract.** Key Files should list actual paths with specific descriptions. Architecture should reference real patterns found in the code, not generic advice.
- **Respect existing content.** Always check AGENTS.md and research.md before overwriting. Ask if substantial content exists.
- **Tasks are suggestions.** Mark the todo plan as "Draft" and tell the user to review. The investigation may miss intent or get priorities wrong.
- **Reference, don't duplicate.** AGENTS.md is the briefing (~1-2 pages). research.md is the evidence (as long as needed). AGENTS.md references research.md for details.
- **Adapt investigation depth.** Small project = inline. Large codebase = parallel Explore subagents. Don't over-orchestrate trivial codebases.
- **Fail loudly.** If there's no active feature, don't try to create one. If something critical is missing, tell the user clearly.
