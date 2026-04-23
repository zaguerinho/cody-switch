# Room Manifesto

## Goal

<!-- Describe the shared objective for this room -->

## Principles

1. **Fail loudly** — never silently mask errors or use fallback defaults that hide problems
2. **Evidence-driven** — every claim of "done" must include reproducible proof
3. **Minimal scope** — solve the problem at hand, don't over-engineer
4. **Clear ownership** — every task has exactly one responsible agent
5. **No unilateral decisions** — architectural choices require an RFC and at least one review

## Quality Dimensions

Track progress across these dimensions. Each must reach GREEN before the work is complete.

| # | Dimension | Status | Owner | Evidence |
|---|-----------|--------|-------|----------|
| 1 | Core functionality | RED | — | — |
| 2 | Error handling | RED | — | — |
| 3 | Testing | RED | — | — |
| 4 | Security | RED | — | — |
| 5 | Documentation | RED | — | — |

### Status Definitions

- **RED** — not started or significant gaps remain
- **YELLOW** — in progress, partial evidence collected
- **GREEN** — complete with full, reproducible evidence

### Evidence Rules

- Every GREEN must include: branch/commit, environment, exact command, output
- Evidence decays when code changes — re-prove after significant modifications
- Code inspection alone is not evidence for runtime behavior

## Definition of Done

- [ ] All quality dimensions are GREEN
- [ ] All agents have reviewed and approved
- [ ] No open questions or unresolved RFCs
- [ ] Status board shows no blockers
