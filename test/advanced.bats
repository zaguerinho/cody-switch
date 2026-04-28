#!/usr/bin/env bats

load test_helper

# =============================================================================
# Doctor (4 tests)
# =============================================================================

@test "doctor reports healthy features" {
    cd "$REPO"

    mkdir -p .codex/features/healthy
    echo "# Healthy" > .codex/features/healthy/AGENTS.md

    run "$CODY_SWITCH_BIN" doctor
    [ "$status" -eq 0 ]
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"healthy"* ]]
}

@test "doctor detects missing AGENTS.md" {
    cd "$REPO"

    # Feature with session but no AGENTS.md
    mkdir -p .codex/features/lost-feat
    echo "abc123" > .codex/features/lost-feat/session

    run "$CODY_SWITCH_BIN" doctor
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"lost-feat"* ]]
    [[ "$cleaned" == *"LOST"* ]] || [[ "$cleaned" == *"missing"* ]] || [[ "$cleaned" == *"Missing"* ]]
}

@test "doctor detects empty AGENTS.md" {
    cd "$REPO"

    mkdir -p .codex/features/empty-feat
    touch .codex/features/empty-feat/AGENTS.md

    run "$CODY_SWITCH_BIN" doctor
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"empty-feat"* ]]
    [[ "$cleaned" == *"EMPTY"* ]] || [[ "$cleaned" == *"empty"* ]]
}

@test "doctor --fix creates scaffold for missing AGENTS.md" {
    cd "$REPO"

    # Feature with session but no AGENTS.md
    mkdir -p .codex/features/fix-me
    echo "abc123" > .codex/features/fix-me/session

    run "$CODY_SWITCH_BIN" doctor --fix
    [ "$status" -eq 0 ]

    [ -f .codex/features/fix-me/AGENTS.md ]
    [ -s .codex/features/fix-me/AGENTS.md ]
    [[ "$(cat .codex/features/fix-me/AGENTS.md)" == *"fix-me"* ]]
}

# =============================================================================
# Merge (4 tests)
# =============================================================================

@test "merge appends AGENTS.md and tasks, archives source" {
    cd "$REPO"

    # Source feature
    mkdir -p .codex/features/merge-src/tasks
    echo "# Source context" > .codex/features/merge-src/AGENTS.md
    echo "- [ ] src task" > .codex/features/merge-src/tasks/todo.md
    echo "src lessons" > .codex/features/merge-src/tasks/lessons.md

    # Active feature has content
    echo "# Active context" > AGENTS.md
    echo "- [ ] active task" > tasks/todo.md

    run bash -c 'echo "y" | "$1" merge merge-src' -- "$CODY_SWITCH_BIN"
    [ "$status" -eq 0 ]

    # Root AGENTS.md should have both contents
    [[ "$(cat AGENTS.md)" == *"Active context"* ]]
    [[ "$(cat AGENTS.md)" == *"Source context"* ]]

    # Source should be archived
    [ -d .codex/features/archived/merge-src ]
    [ ! -d .codex/features/merge-src ]
}

@test "merge blocks when source is active feature" {
    cd "$REPO"

    # Try to merge the active feature
    run bash -c 'echo "y" | "$1" merge test-active' -- "$CODY_SWITCH_BIN"
    [ "$status" -ne 0 ]
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"active"* ]] || [[ "$cleaned" == *"Active"* ]] || [[ "$cleaned" == *"Cannot"* ]]
}

@test "merge blocks self-merge" {
    cd "$REPO"

    # Create a feature and try to merge it into itself
    mkdir -p .codex/features/self-merge
    echo "# Self" > .codex/features/self-merge/AGENTS.md

    run bash -c 'echo "y" | "$1" merge self-merge into self-merge' -- "$CODY_SWITCH_BIN"
    [ "$status" -ne 0 ]
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"itself"* ]] || [[ "$cleaned" == *"same"* ]] || [[ "$cleaned" == *"Cannot"* ]]
}

@test "merge --delete removes source instead of archiving" {
    cd "$REPO"

    mkdir -p .codex/features/del-src
    echo "# Delete Source" > .codex/features/del-src/AGENTS.md

    echo "# Active" > AGENTS.md

    run bash -c 'echo "y" | "$1" merge del-src --delete' -- "$CODY_SWITCH_BIN"
    [ "$status" -eq 0 ]

    # Source should be completely gone (not archived)
    [ ! -d .codex/features/del-src ]
    [ ! -d .codex/features/archived/del-src ]
}

# =============================================================================
# Reference contexts (2 tests)
# =============================================================================

@test "--with appends reference markers to AGENTS.md" {
    cd "$REPO"

    # Create a reference feature
    mkdir -p .codex/features/ref-feat
    echo "# Reference Content" > .codex/features/ref-feat/AGENTS.md

    # Create a target to switch to with --with
    mkdir -p .codex/features/target-feat
    echo "# Target" > .codex/features/target-feat/AGENTS.md
    mkdir -p .codex/features/target-feat/tasks
    echo "todo" > .codex/features/target-feat/tasks/todo.md
    echo "lessons" > .codex/features/target-feat/tasks/lessons.md

    run "$CODY_SWITCH_BIN" target-feat --with ref-feat
    [ "$status" -eq 0 ]

    # AGENTS.md should contain reference markers
    [[ "$(cat AGENTS.md)" == *"cody-switch-ref:ref-feat"* ]]
    [[ "$(cat AGENTS.md)" == *"Reference Content"* ]]

    # Ref tracker file should exist
    [ -f .codex-with-refs ]
    [[ "$(cat .codex-with-refs)" == *"ref-feat"* ]]
}

@test "switch strips reference markers from previous feature" {
    cd "$REPO"

    # Set up active feature with reference markers already in AGENTS.md
    cat > AGENTS.md << 'EOF'
# My Feature

<!-- cody-switch-ref:some-ref:start -->
# Reference content
<!-- cody-switch-ref:some-ref:end -->
EOF
    echo "some-ref" > .codex-with-refs

    # Create target to switch to
    mkdir -p .codex/features/clean-target
    echo "# Clean Target" > .codex/features/clean-target/AGENTS.md
    mkdir -p .codex/features/clean-target/tasks
    echo "todo" > .codex/features/clean-target/tasks/todo.md
    echo "lessons" > .codex/features/clean-target/tasks/lessons.md

    run "$CODY_SWITCH_BIN" clean-target
    [ "$status" -eq 0 ]

    # Saved version of previous feature should NOT contain ref markers
    [ -f .codex/features/test-active/AGENTS.md ]
    local saved
    saved=$(cat .codex/features/test-active/AGENTS.md)
    [[ "$saved" != *"cody-switch-ref"* ]]

    # Ref tracker should be gone
    [ ! -f .codex-with-refs ]
}

@test "strip_reference_contexts removes trailing blank lines (portability)" {
    cd "$REPO"

    # Set up active feature with reference markers AND trailing blank lines
    cat > AGENTS.md << 'EOF'
# My Feature

Real content here

<!-- cody-switch-ref:some-ref:start -->
# Reference content
Some ref data
<!-- cody-switch-ref:some-ref:end -->

EOF
    echo "some-ref" > .codex-with-refs

    # Create target to switch to (triggers strip_reference_contexts via save_current)
    mkdir -p .codex/features/strip-target
    echo "# Strip Target" > .codex/features/strip-target/AGENTS.md
    mkdir -p .codex/features/strip-target/tasks
    echo "todo" > .codex/features/strip-target/tasks/todo.md
    echo "lessons" > .codex/features/strip-target/tasks/lessons.md

    run "$CODY_SWITCH_BIN" strip-target
    [ "$status" -eq 0 ]

    # Saved AGENTS.md should have no reference markers
    local saved
    saved=$(cat .codex/features/test-active/AGENTS.md)
    [[ "$saved" != *"cody-switch-ref"* ]]
    [[ "$saved" == *"Real content here"* ]]

    # Should not end with blank lines (check last line is non-empty)
    local last_line
    last_line=$(tail -1 .codex/features/test-active/AGENTS.md)
    [ -n "$last_line" ]
}

# =============================================================================
# Other (4 tests)
# =============================================================================

@test "legacy AGENTS.md.* files migrated on first run" {
    cd "$REPO"

    # Remove modern layout
    rm -rf .codex/features
    rm -f .codex-current-feature

    # Create legacy files
    echo "# Legacy Feature" > AGENTS.md.old-feat
    mkdir -p tasks.old-feat
    echo "legacy task" > tasks.old-feat/todo.md

    # Any command that touches features should trigger migration
    run "$CODY_SWITCH_BIN" list
    [ "$status" -eq 0 ]

    # Legacy files should be migrated
    [ -f .codex/features/old-feat/AGENTS.md ]
    [[ "$(cat .codex/features/old-feat/AGENTS.md)" == *"Legacy Feature"* ]]
    [ ! -f AGENTS.md.old-feat ]
}

@test "number-based switching resolves correctly" {
    cd "$REPO"

    # Create features (sorted alphabetically)
    mkdir -p .codex/features/alpha
    echo "# Alpha" > .codex/features/alpha/AGENTS.md
    mkdir -p .codex/features/alpha/tasks
    echo "todo" > .codex/features/alpha/tasks/todo.md
    echo "lessons" > .codex/features/alpha/tasks/lessons.md

    mkdir -p .codex/features/beta
    echo "# Beta" > .codex/features/beta/AGENTS.md
    mkdir -p .codex/features/beta/tasks
    echo "todo" > .codex/features/beta/tasks/todo.md
    echo "lessons" > .codex/features/beta/tasks/lessons.md

    # Get list to see numbering
    run "$CODY_SWITCH_BIN" list
    [ "$status" -eq 0 ]

    # Find which number alpha is (features sorted alphabetically)
    # alpha should be first non-active feature or early in the list
    # Switch by number 1 — should resolve to a valid feature
    run "$CODY_SWITCH_BIN" 1
    [ "$status" -eq 0 ]

    # Should have switched to some feature
    [ -f .codex-current-feature ]
}

@test "script works from subdirectory" {
    cd "$REPO"

    mkdir -p subdir/deep
    cd subdir/deep

    run "$CODY_SWITCH_BIN" list
    [ "$status" -eq 0 ]
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"test-active"* ]]
}

@test "switch to feature with missing AGENTS.md creates scaffold and warns" {
    cd "$REPO"

    # Create a feature with only a session (no AGENTS.md)
    mkdir -p .codex/features/broken-feat
    echo "abc123" > .codex/features/broken-feat/session

    run "$CODY_SWITCH_BIN" broken-feat
    [ "$status" -eq 0 ]

    # Should have created a scaffold AGENTS.md at root
    [ -f AGENTS.md ]
    [[ "$(cat AGENTS.md)" == *"broken-feat"* ]]

    # Should have warned about the missing file
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"scaffold"* ]] || [[ "$cleaned" == *"missing"* ]] || [[ "$cleaned" == *"created"* ]] || [[ "$cleaned" == *"Warning"* ]] || [[ "$cleaned" == *"warning"* ]]
}
