## Workflow Orchestration

### 1. Plan Mode Default
- Enter plan mode for ANY non-trivial task (3+ steps or architectural decisions)
- If something goes sideways, STOP and re-plan immediately
- Use plan mode for verification steps, not just building
- Write detailed specs upfront to reduce ambiguity

### 2. Parallel Work Strategy
Pick the right level based on task complexity:

| Complexity | Approach | When |
|---|---|---|
| Simple | Solo | Single-file edits, quick fixes |
| Medium | Subagents | Research, focused tasks where only the result matters |
| Complex | Agent Teams | Cross-cutting work where workers need to discuss and coordinate |

**Subagents (default for parallelism):**
- Use liberally to keep main context window clean
- Offload research, exploration, and parallel analysis
- One task per subagent for focused execution
- **Model selection for subagents:** Use Explore/Plan agent types for research (lighter weight). Reserve general-purpose agents for tasks that need write access. Don't spawn Opus-powered agents for simple file searches — use Glob/Grep directly.

**Agent Teams (only when workers need to talk to each other):**
- Cross-layer changes spanning frontend, backend, and tests
- Debugging with competing hypotheses (investigators disprove each other)
- Parallel code review: security + performance + test coverage simultaneously
- Each teammate must own separate files -- never two agents on the same file
- Use delegate mode so the lead coordinates, not implements
- Aim for 5-6 tasks per teammate; require plan approval for risky changes
- **Agent handoffs:** When chaining agents sequentially, pass a structured handoff (key findings, modified files, open questions) so the next agent starts with full context

### 3. Self-Improvement Loop
- At the start of every session, read `tasks/lessons.md`, `.codex/features/lessons-global.md`, and `~/.codex/lessons-global.md` (if they exist)
- After ANY correction from the user: update `tasks/lessons.md` with the pattern
- **Proactive promotion**: Immediately after writing a lesson to `tasks/lessons.md`, assess and prompt:
  1. Analyze the lesson content
  2. Recommend a destination with a brief reason:
     - **Project** (`.codex/features/lessons-global.md`): codebase-specific patterns, architecture decisions, project conventions
     - **User** (`~/.codex/lessons-global.md`): universal patterns that apply across all projects
     - **Skip**: too feature-specific or already outdated
  3. Ask: "Promote this lesson to {recommended level}? [Y/n/other]"
  4. **Wait for explicit confirmation** before writing — never assume a destination
- This should happen inline, right after the lesson is captured — don't wait for the user to run `cody-switch promote-lesson`
- `cody-switch promote-lesson` still works for bulk/manual promotion of older lessons
- Shortcut to project: `cody-switch promote-lesson --project`
- Shortcut to user: `cody-switch promote-lesson --user`
- Bulk audit: `cody-switch promote-audit`
- Write rules for yourself that prevent the same mistake
- Ruthlessly iterate on these lessons until mistake rate drops

### 4. Context Window Hygiene
- **Compact at workflow boundaries** — after planning is approved, after a debugging session, between major implementation phases. Don't let automatic compaction fire mid-task and lose important context.
- **Offload large results to subagents** — if a search or exploration will return many results, use a subagent so the main context stays clean.
- **After 40+ tool calls in a session**, consider whether a fresh Codex session would help focus.

### 5. Verification Before Done
- Never mark a task complete without proving it works
- Diff behavior between main and your changes when relevant
- Ask yourself: "Would a staff engineer approve this?"
- Run tests, check logs, demonstrate correctness
- For critical changes, consider parallel review (spawn reviewers for security, perf, tests) using the defaults in section 7.

### 6. Autonomous Bug Fixing
- When given a bug report: just fix it. Don't ask for hand-holding
- Point at logs, errors, failing tests then resolve them
- Zero context switching required from the user
- Go fix failing CI tests without being told how
- For elusive bugs: spawn competing investigators to disprove each other's theories

### 7. Agent Orchestration Defaults
When you delegate verification, review, or analysis to subagents, apply these defaults:

- **Mandate execution, not just reading.** Reviewer agents read files and reason. They do not automatically run the suite, build the binary, or invoke the command. For changes with tests or a CLI, build it, run scoped tests, and smoke-invoke at least one entry point.
- **Tie adversarial checks to the diff.** Turn on specific review prompts based on changed files and claims, such as middleware, internal endpoints, passthrough behavior, request headers, or new input flags.
- **Use neutral problem statements.** State what changed, what files it touches, and what it claims. Avoid steering reviewers with confidence labels.

A fresh reviewer finding valid issues after your own reviewers approved means the orchestration needs adjustment.

## Task Management

> **Note:** The `tasks/` folder (`todo.md`, `lessons.md`) is auto-managed by `cody-switch`.
> When you switch features, your current `tasks/` is saved and the target feature's is restored.
> A fresh scaffold is created automatically for new features.

1. **Plan First**: Write plan to `tasks/todo.md` with checkable items
2. **Verify Plan**: Check in before starting implementation
3. **Track Progress**: Mark items complete as you go
4. **Explain Changes**: High-level summary at each step
5. **Document Results**: Add review section to `tasks/todo.md`
6. **Capture Lessons**: Update `tasks/lessons.md` after corrections

## Core Principles

- **Simplicity First**: Make every change as simple as possible. Impact minimal code.
- **No Laziness**: Find root causes. No temporary fixes. Senior developer standards.
- **Minimal Impact**: Changes should only touch what's necessary. Avoid introducing bugs.
- **Right Tool, Right Scale**: Don't over-orchestrate trivial tasks. Don't under-power complex ones.

## Coding Standards

- **NEVER use fallbacks that silently mask errors** — fallbacks hide real problems and make debugging impossible. If data is missing or invalid, FAIL LOUDLY with a clear error message. Don't use "generic" or "default" values as safety nets.
