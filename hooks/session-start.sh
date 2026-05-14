#!/bin/bash
#
# cody-switch SessionStart hook
#
# Injected into Codex via hooks config. Runs on session startup/resume.
# Outputs plain text that gets injected as context for Codex.
#
# Stdin: JSON with { session_id, cwd, source: "startup"|"resume"|"clear"|"compact" }
# Stdout: Context text (injected into Codex's conversation)

# Read stdin (hook input JSON)
INPUT=$(cat)

# Parse fields using python3 (always available — used as readlink fallback too)
CWD=$(echo "$INPUT" | python3 -c "import json,sys; print(json.load(sys.stdin).get('cwd',''))" 2>/dev/null)
SOURCE=$(echo "$INPUT" | python3 -c "import json,sys; print(json.load(sys.stdin).get('source',''))" 2>/dev/null)

# Skip on compact — context is already being cleaned up
if [ "$SOURCE" = "compact" ]; then
    exit 0
fi

# Skip if no cwd
if [ -z "$CWD" ]; then
    exit 0
fi

# Check agent-hub for unread messages. Arg: project root path.
check_agent_hub_unread() {
    local project_root="$1"
    if ! command -v agent-hub >/dev/null 2>&1; then
        return
    fi

    local alias=""
    if [ -f "$project_root/.agent-hub-alias" ]; then
        alias=$(cat "$project_root/.agent-hub-alias")
    elif [ -n "$AGENT_HUB_ALIAS" ]; then
        alias="$AGENT_HUB_ALIAS"
    fi

    if [ -z "$alias" ]; then
        return
    fi

    local hub_check
    hub_check=$(agent-hub check-all --as "$alias" --json 2>/dev/null)
    if [ -n "$hub_check" ]; then
        local unread
        unread=$(echo "$hub_check" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    if data.get('ok'):
        for s in data.get('data', {}).get('rooms', []):
            if s.get('unread', 0) > 0:
                print(f\"  [{s.get('room', s.get('session', '?'))}] {s['unread']} unread message(s)\")
except: pass
" 2>/dev/null)
        if [ -n "$unread" ]; then
            echo ""
            echo "agent-hub: You have unread messages"
            echo "$unread"
            echo "  Run: agent-hub read <room> --unread --as $alias"
        fi
    fi
}

# Detect if running inside a git worktree (.git is a file, not a directory)
MAIN_REPO=""
WT_FEATURE=""
if [ -f "$CWD/.git" ]; then
    # .git is a file — this is a worktree
    MAIN_REPO=$(cd "$CWD" && git rev-parse --git-common-dir 2>/dev/null)
    if [ -n "$MAIN_REPO" ]; then
        MAIN_REPO=$(cd "$CWD" && cd "$MAIN_REPO/.." && pwd)
        WT_FEATURE=$(basename "$CWD")
    fi
fi

# If inside a worktree, output worktree-specific context
if [ -n "$WT_FEATURE" ] && [ -n "$MAIN_REPO" ]; then
    FEATURES_DIR="$MAIN_REPO/.codex/features"
    echo "---"
    echo "cody-switch: Inside worktree for feature '${WT_FEATURE}'"
    echo "  Main repo: ${MAIN_REPO}"
    if [ -d "$FEATURES_DIR/$WT_FEATURE" ]; then
        echo "  Feature storage: .codex/features/${WT_FEATURE}/"
    fi
    echo ""
    # Stopping point detection (worktree)
    WT_STOPPING="$CWD/tasks/STOPPING_POINT.md"
    if [ -f "$WT_STOPPING" ]; then
        SP_SUMMARY=$(sed -n '/^## Where We Left Off/,/^##/{/^##/d;/^$/d;p;}' "$WT_STOPPING" | head -1)
        echo ""
        if [ -n "$SP_SUMMARY" ]; then
            echo "Stopping point: ${SP_SUMMARY}"
        else
            echo "Stopping point found."
        fi
        echo "  Run /stopping-point resume to pick up where you left off"
    fi

    echo ""
    echo "Remember: Read tasks/lessons.md"
    GLOBAL_LESSONS="$FEATURES_DIR/lessons-global.md"
    if [ -f "$GLOBAL_LESSONS" ]; then
        echo "  Also read: .codex/features/lessons-global.md"
    fi
    USER_LESSONS="$HOME/.codex/lessons-global.md"
    if [ -f "$USER_LESSONS" ]; then
        echo "  Also read: ~/.codex/lessons-global.md (user-level)"
    fi

    check_agent_hub_unread "$MAIN_REPO"

    echo "---"
    exit 0
fi

# Skip if project doesn't use cody-switch
TRACKER="$CWD/.codex-current-feature"
if [ ! -f "$TRACKER" ]; then
    exit 0
fi

CURRENT=$(cat "$TRACKER")
if [ -z "$CURRENT" ]; then
    exit 0
fi

FEATURES_DIR="$CWD/.codex/features"
TASKS_DIR="$CWD/tasks"

# Bootstrap root AGENTS.md from storage if missing (handles gitignored root, fresh clones)
FEATURE_CLAUDE="$FEATURES_DIR/$CURRENT/AGENTS.md"
CLAUDE_MD="$CWD/AGENTS.md"
if [ ! -f "$CLAUDE_MD" ] && [ -f "$FEATURE_CLAUDE" ]; then
    cp "$FEATURE_CLAUDE" "$CLAUDE_MD"
fi
DOCS_DIR="$CWD/docs"
LAST_SEEN="$CWD/.codex-last-seen-feature"

# --- Output context ---

echo "---"
echo "cody-switch: Active feature is '${CURRENT}'"

# Show --with references if active
WITH_REFS="$CWD/.codex-with-refs"
if [ -f "$WITH_REFS" ]; then
    REFS=$(paste -sd ', ' "$WITH_REFS")
    echo "  With references: ${REFS}"
fi

# Show available indicators
INDICATORS=""
if [ -d "$TASKS_DIR" ]; then
    INDICATORS="${INDICATORS} [tasks]"
fi
if [ -d "$DOCS_DIR/$CURRENT" ]; then
    INDICATORS="${INDICATORS} [docs]"
fi
if [ -n "$INDICATORS" ]; then
    echo "  Available:${INDICATORS}"
fi

# Compaction awareness (rec 4): detect context pollution from feature switch
if [ -f "$LAST_SEEN" ]; then
    LAST=$(cat "$LAST_SEEN")
    if [ -n "$LAST" ] && [ "$LAST" != "$CURRENT" ]; then
        echo ""
        echo "WARNING: Feature context may be stale!"
        echo "  Last seen: '${LAST}' -> Now active: '${CURRENT}'"
        echo "  The context window may contain instructions from '${LAST}'."
        echo "  Consider running /clear or /compact to clean up."
    fi
fi

# Update last-seen marker
echo "$CURRENT" > "$LAST_SEEN"

# Session resume suggestion (dual: checkpoint + latest)
FEATURE_DIR="$FEATURES_DIR/$CURRENT"
CHECKPOINT_FILE="$FEATURE_DIR/session"
LATEST_FILE="$FEATURE_DIR/session-latest"
SUMMARY_FILE="$FEATURE_DIR/session-summary"

CHECKPOINT_ID=""
LATEST_ID=""
SUMMARY=""

if [ -f "$CHECKPOINT_FILE" ]; then
    CHECKPOINT_ID=$(cat "$CHECKPOINT_FILE")
fi
if [ -f "$LATEST_FILE" ]; then
    LATEST_ID=$(cat "$LATEST_FILE")
fi
if [ -f "$SUMMARY_FILE" ]; then
    SUMMARY=$(cat "$SUMMARY_FILE")
fi

if [ -n "$CHECKPOINT_ID" ] && [ -n "$LATEST_ID" ] && [ "$CHECKPOINT_ID" != "$LATEST_ID" ]; then
    echo ""
    if [ -n "$SUMMARY" ]; then
        echo "Checkpoint: ${SUMMARY}"
    fi
    echo "Fork from checkpoint: codex fork ${CHECKPOINT_ID}"
    echo "Resume latest: codex resume ${LATEST_ID}"
elif [ -n "$CHECKPOINT_ID" ]; then
    echo ""
    if [ -n "$SUMMARY" ]; then
        echo "Checkpoint: ${SUMMARY}"
    fi
    echo "Previous session available: codex resume ${CHECKPOINT_ID}"
elif [ -n "$LATEST_ID" ]; then
    echo ""
    echo "Previous session available: codex resume ${LATEST_ID}"
fi

# Stopping point detection
STOPPING_POINT="$TASKS_DIR/STOPPING_POINT.md"
if [ -f "$STOPPING_POINT" ]; then
    SP_SUMMARY=$(sed -n '/^## Where We Left Off/,/^##/{/^##/d;/^$/d;p;}' "$STOPPING_POINT" | head -1)
    echo ""
    if [ -n "$SP_SUMMARY" ]; then
        echo "Stopping point: ${SP_SUMMARY}"
    else
        echo "Stopping point found."
    fi
    echo "  Run /stopping-point resume to pick up where you left off"
fi

# Lessons reminder
echo ""
echo "Remember: Read tasks/lessons.md"
GLOBAL_LESSONS="$FEATURES_DIR/lessons-global.md"
if [ -f "$GLOBAL_LESSONS" ]; then
    echo "  Also read: .codex/features/lessons-global.md"
fi
USER_LESSONS="$HOME/.codex/lessons-global.md"
if [ -f "$USER_LESSONS" ]; then
    echo "  Also read: ~/.codex/lessons-global.md (user-level)"
fi

# Cross-feature dirt detection. This warns when edits exist under
# .codex/features/<X>/ where X is not the currently-active feature.
if command -v git >/dev/null 2>&1 && git -C "$CWD" rev-parse --git-dir >/dev/null 2>&1; then
    LEAKED=$(git -C "$CWD" status --porcelain '.codex/features/' 2>/dev/null | \
        awk -v active="$CURRENT" '
            {
                line = substr($0, 4)
                arrow_pos = index(line, " -> ")
                if (arrow_pos > 0) line = substr(line, arrow_pos + 4)
                if (match(line, /^\.codex\/features\/[^\/]+\//)) {
                    feature = line
                    sub(/^\.codex\/features\//, "", feature)
                    sub(/\/.*/, "", feature)
                    if (feature != active) print "  " feature ": " line
                }
            }
        ')
    if [ -n "$LEAKED" ]; then
        echo ""
        echo "WARNING: Cross-feature dirt detected"
        echo "  Active feature: '${CURRENT}'. Uncommitted edits exist under non-active feature(s):"
        echo "$LEAKED"
        echo "  These edits belong to the owning feature's base branch, not to the active feature."
        echo "  Recovery: stash, switch to the owning feature's base branch, branch off, pop, commit, PR."
        echo "  Reference: ~/.codex/lessons-global.md 'Cross-Feature .codex/features/<X>/AGENTS.md Edits Leak Across Switches'"
    fi
fi

check_agent_hub_unread "$CWD"

echo "---"
