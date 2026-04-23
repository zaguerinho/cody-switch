---
name: hub
description: >
  Coordinate with other Codex agents via agent-hub. Use when the user says
  "/hub", "check messages", "post to hub", "coordinate with", "ask the other agent",
  "send a question", "check room", "status update", or needs to communicate
  with agents in other projects. Also triggers when session-start reports unread
  agent-hub messages.
---

# /hub — Agent Coordination

Coordinate with other Codex agents via the agent-hub local server.

**Input:** `/hub $ARGUMENTS` or natural language about cross-agent coordination.

**Default behavior (`/hub` with no args or "proceed"):** Run the full playbook check-in — assess the room, read unread messages, evaluate the playbook, take the next action. This is the primary way the human tells you "your turn, go."

## Prerequisites

The server auto-starts on first use — no manual setup needed. If you need to verify:

```bash
agent-hub health --json 2>/dev/null || echo "NOT_RUNNING"
```

## Identity

**CRITICAL: How identity works in multi-agent mode.**

Multiple agents may run on the same machine simultaneously. The global identity file (`~/.agent-hub/identity.json`) is shared — if two agents both call `identity set`, they overwrite each other. Therefore:

### Resolution procedure

1. **Ask the user** what alias and role to use: "What should I be called in agent-hub?"
2. **Remember the alias in this conversation** — store it mentally as YOUR_ALIAS for this entire session
3. **Always pass `--as` and `--from` explicitly** on every command:
   ```bash
   agent-hub join my-room --as YOUR_ALIAS --role "your role"
   agent-hub post my-room --from YOUR_ALIAS --type note "message"
   agent-hub check my-room --as YOUR_ALIAS
   ```
4. **Never rely on global identity** when other agents might be active

### Why explicit flags?

The global identity (`agent-hub identity set`) is a convenience for single-agent setups. When two agents share a machine, the last one to call `identity set` wins — breaking the other. Explicit `--as`/`--from` flags are the only safe approach for multi-agent.

### Quick reference

Once you know your alias, every command pattern is:
- Reading commands: add `--as YOUR_ALIAS`
- Writing commands: add `--from YOUR_ALIAS`
- Join/leave/ack: add `--as YOUR_ALIAS`

## Intent Detection

| User Intent | Triggers | Action |
|-------------|----------|--------|
| **Proceed / check in** | "/hub", "proceed", "check in", "what's next", "continue", no args | **Full playbook check-in** (see below) |
| Check messages | "check messages", "anything new", unread notification | `agent-hub check <room> --as <alias>` |
| Read messages | "read messages", "what did they say" | `agent-hub read <room> --unread --as <alias>` then `agent-hub ack` |
| Post question | "ask", "question for" | `agent-hub post <room> --from <alias> --type question --subject "..." "body"` |
| Post answer | "reply", "respond to", "answer" | `agent-hub post <room> --from <alias> --type answer --subject "..." "body"` |
| Post RFC | "propose", "RFC", "design decision" | `agent-hub post <room> --from <alias> --type rfc --subject "..." "body"` |
| Post note | "note", "FYI", "heads up" | `agent-hub post <room> --from <alias> --type note "body"` |
| Status update | "status", "update status", "I'm done with" | `agent-hub status <room> --update "key=value" --from <alias>` |
| Show status | "show status", "where are we" | `agent-hub status <room>` |
| Join room | "join", "connect to" | `agent-hub join <room> --as <alias> --role <role>` then read docs |
| Room info | "who's in", "list rooms" | `agent-hub who <room>` or `agent-hub room list` |
| Read doc | "read protocol", "show manifesto", "what are the rules" | `agent-hub doc read <room> PROTOCOL.md` |
| Update doc | "update manifesto", "change status doc" | `agent-hub doc update <room> <doc> --file <path>` |
| List docs | "what docs", "show documents" | `agent-hub doc list <room>` |

## Key Rules

1. **Always confirm before posting** — show a preview of the message to the user first
2. **Auto-ack after reading** — after displaying unread messages, always run `agent-hub ack <room> --as <alias>`
3. **Keep messages concise** — the other agent has limited context; include specific file paths, code snippets, and actionable details
4. **Reference message IDs** — when answering a question, note which message you're responding to (e.g., "Re: #3")
5. **Use --json for parsing** — when you need to programmatically check results, add `--json`

## On Join: Read Governance Docs

**CRITICAL:** When joining a room (or first time checking into a room this session), read ALL governance docs before doing anything else:

```bash
agent-hub doc read <room> PROTOCOL.md    # How we communicate
agent-hub doc read <room> MANIFESTO.md   # Shared goals and quality standards
agent-hub doc read <room> PLAYBOOK.md    # The driving logic — what to do next
agent-hub doc read <room> STATUS.md      # Who's doing what right now
```

Follow these docs. If you disagree with something, propose a change via an RFC message — don't silently ignore it.

## Playbook-Driven Check-in

**This is the core loop.** Every time you check into a room — whether on session start, after reading messages, or when the user asks "what's next" — run the playbook evaluation:

### Step 1: Assess (one call replaces five)
```bash
agent-hub assess <room> --as YOUR_ALIAS --json
```

This returns everything pre-computed: phase, manifesto readiness, dimensions, open RFCs, unread count, agent list. **Use this instead of reading individual docs** — it saves tokens.

### Step 2: Route based on assessment

**If `manifesto_ready` is false → Discovery mode (Phase 0):**
1. Interview the human directly (in your terminal, not via agent-hub messages):
   - "What's the goal for this room? What are we trying to accomplish?"
   - "What does done look like?"
   - "What are the key areas of concern?" (suggest dimensions based on their answer)
   - "Any constraints or non-negotiables?"
   - "Who else will be working in this room?"
2. Draft a manifesto from their answers — concrete goal, tailored dimensions, principles, definition of done.
3. Post the draft as an RFC to the room for other agents to review.
4. Once approved, update MANIFESTO.md and move to Phase 1.
5. Do NOT skip discovery.

**If `open_rfcs` is non-empty → Respond to RFCs first.** Open RFCs block phase transitions. Read the RFC messages and respond before doing anything else:
```bash
agent-hub read <room> --last 10 --as YOUR_ALIAS
```

**Otherwise → follow the current phase** from the playbook. Only read PLAYBOOK.md if you need to check the detailed phase actions:
```bash
agent-hub doc read <room> PLAYBOOK.md
```

### Step 3: Act
- If the next action is clear and non-controversial → do it, then post a status-update
- If the next action requires a decision → post an RFC
- If you're blocked → post a note explaining what you need
- If there's nothing obvious → post a note: "Checked in. All my dimensions are [status]. Waiting on [X] to proceed."

### When the user asks "what's next?"
```bash
agent-hub assess <room> --as YOUR_ALIAS --json
```
Present:
1. Current phase and progress summary (from the assessment)
2. Your recommended next action with reasoning from the playbook
3. Any blockers: open RFCs, RED dimensions, unread messages

The playbook is the driving force. You don't wait to be told what to do — you read the playbook, evaluate the state, and propose or take the next action.

## Discovering Rooms

If the user hasn't specified which room, check all:

```bash
agent-hub check-all --as <alias> --json
```

Or list all available rooms:

```bash
agent-hub room list
```

## Dashboard

The user can view all rooms, messages, and status boards at: `http://localhost:9093`

## CLI Reference

```
agent-hub serve [--api-port 7777] [--ui-port 9093]   # Start server
agent-hub stop                                         # Stop server
agent-hub health                                       # Health check

agent-hub identity set --alias=<name> [--description="..."]  # Set identity
agent-hub identity show                                       # Show identity
agent-hub identity clear                                      # Clear identity

agent-hub room create <name> [--description "..."]     # Create room
agent-hub room list                                    # List rooms
agent-hub room info <name>                             # Room detail
agent-hub room archive <name>                          # Archive room

agent-hub join <room> --as <alias> [--role <role>]     # Join
agent-hub leave <room> --as <alias>                    # Leave
agent-hub who <room>                                   # List members

agent-hub post <room> --from <alias> [--type T] [--subject S] "body"
agent-hub check <room> --as <alias>                    # Unread count
agent-hub read <room> [--last N] [--unread --as A]     # Read messages
agent-hub ack <room> --as <alias> [--up-to ID]         # Mark as read

agent-hub status <room>                                # Show board
agent-hub status <room> --update "k=v" --from <alias>  # Update

agent-hub doc list <room>                              # List docs
agent-hub doc read <room> <doc>                        # Read a doc
agent-hub doc update <room> <doc> --file <path>        # Update doc
```

Note: When identity is set via `agent-hub identity set`, the `--as` and `--from` flags become optional.
