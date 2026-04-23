#!/usr/bin/env bats

load test_helper

# Helper: create a fresh repo without any cody-switch state
setup_fresh_repo() {
    export REPO="$BATS_TEST_TMPDIR/fresh-repo"
    mkdir -p "$REPO"
    cd "$REPO"
    git init -q
    echo "# Test repo" > README.md
    git add README.md
    git commit -q -m "init"
}

# =============================================================================
# Init — Project Bootstrap (13 tests)
# =============================================================================

@test "init creates .codex/features/ and AGENTS.md on fresh repo" {
    setup_fresh_repo
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    [ -d .codex/features ]
    [ -f AGENTS.md ]
    # Check AGENTS.md has project header
    grep -q "# Project:" AGENTS.md
}

@test "init detects node stack from package.json" {
    setup_fresh_repo
    echo '{"name": "test"}' > package.json
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    grep -q "Node.js" AGENTS.md
}

@test "init detects typescript from tsconfig.json" {
    setup_fresh_repo
    echo '{"name": "test"}' > package.json
    echo '{}' > tsconfig.json
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    grep -q "TypeScript" AGENTS.md
}

@test "init detects python stack from pyproject.toml" {
    setup_fresh_repo
    echo '[project]' > pyproject.toml
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    grep -q "Python" AGENTS.md
}

@test "init detects go stack from go.mod" {
    setup_fresh_repo
    echo 'module example.com/test' > go.mod
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    grep -q "Go" AGENTS.md
}

@test "init detects jest from jest.config.js" {
    setup_fresh_repo
    echo '{"name": "test"}' > package.json
    echo 'module.exports = {}' > jest.config.js
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    grep -q "npm test" AGENTS.md
}

@test "init detects pytest from conftest.py" {
    setup_fresh_repo
    echo '[project]' > pyproject.toml
    touch conftest.py
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    grep -q "pytest" AGENTS.md
}

@test "init detects bats test framework" {
    setup_fresh_repo
    mkdir -p test
    echo '#!/usr/bin/env bats' > test/example.bats
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    grep -q "bats" AGENTS.md
}

@test "init adds .gitignore entries" {
    setup_fresh_repo
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    [ -f .gitignore ]
    grep -qxF ".codex-current-feature" .gitignore
    grep -qxF ".codex-last-seen-feature" .gitignore
    grep -qxF ".codex-with-refs" .gitignore
    grep -qxF "tasks/" .gitignore
}

@test "init skips existing .gitignore entries" {
    setup_fresh_repo
    echo ".codex-current-feature" > .gitignore
    echo "tasks/" >> .gitignore
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    # Should not have duplicates
    local count
    count=$(grep -cxF ".codex-current-feature" .gitignore)
    [ "$count" -eq 1 ]
    count=$(grep -cxF "tasks/" .gitignore)
    [ "$count" -eq 1 ]
    # Should have the missing ones
    grep -qxF ".codex-last-seen-feature" .gitignore
    grep -qxF ".codex-with-refs" .gitignore
}

@test "init warns if already initialized" {
    setup_fresh_repo
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    # Create a feature so the guard triggers
    mkdir -p .codex/features/some-feature
    echo "# test" > .codex/features/some-feature/AGENTS.md
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    local clean
    clean="$(echo "$output" | strip_ansi)"
    [[ "$clean" == *"already initialized"* ]]
}

@test "init --template overrides auto-detection" {
    setup_fresh_repo
    echo '{"name": "test"}' > package.json
    run "$CLAUDE_SWITCH" init --template go
    [ "$status" -eq 0 ]
    # Should show Go, not Node.js
    grep -q "Go" AGENTS.md
    ! grep -q "Node.js" AGENTS.md
}

@test "init --force reinitializes with backup" {
    setup_fresh_repo
    run "$CLAUDE_SWITCH" init
    [ "$status" -eq 0 ]
    echo "custom content" >> AGENTS.md
    run "$CLAUDE_SWITCH" init --force
    [ "$status" -eq 0 ]
    # Backup should exist
    local backups
    backups=$(ls AGENTS.md.backup.* 2>/dev/null | wc -l | tr -d ' ')
    [ "$backups" -ge 1 ]
    # AGENTS.md should be regenerated (no custom content)
    ! grep -q "custom content" AGENTS.md
}
