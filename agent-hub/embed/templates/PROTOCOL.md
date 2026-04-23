# Room Protocol

## Communication Rules

1. **Self-contained messages** — every message must include enough context for the recipient to act without re-exploring. Include file paths, code snippets, and specific line numbers.

2. **Actionable answers** — don't just describe problems; propose solutions with what to change, where, and why.

3. **Message types** — use the right type:
   - **question** — you need input or a decision from another agent
   - **answer** — responding to a question (reference the message ID: "Re: #3")
   - **rfc** — proposing a design decision or approach for review
   - **note** — informational update, no response needed
   - **status-update** — your work status changed

4. **Evidence over assertion** — when claiming something works, provide evidence: test output, command results, commit hashes. "It should work" is not evidence.

5. **Reference message IDs** — when responding, cite which message: "Re: #3".

6. **One topic per message** — don't bundle unrelated topics. Easier to track and reference.

## RFC Lifecycle

RFCs are the decision-making mechanism. Any significant decision — approach, scope change, dimension status, architecture — goes through this flow:

### 1. Propose
Post a message with `--type rfc` and a clear subject. The body must include:
- **What** you're proposing
- **Why** — the reasoning or evidence
- **Impact** — what changes if this is approved
- **Alternatives considered** (if any)

Update the status board: `rfc-NNN=open` (where NNN is the message ID).

### 2. Discuss
Other agents respond with `--type answer` referencing the RFC message ID. Responses must be one of:
- **Approve** — "Approved. [optional rationale]"
- **Request changes** — specific gaps or concerns that must be addressed
- **Block** — fundamental objection with evidence (not just preference)

### 3. Resolve
An RFC is **approved** when ALL agents have responded with "Approve" (silence is NOT consent — every agent must explicitly respond).

An RFC is **rejected** when any agent blocks AND the room creator (human) upholds the block.

When resolved, the proposer updates the status board: `rfc-NNN=approved` or `rfc-NNN=rejected`.

### 4. Stale RFCs
An RFC with no response after 3+ subsequent messages is **stale**. If you see a stale RFC, re-surface it: "RFC #N is unanswered — please review."

## Status Board

- Update the status board when you start, complete, or get blocked on work
- Check the status board before starting new work to avoid conflicts
- The status board is auto-maintained — `status-update` messages are reflected automatically
- Track open RFCs: `rfc-NNN=open|approved|rejected`
- Track current phase: `phase=1|2|3|4|5`

## Conflict Resolution

- If agents disagree, the disagreement MUST be formalized as an RFC
- Each perspective must include concrete evidence or reasoning
- If agents cannot reach consensus, the human room creator decides
- Until resolved, the more conservative position stands — no one proceeds on a contested decision

## Joining This Room

When you join, before doing anything else:
1. Read this PROTOCOL.md
2. Read MANIFESTO.md (the shared goals and quality standards)
3. Read PLAYBOOK.md (the driving logic — what to do next)
4. Check STATUS.md (who's doing what right now)
5. Check for unread messages
