#!/usr/bin/env bats
#
# Cross-feature .codex/features/<X>/ leak protection.

load test_helper

setup_tracked_active_feature() {
    cd "$REPO"

    mkdir -p .codex/features/test-active
    echo "# Active feature context" > .codex/features/test-active/AGENTS.md
    git add .codex/features/test-active/AGENTS.md
    git -c user.email=t@t -c user.name=t commit -q -m "chore(test-active): seed feature doc"

    git config user.email "switch-test@local"
    git config user.name "switch-test"
}

setup_tracked_other_feature() {
    cd "$REPO"
    mkdir -p .codex/features/other-feat/tasks
    echo "# Other feature context" > .codex/features/other-feat/AGENTS.md
    echo "todo" > .codex/features/other-feat/tasks/todo.md
    echo "lessons" > .codex/features/other-feat/tasks/lessons.md
    git add .codex/features/other-feat/
    git -c user.email=t@t -c user.name=t commit -q -m "chore(other-feat): seed feature doc"
}

install_precommit_hook() {
    local hook_src
    hook_src="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/hooks/pre-commit.sh"
    cp "$hook_src" "$REPO/.git/hooks/pre-commit"
    chmod +x "$REPO/.git/hooks/pre-commit"
}

@test "switch auto-commits active feature dirt on non-protected branch" {
    setup_tracked_active_feature
    setup_tracked_other_feature

    git checkout -q -b work/test-active

    echo "# Active feature context (edited via root)" > AGENTS.md
    cp AGENTS.md .codex/features/test-active/AGENTS.md

    [ -n "$(git status --porcelain .codex/features/test-active/)" ]

    run "$CODY_SWITCH_BIN" other-feat
    [ "$status" -eq 0 ]
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"Auto-committed"* ]]

    [ -z "$(git status --porcelain --untracked-files=no)" ]

    last_msg=$(git log -1 --format=%s)
    [[ "$last_msg" == *"sync feature AGENTS.md"* ]]
    [[ "$last_msg" == *"test-active"* ]]
}

@test "switch refuses to auto-commit on protected branch" {
    setup_tracked_active_feature
    setup_tracked_other_feature

    git checkout -q main 2>/dev/null || git checkout -q master 2>/dev/null || true

    echo "# Active feature context (edited)" > AGENTS.md
    cp AGENTS.md .codex/features/test-active/AGENTS.md

    run "$CODY_SWITCH_BIN" other-feat
    [ "$status" -ne 0 ]
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"protected branch"* ]]
    [[ "$cleaned" == *"test-active"* ]]
}

@test "switch --no-auto-commit aborts on non-protected branch" {
    setup_tracked_active_feature
    setup_tracked_other_feature

    git checkout -q -b work/test-active

    echo "# Active feature context (edited)" > AGENTS.md
    cp AGENTS.md .codex/features/test-active/AGENTS.md

    run "$CODY_SWITCH_BIN" other-feat --no-auto-commit
    [ "$status" -ne 0 ]
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"auto-commit disabled"* ]] || [[ "$cleaned" == *"Active feature doc is dirty"* ]]
}

@test "switch reports cross-feature leak with feature names and recovery" {
    setup_tracked_active_feature
    setup_tracked_other_feature

    git checkout -q -b work/test-active

    echo "# Cross-feature edit" >> .codex/features/other-feat/AGENTS.md

    mkdir -p .codex/features/dest-feat/tasks
    echo "# Dest" > .codex/features/dest-feat/AGENTS.md
    echo "todo" > .codex/features/dest-feat/tasks/todo.md
    echo "lessons" > .codex/features/dest-feat/tasks/lessons.md

    run "$CODY_SWITCH_BIN" dest-feat
    [ "$status" -ne 0 ]
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"Cross-feature leak"* ]]
    [[ "$cleaned" == *"other-feat"* ]]
    [[ "$cleaned" == *"base branch"* ]]
    [[ "$cleaned" == *"lessons-global.md"* ]]
}

@test "switch keeps generic message for code-only dirt" {
    setup_tracked_active_feature
    setup_tracked_other_feature

    git checkout -q -b work/test-active

    echo "more" >> README.md
    git add README.md

    run "$CODY_SWITCH_BIN" other-feat
    [ "$status" -ne 0 ]
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"uncommitted"* ]] || [[ "$cleaned" == *"Uncommitted"* ]]
    [[ "$cleaned" != *"Cross-feature leak"* ]]
}

@test "pre-commit hook blocks staging cross-feature path" {
    setup_tracked_active_feature
    setup_tracked_other_feature
    install_precommit_hook

    cd "$REPO"
    git checkout -q -b work/test-active

    echo "# leaked" >> .codex/features/other-feat/AGENTS.md
    git add .codex/features/other-feat/AGENTS.md

    run git -c user.email=t@t -c user.name=t commit -m "wrong place"
    [ "$status" -ne 0 ]
    [[ "$output" == *"cross-feature edits blocked"* ]]
    [[ "$output" == *"other-feat"* ]]
}

@test "pre-commit hook allows override via env var" {
    setup_tracked_active_feature
    setup_tracked_other_feature
    install_precommit_hook

    cd "$REPO"
    git checkout -q -b work/test-active

    echo "# intentional sweep" >> .codex/features/other-feat/AGENTS.md
    git add .codex/features/other-feat/AGENTS.md

    run env CODY_SWITCH_SKIP_FEATURE_GUARD=1 git -c user.email=t@t -c user.name=t commit -m "docs sweep"
    [ "$status" -eq 0 ]
}

@test "pre-commit hook allows active-feature commits" {
    setup_tracked_active_feature
    setup_tracked_other_feature
    install_precommit_hook

    cd "$REPO"
    git checkout -q -b work/test-active

    echo "# active edit" >> .codex/features/test-active/AGENTS.md
    git add .codex/features/test-active/AGENTS.md

    run git -c user.email=t@t -c user.name=t commit -m "chore(test-active): tweak"
    [ "$status" -eq 0 ]
}

@test "pre-commit hook is no-op outside enrolled projects" {
    install_precommit_hook
    cd "$REPO"
    rm -f .codex-current-feature

    echo "irrelevant" >> README.md
    git add README.md
    run git -c user.email=t@t -c user.name=t commit -m "ok"
    [ "$status" -eq 0 ]
}

@test "switch refuses to auto-commit on detached HEAD" {
    setup_tracked_active_feature
    setup_tracked_other_feature

    git checkout -q --detach HEAD

    echo "# Active feature context (edited)" > AGENTS.md
    cp AGENTS.md .codex/features/test-active/AGENTS.md

    run "$CODY_SWITCH_BIN" other-feat
    [ "$status" -ne 0 ]
    cleaned=$(echo "$output" | strip_ansi)
    [[ "$cleaned" == *"detached HEAD"* ]] || [[ "$cleaned" == *"Detached HEAD"* ]]

    last_msg=$(git log -1 --format=%s)
    [[ "$last_msg" != *"sync feature AGENTS.md"* ]]
}

@test "pre-commit hook blocks staged deletion of cross-feature path" {
    setup_tracked_active_feature
    setup_tracked_other_feature
    install_precommit_hook

    cd "$REPO"
    git checkout -q -b work/test-active

    git rm -q .codex/features/other-feat/AGENTS.md

    run git -c user.email=t@t -c user.name=t commit -m "delete cross"
    [ "$status" -ne 0 ]
    [[ "$output" == *"cross-feature edits blocked"* ]]
    [[ "$output" == *"other-feat"* ]]
}

@test "pre-commit hook allows worktree-feature commits inside the worktree" {
    setup_tracked_active_feature
    setup_tracked_other_feature

    cd "$REPO"

    mkdir -p .codex/features/wt-feat
    echo "# wt-feat" > .codex/features/wt-feat/AGENTS.md
    git add .codex/features/wt-feat/AGENTS.md
    git -c user.email=t@t -c user.name=t commit -q -m "seed wt-feat"

    git worktree add -q .codex/worktrees/wt-feat -b worktree-wt-feat

    install_precommit_hook

    cd .codex/worktrees/wt-feat
    echo "# wt-feat edit" >> .codex/features/wt-feat/AGENTS.md
    git add .codex/features/wt-feat/AGENTS.md

    run git -c user.email=t@t -c user.name=t commit -m "chore(wt-feat): tweak"
    [ "$status" -eq 0 ]
}
