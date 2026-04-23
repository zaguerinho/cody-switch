---
allowed-tools:
  - Bash(git *)
  - Read
  - Grep
  - Glob
  - Agent
---

# Security Scan

Run a security audit on current changes or the full codebase. Adapts to the project's stack automatically.

## Arguments

`$ARGUMENTS` may contain:
- `diff` — scan only changed files (default)
- `full` — scan the entire codebase source directories
- `<path>` — scan a specific directory or file

## Step 1: Identify the Stack

Read the project's root AGENTS.md and any relevant nested AGENTS.md files to understand the tech stack, framework, and conventions. This determines which scan rules apply.

## Step 2: Determine Scope

- **diff (default):** Get changed files from `git diff --name-only HEAD` and `git diff --name-only --cached`
- **full:** Scan source directories (e.g., `app/`, `src/`, `routes/`, `resources/`, `config/`, `database/`)
- **path:** Scan the specified path

If no changes in diff mode, say "No changes to scan" and stop.

## Step 3: Spawn Security Scanner

Launch a subagent to scan the files:

> You are a security auditor. First, read the project's root AGENTS.md and any relevant nested AGENTS.md files to understand the stack. Then scan the specified files for vulnerabilities relevant to that stack.
>
> **Universal scan categories:**
>
> ### Injection
> - SQL injection: raw queries with user input, string interpolation in queries
> - Command injection: shell commands with user-controlled arguments
> - Template injection: user input in template rendering
>
> ### Authentication & Authorization
> - Missing auth checks on sensitive endpoints
> - Missing ownership verification on resource access
> - Privilege escalation paths
> - Sensitive operations without re-authentication
>
> ### XSS / Output Encoding
> - Unescaped user output in HTML (dangerouslySetInnerHTML, {!! !!}, v-html, etc.)
> - User input in meta tags, URLs, or attributes without sanitization
>
> ### Data Exposure
> - Sensitive fields leaked in API responses or page props (passwords, tokens, internal IDs)
> - Debug output left in code (dd(), console.log with secrets, print_r)
> - Environment values accessed directly instead of through config
> - Error messages exposing stack traces or internal paths
>
> ### Input Validation
> - Mass assignment / over-posting vulnerabilities
> - Missing file upload validation (type, size, path traversal)
> - Missing rate limiting on auth or sensitive endpoints
>
> ### CSRF / Request Forgery
> - State-changing endpoints without CSRF protection
> - API routes that should be authenticated but aren't
>
> ### Configuration
> - Debug mode in production references
> - Insecure session/cookie settings
> - Hardcoded secrets or credentials
>
> **Output format:**
>
> ```
> [SEVERITY] Category — file:line
> Finding description.
> Impact: What could an attacker do?
> Fix: How to remediate.
> ```
>
> Severities: CRITICAL (exploitable now), HIGH (likely exploitable), MEDIUM (defense-in-depth), LOW (best practice)
>
> End with:
> ```
> Security Scan Summary
> =====================
> Critical: N
> High:     N
> Medium:   N
> Low:      N
> Status:   PASS / FAIL
> ```
>
> PASS = no Critical or High findings. FAIL = at least one Critical or High.

## Step 3: Present Results

Show findings to the user. For CRITICAL/HIGH findings, include the exact file and line number so they can navigate directly.

## Safety Rules

- This command is READ-ONLY — it never modifies files
- Do not report false positives for framework-provided protections (e.g., parameterized queries, built-in CSRF tokens)
- Do not flag raw SQL in migrations — those are safe (no user input at migration time)
