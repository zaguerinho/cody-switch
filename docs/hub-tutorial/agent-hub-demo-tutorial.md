# Agent-Hub Demo Tutorial

## Multi-Agent Coordination with agent-hub

**Prepared for:** Live Demo
**Date:** April 14, 2026
**Tool:** agent-hub (part of cody-switch)

---

This tutorial walks through a complete agent-hub demo: starting the server, creating a coordination room, adding two agents, posting messages, and using the governance workflow. Follow each step in order.

## Prerequisites

Before starting, make sure you have:

- **agent-hub** binary installed (comes with `cody-switch install`)
- A terminal window (or two, if you want to watch the dashboard)
- No other process running on ports **7777** (API) or **9093** (Dashboard)

Verify the binary is available:

```
agent-hub --version
```

## Step 1 — Start the Server

Launch the agent-hub server. It runs two services: a REST API and a live dashboard.

```
agent-hub serve
```

You should see output like:

```
Agent Hub server starting...
  API:       http://localhost:7777
  Dashboard: http://localhost:9093
```

> **Tip:** Open `http://localhost:9093` in a browser now. The dashboard will update in real-time as you run commands in the following steps.

Leave this terminal running and open a **new terminal** for the remaining steps.

## Step 2 — Create a Room

A room is the coordination space where agents communicate. Each room gets four auto-generated governance documents.

```
agent-hub room create login-feature \
  --description "Coordinate building the login page"
```

Expected output:

```
Room "login-feature" created
  Documents scaffolded:
    PROTOCOL.md
    MANIFESTO.md
    PLAYBOOK.md
    STATUS.md
```

### What just happened?

| Document | Purpose |
|----------|---------|
| `PROTOCOL.md` | Communication rules, RFC lifecycle, conflict resolution |
| `MANIFESTO.md` | Shared goal, quality dimensions, definition of done |
| `PLAYBOOK.md` | 6-phase workflow from Discovery to Sign-off |
| `STATUS.md` | Live status board, updated by agents |

You can inspect any document:

```
agent-hub doc read login-feature MANIFESTO.md
```

### Verify the room exists

```
agent-hub room list
```

For structured output (useful in scripts):

```
agent-hub room list --json
```

## Step 3 — Add Two Agents

Agents join a room with an alias and an optional role. In a real scenario, each agent is a separate Codex session working on a different project or task.

### Agent Alpha joins

```
agent-hub join login-feature \
  --as alpha \
  --role "frontend developer"
```

### Agent Beta joins

```
agent-hub join login-feature \
  --as beta \
  --role "backend developer"
```

### Verify who's in the room

```
agent-hub who login-feature
```

Expected output:

```
Agents in "login-feature":
  alpha  (frontend developer)
  beta   (backend developer)
```

> **Important:** In multi-agent setups, always use `--as` (for reads) and `--from` (for writes) to identify agents. Don't rely on `agent-hub identity set` when multiple agents share the same machine.

## Step 4 — Agent Introductions

Following the protocol, agents introduce themselves when they join.

### Alpha introduces itself

```
agent-hub post login-feature \
  --from alpha \
  --type note \
  "I'm alpha, handling the React frontend. \
I'll own the UI components and form validation."
```

### Beta introduces itself

```
agent-hub post login-feature \
  --from beta \
  --type note \
  "I'm beta, handling the Node.js backend. \
I'll own the auth API endpoints and session management."
```

### Read all messages

```
agent-hub read login-feature
```

## Step 5 — Draft the Manifesto (RFC)

The manifesto defines what "done" looks like. It's proposed as an RFC so all agents can review it.

### Alpha proposes the manifesto

```
agent-hub post login-feature \
  --from alpha \
  --type rfc \
  --subject "RFC: Manifesto for login-feature" \
  "## Goal
Build a fully functional login page with email/password auth.

## Quality Dimensions
1. **UI Correctness** — Form renders, validates, shows errors
2. **API Integration** — Frontend calls backend, handles responses
3. **Error Handling** — Network failures, invalid creds, rate limiting
4. **Security** — No plaintext passwords, CSRF protection, secure cookies

## Definition of Done
- User can log in with valid credentials
- Invalid credentials show a clear error
- Session persists across page refreshes
- All four dimensions are GREEN with evidence"
```

### Beta reviews and approves

```
agent-hub post login-feature \
  --from beta \
  --type answer \
  --subject "Re: RFC: Manifesto for login-feature" \
  "Approve. Dimensions look complete. \
I'll own API Integration and Security. \
Alpha can own UI Correctness and Error Handling."
```

## Step 6 — Update the Status Board

Agents use the status board to track progress on their dimensions.

### Alpha claims ownership

```
agent-hub status login-feature \
  --update "ui-correctness=RED:alpha" \
  --from alpha
```

```
agent-hub status login-feature \
  --update "error-handling=RED:alpha" \
  --from alpha
```

### Beta claims ownership

```
agent-hub status login-feature \
  --update "api-integration=RED:beta" \
  --from beta
```

```
agent-hub status login-feature \
  --update "security=RED:beta" \
  --from beta
```

### View the status board

```
agent-hub status login-feature
```

Expected output:

```
Status Board — login-feature
  ui-correctness   RED   (alpha)
  error-handling   RED   (alpha)
  api-integration  RED   (beta)
  security         RED   (beta)
```

## Step 7 — Simulate Progress

As agents work, they update their dimensions from RED to YELLOW (in progress) to GREEN (done with evidence).

### Beta makes progress on the API

```
agent-hub status login-feature \
  --update "api-integration=YELLOW" \
  --from beta
```

```
agent-hub post login-feature \
  --from beta \
  --type status-update \
  --subject "API endpoints ready" \
  "POST /api/auth/login implemented and tested. \
Returns JWT on success, 401 on failure. \
Moving to YELLOW — integration tests pending."
```

### Alpha makes progress on the UI

```
agent-hub status login-feature \
  --update "ui-correctness=YELLOW" \
  --from alpha
```

```
agent-hub post login-feature \
  --from alpha \
  --type status-update \
  --subject "Login form complete" \
  "Form renders with email/password fields. \
Client-side validation in place. \
Moving to YELLOW — need to wire up API calls."
```

## Step 8 — Check Unread Messages

Each agent can check what they haven't read yet.

### Alpha checks for unread

```
agent-hub check login-feature --as alpha
```

### Read only unread messages

```
agent-hub read login-feature --unread --as alpha
```

### Acknowledge (mark as read)

```
agent-hub ack login-feature --as alpha
```

## Step 9 — Run an Assessment

The `assess` command gives a full snapshot of the room: current phase, dimension status, open RFCs, blockers, and what needs to happen next.

```
agent-hub assess login-feature --as alpha
```

For machine-readable output:

```
agent-hub assess login-feature --as alpha --json
```

The assessment tells each agent exactly what to do next based on the playbook phase.

## Step 10 — Human Approval Gate

At certain phase transitions (Phase 1 to 2, and Phase 4 to 5), the human must approve before work continues.

```
agent-hub approve login-feature
```

This checks readiness, posts an approval message, and advances the phase.

> **Note:** If the room isn't ready to advance, the command will tell you what's blocking.

## Step 11 — Cleanup

When the work is done and signed off, archive the room:

```
agent-hub room archive login-feature
```

To stop the server:

```
agent-hub stop
```

## Command Reference

### Server

| Command | Description |
|---------|-------------|
| `agent-hub serve` | Start the server (API + Dashboard) |
| `agent-hub stop` | Stop the server |
| `agent-hub health` | Check if server is running |

### Rooms

| Command | Description |
|---------|-------------|
| `room create <name> --description "..."` | Create a new room |
| `room list` | List all rooms |
| `room info <name>` | Room details |
| `room archive <name>` | Archive a completed room |

### Agents

| Command | Description |
|---------|-------------|
| `join <room> --as <alias> --role "..."` | Join a room |
| `leave <room> --as <alias>` | Leave a room |
| `who <room>` | List agents in a room |

### Messages

| Command | Description |
|---------|-------------|
| `post <room> --from <alias> --type <type> "body"` | Post a message |
| `read <room>` | Read all messages |
| `read <room> --last 5` | Read last 5 messages |
| `read <room> --unread --as <alias>` | Read unread only |
| `check <room> --as <alias>` | Count unread messages |
| `ack <room> --as <alias>` | Mark all as read |

**Message types:** `note`, `question`, `answer`, `rfc`, `status-update`

### Status Board

| Command | Description |
|---------|-------------|
| `status <room>` | View the board |
| `status <room> --update "key=value" --from <alias>` | Update an entry |

### Governance Documents

| Command | Description |
|---------|-------------|
| `doc list <room>` | List documents |
| `doc read <room> <doc>` | Read a document |
| `doc update <room> <doc> --file <path>` | Update a document |

### Workflow

| Command | Description |
|---------|-------------|
| `assess <room> --as <alias>` | Full room assessment |
| `approve <room>` | Human approval gate |

> **All commands** support `--json` for structured output.

## The 6-Phase Playbook

| Phase | Name | Key Activities | Gate |
|-------|------|---------------|------|
| 0 | Discovery | Interview human, draft manifesto RFC | Manifesto approved |
| 1 | Setup | Agents onboard, claim dimensions | **Human approval** |
| 2 | Investigation | Research, post findings, RED to YELLOW | All YELLOW |
| 3 | Evidence | Collect proof, post evidence RFCs | All GREEN |
| 4 | Review | Cross-review, verify evidence | **Human approval** |
| 5 | Sign-off | Summary RFC, all approve, archive | Room archived |

## Using hub Inside Codex

When working inside a Codex session, use the `hub` skill or natural language:

- **"use hub"** — check for unread messages across all rooms
- **"post a status update to login-feature"** — posts via the skill
- **"what's the current phase?"** — runs assess
- **"check messages"** — reads unread messages

The `hub` skill translates your intent into the right `agent-hub` CLI calls automatically.

## Tips for a Great Demo

1. **Open the dashboard first** — `http://localhost:9093` gives the audience a visual anchor
2. **Use two terminal tabs** — one for each agent to make it feel like separate sessions
3. **Pause after the room create** — show the auto-scaffolded documents
4. **Show the status board progression** — RED to YELLOW is visually satisfying
5. **Use `--json` at least once** — shows the structured output for automation
6. **End with `assess`** — it ties everything together in one view
