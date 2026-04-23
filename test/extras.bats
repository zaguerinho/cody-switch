#!/usr/bin/env bats

load test_helper

# =============================================================================
# Extras Install Mechanism (12 tests)
# =============================================================================

# Helper: create a mock extras directory in a temp location
# and point the script at it via a wrapper
setup_extras_env() {
    # Create mock global-skills-extra directory alongside the script
    export SCRIPT_DIR="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    export EXTRAS_DIR="$SCRIPT_DIR/global-skills-extra"
    export SKILLS_TARGET="$BATS_TEST_TMPDIR/claude-skills"

    # Create a temp HOME so install doesn't touch real ~/.codex
    export REAL_HOME="$HOME"
    export HOME="$BATS_TEST_TMPDIR/fakehome"
    mkdir -p "$HOME/.codex"

    # Minimal settings.json so hook install doesn't error
    echo '{}' > "$HOME/.codex/settings.json"
}

teardown_extras_env() {
    export HOME="$REAL_HOME"
}

# --- Help text ---

@test "help shows --extras flag" {
    run "$CLAUDE_SWITCH" help
    [ "$status" -eq 0 ]
    echo "$output" | strip_ansi | grep -q "\-\-extras"
}

@test "help shows --skill flag" {
    run "$CLAUDE_SWITCH" help
    [ "$status" -eq 0 ]
    echo "$output" | strip_ansi | grep -q "\-\-skill"
}

# --- Install --skill with valid extra ---

@test "install --skill installs a specific extra skill" {
    setup_extras_env

    # Verify the video-tutorial extra exists in the repo
    [ -f "$EXTRAS_DIR/video-tutorial/SKILL.md" ]

    run "$CLAUDE_SWITCH" install --skill video-tutorial
    [ "$status" -eq 0 ]

    # Check it was installed
    local output_clean
    output_clean="$(echo "$output" | strip_ansi)"
    echo "$output_clean" | grep -q "Installed Extra skill: video-tutorial"

    # Verify the file exists
    [ -f "$HOME/.codex/skills/video-tutorial/SKILL.md" ]

    teardown_extras_env
}

@test "install --skill twice says already installed" {
    setup_extras_env

    # First install
    run "$CLAUDE_SWITCH" install --skill video-tutorial
    [ "$status" -eq 0 ]

    # Second install
    run "$CLAUDE_SWITCH" install --skill video-tutorial
    [ "$status" -eq 0 ]
    echo "$output" | strip_ansi | grep -q "Extra skill already installed: video-tutorial"

    teardown_extras_env
}

# --- Install --skill with nonexistent extra ---

@test "install --skill with nonexistent name errors" {
    setup_extras_env

    run "$CLAUDE_SWITCH" install --skill nonexistent-skill
    [ "$status" -ne 0 ]
    echo "$output" | strip_ansi | grep -q "Extra skill 'nonexistent-skill' not found"

    teardown_extras_env
}

@test "install --skill error lists available extras" {
    setup_extras_env

    run "$CLAUDE_SWITCH" install --skill nonexistent-skill
    echo "$output" | strip_ansi | grep -q "video-tutorial"

    teardown_extras_env
}

# --- Install --extras (all extras) ---

@test "install --extras installs all extra skills" {
    setup_extras_env

    run "$CLAUDE_SWITCH" install --extras
    [ "$status" -eq 0 ]

    echo "$output" | strip_ansi | grep -q "Installing extra skills"
    [ -f "$HOME/.codex/skills/video-tutorial/SKILL.md" ]

    teardown_extras_env
}

# --- Install without --extras skips extras ---

@test "install without --extras does not install extras" {
    setup_extras_env

    run "$CLAUDE_SWITCH" install
    [ "$status" -eq 0 ]

    # Should NOT have installed the extra
    [ ! -f "$HOME/.codex/skills/video-tutorial/SKILL.md" ]

    teardown_extras_env
}

# --- Install --force updates extras ---

@test "install --skill --force updates modified extra" {
    setup_extras_env

    # First install
    "$CLAUDE_SWITCH" install --skill video-tutorial

    # Modify the installed copy
    echo "modified" >> "$HOME/.codex/skills/video-tutorial/SKILL.md"

    # Force reinstall
    run "$CLAUDE_SWITCH" install --skill video-tutorial --force
    [ "$status" -eq 0 ]
    echo "$output" | strip_ansi | grep -q "Updated Extra skill: video-tutorial"

    # Should have created a backup
    local backups
    backups=$(ls "$HOME/.codex/skills/video-tutorial/"*.backup.* 2>/dev/null | wc -l)
    [ "$backups" -gt 0 ]

    teardown_extras_env
}

# --- Install --skill differs without --force ---

@test "install --skill shows differs warning without --force" {
    setup_extras_env

    # First install
    "$CLAUDE_SWITCH" install --skill video-tutorial

    # Modify the installed copy
    echo "modified" >> "$HOME/.codex/skills/video-tutorial/SKILL.md"

    # Reinstall without --force
    run "$CLAUDE_SWITCH" install --skill video-tutorial
    [ "$status" -eq 0 ]
    echo "$output" | strip_ansi | grep -q "differs, use --force to update"

    teardown_extras_env
}

# --- Unknown install flag ---

@test "install with unknown flag errors" {
    setup_extras_env

    run "$CLAUDE_SWITCH" install --unknown-flag
    [ "$status" -ne 0 ]
    echo "$output" | strip_ansi | grep -q "Unknown install option"

    teardown_extras_env
}

# --- Core skills still install normally ---

@test "install still installs core skills" {
    setup_extras_env

    run "$CLAUDE_SWITCH" install
    [ "$status" -eq 0 ]

    # At least one core skill should be installed
    local core_skills
    core_skills=$(ls "$HOME/.codex/skills/" 2>/dev/null | grep -v video-tutorial | head -1)
    [ -n "$core_skills" ]

    teardown_extras_env
}
