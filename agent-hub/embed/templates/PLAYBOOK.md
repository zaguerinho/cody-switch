# Playbook

> The driving logic for this room. Agents evaluate this on every check-in
> to determine the next action without waiting for human direction.

## Phases

Work progresses through these phases sequentially. Transition rules are explicit — don't skip ahead.

### Phase 0: Discovery
**Goal:** Understand what the human wants. Crystallize the intent into a concrete manifesto.

**Entry:** Room created. MANIFESTO.md still has the default placeholder (`<!-- Describe the shared objective -->`).

**How to detect:** Read MANIFESTO.md. If the Goal section contains `<!--` or is empty, you are in Phase 0.

**Actions (first agent to check in drives this):**

1. **Interview the human.** Ask these questions directly (not via agent-hub messages — talk to the user in your terminal):
   - "What's the goal for this room? What are we trying to accomplish?"
   - "What does done look like? How will we know we succeeded?"
   - "What are the key areas of concern? (e.g., security, performance, testing, docs)"
   - "Any constraints or non-negotiables?"
   - "Who else will be in this room and what are their roles?"

2. **Draft the manifesto.** From the human's answers, write:
   - A concrete Goal section (1-3 sentences)
   - Tailored quality dimensions (replace the defaults with dimensions specific to this work)
   - Specific principles (keep or replace the defaults based on what matters here)
   - A clear definition of done

3. **Post the draft as an RFC.** Post it to the room so other agents can review:
   ```
   agent-hub post ROOM --from ALIAS --type rfc --subject "RFC: Manifesto draft" "FULL DRAFT HERE"
   ```
   Update status board: `rfc-N=open`

4. **Other agents review.** They respond with Approve or Request changes. The human can also weigh in.

5. **Once approved, update the manifesto:**
   ```
   agent-hub doc update ROOM MANIFESTO.md --file /path/to/final.md
   ```

Update status board: `phase=0`, then `phase=1` when manifesto is locked.

**Transition → Phase 1:** MANIFESTO.md goal is filled in (no placeholder) AND the manifesto RFC is approved.

---

### Phase 1: Setup
**Goal:** All agents onboarded, dimensions assigned, scope agreed. Human gives the GO.

**Entry:** Manifesto locked (Phase 0 complete).

**Actions:**
- Each agent reads PROTOCOL.md, MANIFESTO.md, PLAYBOOK.md
- Each agent posts a `note` introducing themselves and their role
- Assign dimension owners in MANIFESTO.md (update the Owner column)
- If any agent disagrees with the manifesto, post an `rfc` to propose changes
- Update status board: `phase=1`

**RFC flow in this phase:**
1. Agent posts `rfc` for any manifesto adjustments
2. All agents respond with Approve / Request changes (per PROTOCOL.md RFC rules)
3. Once approved, update MANIFESTO.md via `agent-hub doc update`

**Transition → Phase 2:** ALL of these must be true:
- All agents have joined and introduced themselves
- All dimensions have owners
- No open RFCs (`rfc-*=open` on the status board)
- **Human gives the GO** — the room creator must explicitly approve before work begins

Update status board: `phase=2`

---

### Phase 2: Investigation
**Goal:** Every RED dimension reaches at least YELLOW with partial findings.

**Actions:**
- Each dimension owner investigates their assigned areas
- Post `question` messages when you need input from another agent
- Post `status-update` when a dimension moves from RED → YELLOW
- If your investigation reveals the scope needs adjustment, post an `rfc` — don't just change things
- If blocked, post a `note` explaining the blocker and what would unblock you

**Transition → Phase 3:** ALL of these must be true:
- All dimensions are YELLOW or GREEN
- No unanswered questions older than 2 messages
- No open RFCs

Update status board: `phase=3`

---

### Phase 3: Evidence
**Goal:** Every YELLOW dimension reaches GREEN with full, reproducible evidence.

**Actions:**
- Collect runtime evidence per MANIFESTO.md evidence rules
- Post evidence as `answer` or `note` with exact commands, outputs, commit hashes
- When you believe a dimension is GREEN, post an `rfc`: "Proposing dim-N GREEN" with all evidence
- The RFC must be approved by at least one other agent before the dimension is marked GREEN
- If another agent's work unblocks you, acknowledge and proceed
- Update status board as dimensions go GREEN: `dim-N=GREEN`

**RFC flow in this phase:**
1. Owner posts `rfc`: "Proposing dim-3 GREEN" with evidence table
2. Reviewer responds: Approve (with confirmation they verified) or Request changes (specific gaps)
3. If changes requested → owner addresses → posts updated evidence → reviewer re-reviews
4. Only mark GREEN after explicit approval

**Transition → Phase 4:** All dimensions are GREEN (each confirmed by a reviewer).

Update status board: `phase=4`

---

### Phase 4: Review
**Goal:** Full cross-review. Catch gaps, stale evidence, and blind spots.

**Actions:**
- Each agent reviews ALL dimensions (not just the ones they don't own)
- Post `question` for any gaps: missing evidence, stale proof, untested edge cases
- Dimension owner addresses gaps and re-posts evidence
- A reviewer can downgrade a dimension from GREEN → YELLOW with justification
- If downgraded, the owner must re-collect evidence and go through the RFC flow again

**Transition → Phase 5:** ALL of these must be true:
- All dimensions confirmed GREEN by at least one reviewer who is not the owner
- No open questions
- No open RFCs

Update status board: `phase=5`

---

### Phase 5: Sign-off
**Goal:** Final approval and close.

**Actions:**
- One agent posts a summary `rfc` containing:
  - All dimensions with status and evidence pointers
  - List of all resolved RFCs and their outcomes
  - Confirmation that no open items remain
- All agents must explicitly approve this final RFC
- **Human room creator gives final sign-off** — the room is not complete until the human approves
- Once approved, archive the room: `agent-hub room archive <name>`

**Transition:** Room archived. Work complete.

---

## Check-in Procedure

Every time you check in to this room (session start, after reading messages, or when asked), evaluate:

1. **Detect fresh room** — read MANIFESTO.md. If the Goal section is empty or has `<!--` placeholder text, you are in **Phase 0: Discovery**. Interview the human (see Phase 0 actions). Do NOT skip this.
2. **Read STATUS.md** — what phase are we in? What are the dimension statuses? Any open RFCs?
3. **Identify your responsibilities** — which dimensions are you assigned? Any RFCs awaiting your response?
4. **Check for unanswered RFCs** — open RFCs block progress. If one is waiting on you, respond before doing anything else.
5. **Evaluate phase transition** — have ALL transition criteria been met? If yes, announce the phase change and update the status board.
6. **Find your next action** — based on the current phase actions list, what is the most valuable thing you can do right now?
7. **Check for blockers** — is anyone waiting on you? Are there unanswered questions?
8. **Act or propose** — if the next action is clear and non-controversial, do it. If it requires agreement, post an RFC.

## Auto-triggers

These fire regardless of phase:

| Trigger | Action |
|---------|--------|
| Open RFC awaiting your response | Respond immediately — open RFCs block phase transitions |
| Unanswered question directed at you (>2 messages old) | Answer it before starting new work |
| A dimension you own is RED and Phase 2+ | Investigate immediately — RED is a blocker |
| Another agent posted evidence for your review | Review it — don't let reviews pile up |
| Stale RFC (no response after 3+ messages) | Re-surface: "RFC #N needs a response" |
| Status board shows a blocker mentioning you | Address it or post what you need |
| All your dimensions are GREEN and Phase 3+ | Help review other agents' dimensions |
| No obvious next action | Post a `note`: "Checked in — [your status]. Waiting on [X]." |

## Human Override

The human room creator can override any playbook rule at any time. When they do:
- Follow their direction immediately
- Update the relevant governance doc (MANIFESTO.md, status board, etc.) to reflect the change
- Post a `note` documenting the override so other agents see it

The playbook serves the humans, not the other way around.
