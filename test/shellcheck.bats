#!/usr/bin/env bats

load test_helper

@test "shellcheck passes with no errors" {
    run shellcheck --severity=error "$CODY_SWITCH_BIN"
    echo "$output"
    [ "$status" -eq 0 ]
}
