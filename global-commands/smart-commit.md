---
disable-model-invocation: true
allowed-tools:
  - Bash(git *)
  - Bash(gh api *)
  - Bash(gh pr create *)
  - Bash(echo *)
  - AskUserQuestion
---

# Smart Commit

You are a commit assistant. Your job is to analyze all pending changes and group them into logical, well-structured commits — never lumping everything into one big commit.

## Arguments

`$ARGUMENTS` may contain:
- `all` — scan ALL repos visible in this session (primary + additional working directories)
- `dry-run` — propose commits but don't execute them
- `single` — force a single commit (traditional behavior)
- A file path or glob — only include matching files

Arguments can be combined: `/smart-commit all dry-run`

## Step 0: Discover Repos

**If `$ARGUMENTS` does NOT contain `all`:** Skip this step. Only operate on the current working directory.

**If `$ARGUMENTS` contains `all`:**

1. The primary repo is the current working directory. Confirm it with:
   ```bash
   git rev-parse --show-toplevel
   ```

2. Look at the session's "Additional working directories" listed in the environment context. For each unique directory path, resolve its git repo root:
   ```bash
   git -C <directory_path> rev-parse --show-toplevel 2>/dev/null
   ```

3. Deduplicate the results — multiple subdirectories often resolve to the same repo root.

4. If no additional repos are found, use `AskUserQuestion` to ask:
   > "I couldn't detect additional repos from the session. Enter the path(s) to include, or choose 'Current repo only'."

5. Store the unique repo roots as the **repo list** for all subsequent steps.

**Example:** If additional working directories include:
```
/workspace/tenant-tools/app/Models
/workspace/tenant-tools/app/Services
/workspace/tenant-tools/database/migrations
```
All three resolve to `/workspace/tenant-tools` — one repo, not three.

## Step 0.5: Sync Feature Context (cody-switch)

For each repo in scope, check if it uses cody-switch:

```bash
[ -f <repo_root>/.codex-current-feature ] && command -v cody-switch >/dev/null
```

If both are true, sync the active feature's context to storage before inventorying:

```bash
(cd <repo_root> && cody-switch save 2>/dev/null)
```

This ensures `.codex/features/` is up-to-date so the commit naturally includes enriched feature context for team sharing. The save is idempotent — if nothing changed, no files are modified.

## Step 1: Inventory

**Single-repo mode (no `all`):**

```
!git status
!git diff --stat
!git diff --cached --stat
!git log --oneline -8
```

**Multi-repo mode (`all`):**

For each repo in the repo list, run (using `git -C <repo_root>`):

```
!git -C <repo_root> status
!git -C <repo_root> diff --stat
!git -C <repo_root> diff --cached --stat
!git -C <repo_root> log --oneline -8
```

Label each block clearly:
```
=== propmgmt-new (/Users/.../propmgmt-new) ===
[output]

=== tenant-tools (/Users/.../tenant-tools) ===
[output]
```

If a repo has no changes, note it and exclude it from further steps.
If ALL repos have no changes, say so and stop.

## Step 2: Branch Safety

For each repo with changes, check if the current branch is protected:

```bash
git [-C <repo>] branch --show-current
```

Then check if that branch has push restrictions:

```bash
gh api repos/{owner}/{repo}/branches/{branch}/protection --jq '.required_status_checks // .required_pull_request_reviews // "protected"' 2>/dev/null
```

- If the API returns protection data → the branch is **protected**
- If the API returns a 404 → the branch is **not protected** (safe to push directly)
- If `gh` is not authenticated or the request fails for another reason → **assume not protected** and continue normally

**If any repo's current branch is protected:**

1. Inform the user which branch is protected
2. Propose creating a temporary branch:
   ```
   ⚠ Branch "main" in <repo> is protected. Direct push will be rejected.

   I'll create a temporary branch, commit there, and offer to create a PR.
   Suggested branch name: <type>/<short-description>  (e.g., feat/add-validation)

   OK? [Y/n]:
   ```
3. Wait for the user to confirm or provide a different branch name
4. Create and checkout the branch:
   ```bash
   git [-C <repo>] checkout -b <branch-name>
   ```
5. At the end of Step 7 (Execute), after all commits, offer to create a PR:
   ```
   Commits pushed to <branch-name>. Create a PR to <protected-branch>? [Y/n]:
   ```
   If yes, create the PR using `gh pr create`.

**Multi-repo:** each repo is evaluated independently. One repo may need a temp branch while another doesn't.

## Step 3: Analyze

Read the actual diffs to understand what changed. Run:
- `git [-C <repo>] diff` for unstaged changes
- `git [-C <repo>] diff --cached` for staged changes
- `git [-C <repo>] diff HEAD -- <file>` for specific files if the overall diff is very large

For untracked files, read their contents with `git [-C <repo>] diff --no-index /dev/null <file>` (limit to first 200 lines per file for large files).

Understand the **purpose** of each change, not just what lines moved.

## Step 4: Group

Apply these heuristic rules to cluster related files into commit groups:

### Grouping Rules (highest priority first)

1. **Feature unit** — a controller + its form request + route change + its test = one commit
2. **Data layer** — model + migration + factory + seeder = one commit
3. **Frontend feature** — page component + child components + types = one commit
4. **Backend code travels with its tests** — never commit source without its test in the same commit
5. **Config/docs** — attribute to the feature they support; standalone config changes are their own commit
6. **Formatting/style** — Pint fixes, Prettier fixes = their own commit (always first or last)
7. **Single-concern files** — if a file doesn't fit any group, it gets its own commit or joins the most related group
8. **Feature context** — `.codex/features/` changes should be grouped with the feature code they describe, not as a separate commit. These are the AI context that the team shares

### Multi-Repo Grouping Rules

When operating in multi-repo mode:

8. **Same-repo affinity** — files within the same repo should be grouped together by default. Do NOT create a single commit spanning multiple repos (git doesn't support that).
9. **Cross-repo features** — when a feature spans repos (e.g., a service in repo A + its Filament UI in repo B), create **separate commits per repo** but with **coordinated messages** that reference each other:
   ```
   Repo A: "Add SchemaSnapshotService for tenant schema bridge"
   Repo B: "Add Filament snapshot action and CLI command"
   ```
10. **Commit ordering across repos** — infrastructure/data repos first, then consuming repos. For example: migrations repo before the app repo that uses those tables.

### Ordering Rules

- Infrastructure/config commits before feature commits
- Backend before frontend when both exist
- Formatting commits first (clean slate) or last (cleanup)
- Migrations before the code that uses them
- In multi-repo mode: dependency repos before dependent repos

## Step 5: Propose

Present a numbered commit plan. In multi-repo mode, label each commit with its repo.

**Single-repo format:**

```
Commit Plan (N commits)
========================

1. [commit message draft]
   Files: file1.php, file2.php, tests/file1Test.php
   Reason: [one-line explanation of grouping]

2. [commit message draft]
   Files: ...
   Reason: ...
```

**Multi-repo format:**

```
Commit Plan (N commits across M repos)
========================================

--- propmgmt-new ---

1. [commit message draft]
   Files: file1.php, file2.php
   Reason: [one-line explanation]

--- tenant-tools ---

2. [commit message draft]
   Files: app/Services/Foo.php, app/Console/Commands/Bar.php
   Reason: [one-line explanation]

3. [commit message draft]
   Files: docs/workflow.md
   Reason: [one-line explanation]
```

### Commit Message Rules

- **Imperative mood**, present tense: "Add", "Fix", "Update", "Refactor" — match the repo style
- Under 72 characters for the subject line
- If the commit needs explanation, add a blank line then a body paragraph

## Step 6: Interactive Approval

Ask the user to review the plan using `AskUserQuestion`. Offer these options:

- **Approve all** — execute the plan as proposed
- **Edit** — let user specify changes (rename messages, move files between commits, merge/split groups)
- **Drop commits** — remove specific commits from the plan (files become unstaged)
- **Reorder** — change commit sequence

If the user picks Edit, ask follow-up questions until the plan is finalized, then confirm once more before executing.

If `$ARGUMENTS` contains `dry-run`, show the plan and stop here — do not execute.

## Step 7: Execute

For each commit in the approved plan:

1. Stage only the specific files for this commit (use `-C` for multi-repo):
   ```bash
   git -C <repo_root> add file1.php file2.php tests/file1Test.php
   ```
2. Create the commit using a HEREDOC (never `-m "..."` for multi-line):
   ```bash
   git -C <repo_root> commit -m "$(cat <<'EOF'
   Add feature X with tests
   EOF
   )"
   ```
3. Verify with `git -C <repo_root> log --oneline -1` after each commit.

After all commits, show the final result per repo:
```bash
git -C <repo_root> log --oneline -N
```

In multi-repo mode, label each verification block with the repo name.

### Push and PR (for protected branch workflows)

If Step 2 created a temporary branch for any repo:

1. Push the branch:
   ```bash
   git [-C <repo>] push -u origin <branch-name>
   ```
2. Ask the user if they want to create a PR:
   ```
   Commits pushed to <branch-name>. Create a PR to <protected-branch>? [Y/n]:
   ```
3. If yes, create the PR:
   ```bash
   gh pr create --base <protected-branch> --head <branch-name> --title "<commit-message-or-summary>" --body "$(cat <<'EOF'
   ## Summary
   <bullet points from commit messages>
   EOF
   )"
   ```
4. Show the PR URL to the user.

If no temp branch was created (branch was not protected), do **not** push automatically — the user may want to review first.

## Safety Rules

- **NEVER** commit `.env`, credentials, secrets, or gitignored files — warn the user if these appear in the diff
- **NEVER** amend a previous commit unless the user explicitly says "amend"
- **NEVER** use `--no-verify` to skip pre-commit hooks
- **NEVER** force push or run destructive git commands
- **NEVER** edit, write, or modify any source files — you are a commit tool, not an editor
- If a pre-commit hook fails, report the error and let the user decide how to fix it
- If `$ARGUMENTS` contains a path filter, only include files matching that filter
- In multi-repo mode, **NEVER** commit in a repo that has no changes — skip it entirely

## Edge Cases

- **Already staged files**: respect existing staging — include them in the first relevant commit group
- **Mixed staged + unstaged in same file**: warn the user and ask how to handle (stage all, or commit only staged portion)
- **Binary files**: include them in commits but note them as binary in the plan
- **Submodule changes**: flag and ask before including
- **Multi-repo with no additional dirs**: if `all` is specified but no additional repos are detected, fall back to single-repo mode after asking the user
- **Repo on different branch**: in multi-repo mode, show the current branch for each repo in the inventory output so the user can verify they're committing to the right branches
