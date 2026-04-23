---
allowed-tools:
  - Bash(git *)
  - AskUserQuestion
  - Read
---

# Checkpoint

Create, list, or diff named code snapshots during development. Useful for saving known-good states during long features.

## Arguments

`$ARGUMENTS` may contain:
- `save <name>` — create a named checkpoint (default if just a name is given)
- `list` — list all checkpoints for the current branch
- `diff <name>` — diff current working tree against a named checkpoint
- `restore <name>` — restore working tree to a checkpoint (asks for confirmation)
- No arguments — show usage

## How Checkpoints Work

Checkpoints are lightweight git tags prefixed with `checkpoint/` and the current branch name. They're local-only (never pushed).

Tag format: `checkpoint/<branch>/<name>`

## Commands

### save (default)

If `$ARGUMENTS` is just a name (no subcommand), treat it as `save <name>`.

1. Check for uncommitted changes:
   ```bash
   git status --short
   ```
2. If there are uncommitted changes, create a temporary commit:
   ```bash
   git stash
   git stash apply
   ```
   Then warn the user: "Note: You have uncommitted changes. The checkpoint captures the last commit only. Consider committing first."

3. Create the checkpoint tag:
   ```bash
   git tag "checkpoint/$(git branch --show-current)/<name>"
   ```

4. Confirm:
   ```
   Checkpoint '<name>' created at $(git log --oneline -1)
   ```

### list

```bash
git tag -l "checkpoint/$(git branch --show-current)/*" --sort=-creatordate
```

Show each checkpoint with its commit message:
```bash
git log --oneline -1 <tag>
```

Format as a table:
```
Checkpoints (branch: feature-x)
================================
1. before-refactor  abc1234  Add venue menu CRUD
2. pre-migration    def5678  Add menu items table
```

If no checkpoints exist, say so.

### diff

```bash
git diff "checkpoint/$(git branch --show-current)/<name>"..HEAD --stat
```

Then show the full diff:
```bash
git diff "checkpoint/$(git branch --show-current)/<name>"..HEAD
```

If the checkpoint doesn't exist, list available checkpoints.

### restore

1. Confirm with `AskUserQuestion`:
   > "This will reset your working tree to checkpoint '<name>' (commit abc1234). Uncommitted changes will be lost. Proceed?"
   - **Yes** — proceed
   - **No** — abort

2. If confirmed:
   ```bash
   git checkout "checkpoint/$(git branch --show-current)/<name>" -- .
   ```

3. Show what changed:
   ```bash
   git status --short
   ```

## Safety Rules

- **NEVER** delete checkpoint tags without explicit user request
- **NEVER** push checkpoint tags to remote
- **NEVER** restore without confirmation
- Checkpoint names must be alphanumeric with hyphens (no spaces or special characters)
- If a checkpoint name already exists, ask before overwriting
