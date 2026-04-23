#!/usr/bin/env bats

load test_helper

# =============================================================================
# Tier 1: Data integrity (7 tests)
# =============================================================================

@test "help works outside git repo" {
    cd "$BATS_TEST_TMPDIR"
    run "$CLAUDE_SWITCH" help
    [ "$status" -eq 0 ]
    [[ "$output" == *"cody-switch"* ]]
}

@test "fails outside git repo for project commands" {
    cd "$BATS_TEST_TMPDIR"
    run "$CLAUDE_SWITCH" list
    [ "$status" -ne 0 ]
    [[ "$output" == *"Not inside a git repository"* ]]
}

@test "uncommitted changes block switching" {
    cd "$REPO"

    # Create a second feature to switch to
    mkdir -p .codex/features/other
    echo "# Other" > .codex/features/other/AGENTS.md

    # Stage a change to a tracked file (triggers git diff --cached)
    echo "modified" >> README.md
    git add README.md

    run "$CLAUDE_SWITCH" other
    [ "$status" -ne 0 ]
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"uncommitted"* ]] || [[ "$cleaned" == *"Uncommitted"* ]]
}

@test "switch preserves source feature data" {
    cd "$REPO"

    # Set up source feature with known content
    echo "# Source Content" > AGENTS.md
    echo "- [ ] source task" > tasks/todo.md
    echo "source-feat" > .codex-current-feature
    mkdir -p .codex/features/source-feat
    echo "# Source Content" > .codex/features/source-feat/AGENTS.md

    # Create target feature
    mkdir -p .codex/features/target-feat
    echo "# Target Content" > .codex/features/target-feat/AGENTS.md
    mkdir -p .codex/features/target-feat/tasks
    echo "- [ ] target task" > .codex/features/target-feat/tasks/todo.md
    echo "## Lessons" > .codex/features/target-feat/tasks/lessons.md

    run "$CLAUDE_SWITCH" target-feat
    [ "$status" -eq 0 ]

    # Source data should be saved to storage
    [ -f .codex/features/source-feat/AGENTS.md ]
    [[ "$(cat .codex/features/source-feat/AGENTS.md)" == *"Source Content"* ]]
    [ -f .codex/features/source-feat/tasks/todo.md ]
    [[ "$(cat .codex/features/source-feat/tasks/todo.md)" == *"source task"* ]]
}

@test "switch restores target feature data" {
    cd "$REPO"

    # Create target feature with known content
    mkdir -p .codex/features/target-feat/tasks
    echo "# Target Content" > .codex/features/target-feat/AGENTS.md
    echo "- [ ] target task" > .codex/features/target-feat/tasks/todo.md
    echo "## Lessons" > .codex/features/target-feat/tasks/lessons.md

    run "$CLAUDE_SWITCH" target-feat
    [ "$status" -eq 0 ]

    # Target data should be at project root
    [[ "$(cat AGENTS.md)" == *"Target Content"* ]]
    [[ "$(cat tasks/todo.md)" == *"target task"* ]]
}

@test "round-trip switch preserves all data" {
    cd "$REPO"

    # Feature A is active with known content
    echo "# Feature A" > AGENTS.md
    echo "- [ ] A task" > tasks/todo.md
    echo "feat-a" > .codex-current-feature
    mkdir -p .codex/features/feat-a
    echo "# Feature A" > .codex/features/feat-a/AGENTS.md

    # Feature B exists in storage
    mkdir -p .codex/features/feat-b/tasks
    echo "# Feature B" > .codex/features/feat-b/AGENTS.md
    echo "- [ ] B task" > .codex/features/feat-b/tasks/todo.md
    echo "## Lessons" > .codex/features/feat-b/tasks/lessons.md

    # Switch A -> B
    run "$CLAUDE_SWITCH" feat-b
    [ "$status" -eq 0 ]

    # Switch B -> A
    run "$CLAUDE_SWITCH" feat-a
    [ "$status" -eq 0 ]

    # Feature A data should be back at root
    [[ "$(cat AGENTS.md)" == *"Feature A"* ]]
    [[ "$(cat tasks/todo.md)" == *"A task"* ]]
}

@test "untracked AGENTS.md not silently deleted on switch" {
    cd "$REPO"

    # Remove the current feature tracker so AGENTS.md appears untracked
    rm -f .codex-current-feature

    # Create a feature to switch to
    mkdir -p .codex/features/target-feat
    echo "# Target" > .codex/features/target-feat/AGENTS.md

    # Pipe "skip" to stdin so the save prompt is skipped
    run bash -c 'echo "" | "$1" target-feat' -- "$CLAUDE_SWITCH"

    # Root AGENTS.md should now be the target's content (we didn't save)
    [[ "$(cat AGENTS.md)" == *"Target"* ]]
}

# =============================================================================
# Tier 2: Core commands (15 tests)
# =============================================================================

@test "list shows features with correct indicators" {
    cd "$REPO"

    # Feature with tasks
    mkdir -p .codex/features/with-tasks/tasks
    echo "# With Tasks" > .codex/features/with-tasks/AGENTS.md
    echo "todo" > .codex/features/with-tasks/tasks/todo.md

    # Feature with session
    mkdir -p .codex/features/with-session
    echo "# With Session" > .codex/features/with-session/AGENTS.md
    echo "abc123" > .codex/features/with-session/session

    run "$CLAUDE_SWITCH" list
    [ "$status" -eq 0 ]
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"with-tasks"* ]]
    [[ "$cleaned" == *"[tasks]"* ]]
    [[ "$cleaned" == *"with-session"* ]]
    [[ "$cleaned" == *"[session]"* ]]
}

@test "list works with no features" {
    cd "$REPO"

    # Remove all features
    rm -rf .codex/features
    rm -f .codex-current-feature
    rm -f AGENTS.md

    run "$CLAUDE_SWITCH" list
    # Should not error — just show empty or "no features" message
    [ "$status" -eq 0 ]
}

@test "blank creates feature dir and docs folder" {
    cd "$REPO"

    run bash -c 'echo "n" | "$1" blank new-feat' -- "$CLAUDE_SWITCH"
    [ "$status" -eq 0 ]

    [ -d .codex/features/new-feat ]
    [ -f .codex/features/new-feat/AGENTS.md ]
    [ -d docs/new-feat ]
}

@test "blank rejects reserved name archived" {
    cd "$REPO"

    run bash -c 'echo "n" | "$1" blank archived' -- "$CLAUDE_SWITCH"
    [ "$status" -ne 0 ]
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"reserved"* ]] || [[ "$cleaned" == *"Reserved"* ]]
}

@test "blank rejects duplicate feature name" {
    cd "$REPO"

    mkdir -p .codex/features/existing
    echo "# Existing" > .codex/features/existing/AGENTS.md

    run bash -c 'echo "n" | "$1" blank existing' -- "$CLAUDE_SWITCH"
    [ "$status" -ne 0 ]
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"already exists"* ]] || [[ "$cleaned" == *"Already exists"* ]]
}

@test "blank --branch creates git branch" {
    cd "$REPO"

    run bash -c 'echo "n" | "$1" blank branch-feat --branch' -- "$CLAUDE_SWITCH"
    [ "$status" -eq 0 ]

    # Branch should exist
    run git branch --list branch-feat
    [[ "$output" == *"branch-feat"* ]]
}

@test "blank --workflow seeds from workflow.md template" {
    cd "$REPO"

    run bash -c 'echo "n" | "$1" blank wf-feat --workflow' -- "$CLAUDE_SWITCH"
    [ "$status" -eq 0 ]

    # AGENTS.md should contain workflow template content
    [[ "$(cat .codex/features/wf-feat/AGENTS.md)" == *"Context"* ]]
    [[ "$(cat .codex/features/wf-feat/AGENTS.md)" == *"Architecture"* ]]
}

@test "fork copies AGENTS.md and tasks from source" {
    cd "$REPO"

    # Set up source feature
    mkdir -p .codex/features/source-feat/tasks
    echo "# Source CLAUDE" > .codex/features/source-feat/AGENTS.md
    echo "- [ ] source task" > .codex/features/source-feat/tasks/todo.md
    echo "source lessons" > .codex/features/source-feat/tasks/lessons.md
    mkdir -p docs/source-feat
    echo "source docs" > docs/source-feat/notes.md

    run bash -c 'echo "n" | "$1" fork source-feat forked-feat' -- "$CLAUDE_SWITCH"
    [ "$status" -eq 0 ]

    # Forked feature should have copies
    [ -f .codex/features/forked-feat/AGENTS.md ]
    [[ "$(cat .codex/features/forked-feat/AGENTS.md)" == *"Source CLAUDE"* ]]
    [ -f .codex/features/forked-feat/tasks/todo.md ]
    [ -d docs/forked-feat ]
}

@test "fork --without-docs skips docs" {
    cd "$REPO"

    mkdir -p .codex/features/source-feat
    echo "# Source" > .codex/features/source-feat/AGENTS.md
    mkdir -p docs/source-feat
    echo "docs" > docs/source-feat/notes.md

    run bash -c 'echo "n" | "$1" fork source-feat no-docs-feat --without-docs' -- "$CLAUDE_SWITCH"
    [ "$status" -eq 0 ]

    [ -f .codex/features/no-docs-feat/AGENTS.md ]
    # --without-docs skips copying source docs (empty docs folder is still created)
    [ ! -f docs/no-docs-feat/notes.md ]
}

@test "fork --without-tasks creates fresh scaffold" {
    cd "$REPO"

    mkdir -p .codex/features/source-feat/tasks
    echo "# Source" > .codex/features/source-feat/AGENTS.md
    echo "old task" > .codex/features/source-feat/tasks/todo.md

    run bash -c 'echo "n" | "$1" fork source-feat no-tasks-feat --without-tasks' -- "$CLAUDE_SWITCH"
    [ "$status" -eq 0 ]

    [ -f .codex/features/no-tasks-feat/AGENTS.md ]
    # Tasks should be fresh scaffold, not copies
    if [ -d .codex/features/no-tasks-feat/tasks ]; then
        [[ "$(cat .codex/features/no-tasks-feat/tasks/todo.md)" != *"old task"* ]]
    fi
}

@test "new persists AGENTS.md and tasks to storage immediately" {
    # Need a fresh repo without tracker
    local fresh="$BATS_TEST_TMPDIR/new-persist"
    mkdir -p "$fresh" && cd "$fresh"
    git init -q && echo "# Test" > README.md && git add . && git commit -q -m "init"
    echo "# My Feature Content" > AGENTS.md
    mkdir -p tasks
    echo "- [ ] my task" > tasks/todo.md
    echo "## Lessons" > tasks/lessons.md
    mkdir -p .codex/features

    run "$CLAUDE_SWITCH" new persist-feat
    [ "$status" -eq 0 ]

    # AGENTS.md should be persisted to storage immediately
    [ -f .codex/features/persist-feat/AGENTS.md ]
    [[ "$(cat .codex/features/persist-feat/AGENTS.md)" == *"My Feature Content"* ]]

    # Tasks should be persisted to storage immediately
    [ -d .codex/features/persist-feat/tasks ]
    [ -f .codex/features/persist-feat/tasks/todo.md ]
    [[ "$(cat .codex/features/persist-feat/tasks/todo.md)" == *"my task"* ]]

    # Docs folder should exist
    [ -d docs/persist-feat ]

    # Tracker should be set
    [[ "$(cat .codex-current-feature)" == "persist-feat" ]]
}

@test "current shows active feature" {
    cd "$REPO"

    echo "my-feature" > .codex-current-feature

    run "$CLAUDE_SWITCH" current
    [ "$status" -eq 0 ]
    local cleaned
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"my-feature"* ]]
}

@test "peek shows content without switching" {
    cd "$REPO"

    mkdir -p .codex/features/peek-target
    echo "# Peek Content Here" > .codex/features/peek-target/AGENTS.md

    run "$CLAUDE_SWITCH" peek peek-target
    [ "$status" -eq 0 ]
    [[ "$output" == *"Peek Content Here"* ]]

    # Active feature should not have changed
    [[ "$(cat .codex-current-feature)" == "test-active" ]]
}

@test "archive moves feature to archived/" {
    cd "$REPO"

    # Create a feature to archive (not the active one)
    mkdir -p .codex/features/to-archive
    echo "# Archive Me" > .codex/features/to-archive/AGENTS.md
    mkdir -p docs/to-archive

    run "$CLAUDE_SWITCH" archive to-archive
    [ "$status" -eq 0 ]

    [ -f .codex/features/archived/to-archive/AGENTS.md ]
    [ ! -d .codex/features/to-archive ]
    [ -d docs/archived/to-archive ]
    [ ! -d docs/to-archive ]
}

@test "unarchive restores from archived/" {
    cd "$REPO"

    # Set up archived feature
    mkdir -p .codex/features/archived/was-archived
    echo "# Was Archived" > .codex/features/archived/was-archived/AGENTS.md
    mkdir -p docs/archived/was-archived

    run "$CLAUDE_SWITCH" unarchive was-archived
    [ "$status" -eq 0 ]

    [ -f .codex/features/was-archived/AGENTS.md ]
    [ ! -d .codex/features/archived/was-archived ]
    [ -d docs/was-archived ]
}

@test "delete removes feature and docs" {
    cd "$REPO"

    # Create a feature to delete (not the active one)
    mkdir -p .codex/features/to-delete
    echo "# Delete Me" > .codex/features/to-delete/AGENTS.md
    mkdir -p docs/to-delete

    run bash -c 'echo "y" | "$1" delete to-delete' -- "$CLAUDE_SWITCH"
    [ "$status" -eq 0 ]

    [ ! -d .codex/features/to-delete ]
    [ ! -d docs/to-delete ]
}
