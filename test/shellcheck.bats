#!/usr/bin/env bats

load test_helper

@test "shellcheck passes with no errors" {
    run shellcheck --severity=error "$CLAUDE_SWITCH"
    echo "$output"
    [ "$status" -eq 0 ]
}
