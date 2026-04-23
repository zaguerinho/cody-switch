---
allowed-tools:
  - Bash(git *)
  - Read
  - Grep
  - Glob
  - Agent
---

# Code Review

Perform a thorough code review of current changes. Spawns a review subagent that analyzes the diff for bugs, security issues, performance problems, and convention violations.

## Arguments

`$ARGUMENTS` may contain:
- `staged` — only review staged changes
- `branch` — review all commits on current branch vs main
- `<file>` — review a specific file
- No arguments — review all uncommitted changes (staged + unstaged)

## Step 1: Gather the Diff

Based on arguments:

- **No args:** `git diff HEAD` (all uncommitted changes)
- **staged:** `git diff --cached`
- **branch:** `git diff main...HEAD`
- **file:** `git diff HEAD -- <file>`

If the diff is empty, say "No changes to review" and stop.

Also gather context:
```bash
git diff --stat HEAD  # (or appropriate variant)
git log --oneline -5
```

## Step 2: Spawn Review Agent

Launch a subagent with the following prompt:

> You are a senior code reviewer. First, read the project's root AGENTS.md and any relevant nested AGENTS.md files to understand the stack, conventions, and coding standards. Then review the following changes thoroughly.
>
> **Review checklist:**
>
> ### Correctness
> - Logic errors, off-by-one, null/undefined handling
> - Missing edge cases (empty collections, null references, unauthenticated users)
> - Incorrect ORM/database usage (N+1 queries, missing eager loads, wrong relationships)
> - Race conditions in concurrent operations
> - Type mismatches or unsafe casts
>
> ### Security
> - SQL injection (raw queries with user input, string interpolation in queries)
> - XSS (unescaped user output, dangerouslySetInnerHTML, {!! !!})
> - Mass assignment or over-posting vulnerabilities
> - Missing authorization/authentication checks
> - CSRF protection gaps
> - Sensitive data exposure (tokens, passwords, internal IDs in responses)
>
> ### Performance
> - N+1 query patterns
> - Unnecessary queries or computations inside loops
> - Missing database indexes for new query patterns
> - Oversized API/page payloads
> - Synchronous operations that should be async/queued
>
> ### Conventions
> - Follow the project's established patterns (read AGENTS.md for specifics)
> - Tests covering new behavior
> - Proper types (no `any` in TypeScript, proper type hints in PHP/Python)
> - Consistent error handling (no silent fallbacks)
>
> ### Testing Gaps
> - New code paths without test coverage
> - Edge cases that should be tested (time-dependent logic, boundary conditions)
> - Missing regression tests for bug fixes
>
> **Output format:**
>
> For each finding, report:
> ```
> [SEVERITY] Category — file:line
> Description of the issue.
> Suggested fix (if applicable).
> ```
>
> Severities: CRITICAL (must fix), WARNING (should fix), INFO (consider)
>
> End with a summary:
> ```
> Review Summary
> ==============
> Critical: N
> Warning:  N
> Info:     N
> Verdict:  APPROVE / NEEDS CHANGES
> ```

The subagent should read the changed files to understand full context, not just the diff lines.

## Step 3: Present Results

Show the subagent's review output to the user. If there are CRITICAL findings, highlight them prominently.

## Safety Rules

- This command is READ-ONLY — it never modifies files
- If the diff is very large (50+ files), warn the user and suggest reviewing a subset
