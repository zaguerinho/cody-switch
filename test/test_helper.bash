# test_helper.bash — shared setup/teardown for cody-switch bats tests

# Absolute path to the script under test
CODY_SWITCH_BIN="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/cody-switch"

# Strip ANSI color codes from output
strip_ansi() {
    sed $'s/\033\[[0-9;]*m//g'
}

setup() {
    # Create an isolated git repo for each test
    export REPO="$BATS_TEST_TMPDIR/repo"
    mkdir -p "$REPO"
    cd "$REPO"

    git init -q
    echo "# Test repo" > README.md
    git add README.md
    git commit -q -m "init"

    # Create root AGENTS.md and tasks (untracked — won't trigger check_uncommitted)
    echo "# Active feature context" > AGENTS.md
    mkdir -p tasks
    echo "- [ ] task one" > tasks/todo.md
    echo "## Lessons" > tasks/lessons.md

    # Track the active feature
    echo "test-active" > .codex-current-feature
    mkdir -p .codex/features/test-active
    echo "# Active feature context" > .codex/features/test-active/AGENTS.md
}

teardown() {
    # bats auto-cleans $BATS_TEST_TMPDIR — nothing to do
    :
}
