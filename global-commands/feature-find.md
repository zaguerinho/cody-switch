---
allowed-tools:
  - Read
  - Glob
  - Grep
  - Bash(cat *)
  - Bash(ls *)
  - AskUserQuestion
---

# Feature Find

You are a feature search assistant for `cody-switch`. Your job is to scan all features' content, match them against a user's description, and recommend which feature to work with.

## Step 1: Get Search Query

The user's search description is: `$ARGUMENTS`

If `$ARGUMENTS` is empty, use `AskUserQuestion` to ask:
- "What are you looking for? Describe the feature, topic, or functionality."

## Step 2: Discover All Features

Find the project root (look for `.codex-current-feature` or `.git`), then discover every feature:

1. **Active feature**: Read `.codex-current-feature` to get the name. Its content lives at the project root:
   - `AGENTS.md`
   - `tasks/todo.md`
   - `tasks/lessons.md`
2. **Stored features**: List directories under `.codex/features/*/` (exclude `archived/` and `lessons-global.md`). For each:
   - `.codex/features/{name}/AGENTS.md`
   - `.codex/features/{name}/tasks/todo.md`
   - `.codex/features/{name}/tasks/lessons.md`
3. **Archived features**: List directories under `.codex/features/archived/*/`. For each:
   - `.codex/features/archived/{name}/AGENTS.md`
   - `.codex/features/archived/{name}/tasks/todo.md`
   - `.codex/features/archived/{name}/tasks/lessons.md`
4. **Documentation**: Check for `docs/{name}/` for each feature (active, stored, and archived under `docs/archived/{name}/`). Note which files exist.

Read all available files for each feature. If a file is missing or empty, note it but continue — some features may only have partial artifacts.

## Step 3: Analyze and Match

For each feature, consider ALL of its content (AGENTS.md, tasks, docs file list) and determine how well it matches the search query:

- **Strong match**: The feature directly addresses the described topic. Its AGENTS.md or tasks explicitly mention the functionality.
- **Partial match**: The feature touches on the topic but isn't primarily about it — e.g., a shared-types feature that includes auth types when searching for "authentication".
- **No match**: The feature is unrelated.

## Step 4: Detect Duplicates

After matching, look for potential duplicates or overlapping features:
- Features with similar names (e.g., `auth` and `authentication`, `api-v2` and `api-refactor`)
- Features whose AGENTS.md describes overlapping scope
- Multiple features that match the same search query strongly

## Step 5: Report Results

Present results grouped by match strength. For each matching feature:

```
## Search Results for: "{query}"

### Strong Matches

**{name}** [{status}]
  Summary: {one-line summary from AGENTS.md or tasks}
  Artifacts: AGENTS.md, tasks/todo.md, docs/ (N files)
  Match reason: {why this matches the query}
  → cody-switch {name}

### Partial Matches

**{name}** [{status}]
  Summary: {one-line summary}
  Match reason: {why this partially matches}
  → cody-switch {name}

### No Matches

{count} features did not match: {comma-separated names}
```

Where `{status}` is one of:
- `active` — currently active feature
- `stored` — available to switch to
- `archived` — hidden, needs `cody-switch unarchive {name}` first
- `no AGENTS.md` — feature has artifacts but no AGENTS.md (unhealthy)

If there are potential duplicates, add a section:

```
### Possible Duplicates

⚠ **{name1}** and **{name2}** may overlap:
  {brief explanation of the overlap}
  Consider: `cody-switch merge {name1} into {name2}` or review both with `cody-switch peek {name}`
```

If no features match at all:

```
No features match "{query}".

You can create a new one:
  cody-switch blank {suggested-name}
  cody-switch blank {suggested-name} --branch
```

## Rules

- **Read-only**: NEVER modify any files. This is a search/discovery command only.
- **Always read content**: Don't match on feature names alone — read the actual AGENTS.md and tasks to understand what each feature covers.
- **Handle missing files gracefully**: A feature directory might exist with only a `session` file and no AGENTS.md. Report it with `[no AGENTS.md]` status but still check tasks if they exist.
- **Active feature reads from project root**: The active feature's AGENTS.md is at the project root, NOT in `.codex/features/{name}/`.
- **Be concise**: Summaries should be one line. Match reasons should be one sentence.
- **Sort by relevance**: Strong matches first, then partial, then duplicates warning.
