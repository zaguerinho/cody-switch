#!/usr/bin/env bats

load test_helper

# Helper: validate JSON and extract a field
json_field() {
    python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('$1',''))" <<< "$1_input"
}

# Helper: validate output is parseable JSON
assert_valid_json() {
    echo "$output" | python3 -c "import json,sys; json.load(sys.stdin)" 2>/dev/null
}

# Helper: extract a field from JSON output
get_field() {
    echo "$output" | python3 -c "import json,sys; d=json.load(sys.stdin); v=d.get('$1'); print('' if v is None else v)"
}

# Helper: extract a boolean field
get_bool() {
    echo "$output" | python3 -c "import json,sys; d=json.load(sys.stdin); print(str(d.get('$1','')).lower())"
}

# =============================================================================
# JSON Output Tests (16 tests)
# =============================================================================

@test "current --json outputs valid JSON with feature and branch" {
    run "$CLAUDE_SWITCH" current --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local feat
    feat=$(get_field command)
    [ "$feat" = "current" ]
    local success
    success=$(get_bool success)
    [ "$success" = "true" ]
}

@test "list --json outputs features array" {
    # Create a second feature
    mkdir -p .codex/features/other-feat
    echo "# Other" > .codex/features/other-feat/AGENTS.md
    run "$CLAUDE_SWITCH" list --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local cmd
    cmd=$(get_field command)
    [ "$cmd" = "list" ]
    # Should have features array
    echo "$output" | python3 -c "
import json,sys
d=json.load(sys.stdin)
assert isinstance(d['features'], list), 'features should be a list'
assert len(d['features']) >= 1, 'should have at least 1 feature'
"
}

@test "init --json outputs stack detection" {
    local fresh="$BATS_TEST_TMPDIR/json-init"
    mkdir -p "$fresh" && cd "$fresh"
    git init -q && echo "# Test" > README.md && git add . && git commit -q -m "init"
    echo '{"name":"test"}' > package.json
    echo '{}' > tsconfig.json
    run "$CLAUDE_SWITCH" init --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local stack
    stack=$(get_field stack)
    [ "$stack" = "typescript" ]
}

@test "blank --json creates feature and returns JSON" {
    run "$CLAUDE_SWITCH" blank json-test-feat --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local feat
    feat=$(get_field feature)
    [ "$feat" = "json-test-feat" ]
    # Feature should actually exist
    [ -d .codex/features/json-test-feat ]
}

@test "archive --json returns success" {
    # Create a feature to archive
    mkdir -p .codex/features/to-archive
    echo "# Archive me" > .codex/features/to-archive/AGENTS.md
    run "$CLAUDE_SWITCH" archive to-archive --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local cmd
    cmd=$(get_field command)
    [ "$cmd" = "archive" ]
}

@test "unarchive --json returns success" {
    # Create and archive a feature
    mkdir -p .codex/features/archived/to-unarch
    echo "# Unarchive me" > .codex/features/archived/to-unarch/AGENTS.md
    run "$CLAUDE_SWITCH" unarchive to-unarch --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local cmd
    cmd=$(get_field command)
    [ "$cmd" = "unarchive" ]
}

@test "peek --json returns content" {
    run "$CLAUDE_SWITCH" peek test-active --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local status_field
    status_field=$(get_field status)
    [ "$status_field" = "active" ]
    # Content should be non-empty
    echo "$output" | python3 -c "
import json,sys
d=json.load(sys.stdin)
assert len(d['content']) > 0, 'content should not be empty'
"
}

@test "delete --json auto-confirms and deletes" {
    # Create a feature to delete
    mkdir -p .codex/features/to-delete
    echo "# Delete me" > .codex/features/to-delete/AGENTS.md
    run "$CLAUDE_SWITCH" delete to-delete --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local cmd
    cmd=$(get_field command)
    [ "$cmd" = "delete" ]
    # Should actually be deleted
    [ ! -d .codex/features/to-delete ]
}

@test "doctor --json returns features and summary" {
    run "$CLAUDE_SWITCH" doctor --json
    [ "$status" -eq 0 ]
    assert_valid_json
    echo "$output" | python3 -c "
import json,sys
d=json.load(sys.stdin)
assert 'features' in d, 'should have features'
assert 'summary' in d, 'should have summary'
assert 'issues' in d, 'should have issues'
"
}

@test "switch --json returns previous and target" {
    # Create a target feature to switch to
    mkdir -p .codex/features/switch-target
    echo "# Switch target" > .codex/features/switch-target/AGENTS.md
    run "$CLAUDE_SWITCH" switch-target --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local cmd
    cmd=$(get_field command)
    [ "$cmd" = "switch" ]
    local feat
    feat=$(get_field feature)
    [ "$feat" = "switch-target" ]
}

@test "fork --json returns source and target" {
    run "$CLAUDE_SWITCH" fork test-active json-fork-test --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local cmd
    cmd=$(get_field command)
    [ "$cmd" = "fork" ]
    local src
    src=$(get_field source)
    [ "$src" = "test-active" ]
}

@test "error in --json mode outputs JSON error" {
    run "$CLAUDE_SWITCH" archive --json
    [ "$status" -ne 0 ]
    assert_valid_json
    local success
    success=$(get_bool success)
    [ "$success" = "false" ]
    # Should have error message
    echo "$output" | python3 -c "
import json,sys
d=json.load(sys.stdin)
assert len(d.get('error','')) > 0, 'should have error message'
"
}

@test "--json flag works in any position" {
    run "$CLAUDE_SWITCH" --json current
    [ "$status" -eq 0 ]
    assert_valid_json
}

@test "--output=json is equivalent to --json" {
    run "$CLAUDE_SWITCH" current --output=json
    [ "$status" -eq 0 ]
    assert_valid_json
}

@test "text output is default (no JSON)" {
    run "$CLAUDE_SWITCH" current
    [ "$status" -eq 0 ]
    # Should NOT be valid JSON (has ANSI codes, human text)
    if echo "$output" | python3 -c "import json,sys; json.load(sys.stdin)" 2>/dev/null; then
        # If it parsed as JSON, that's wrong for text mode
        false
    fi
}

@test "new --json returns feature name" {
    # Need a fresh repo without tracker
    local fresh="$BATS_TEST_TMPDIR/json-new"
    mkdir -p "$fresh" && cd "$fresh"
    git init -q && echo "# Test" > README.md && git add . && git commit -q -m "init"
    echo "# My context" > AGENTS.md
    mkdir -p tasks
    echo "- [ ] json task" > tasks/todo.md
    mkdir -p .codex/features
    run "$CLAUDE_SWITCH" new my-new-feat --json
    [ "$status" -eq 0 ]
    assert_valid_json
    local feat
    feat=$(get_field feature)
    [ "$feat" = "my-new-feat" ]

    # Storage should be created immediately
    [ -f .codex/features/my-new-feat/AGENTS.md ]
    [[ "$(cat .codex/features/my-new-feat/AGENTS.md)" == *"My context"* ]]
    [ -d .codex/features/my-new-feat/tasks ]
}
