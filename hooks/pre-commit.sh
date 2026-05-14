#!/bin/bash
#
# cody-switch pre-commit hook (managed)
#
# Blocks staging .codex/features/<X>/... when X is not the active feature.
# This prevents feature-context edits from being committed onto the wrong
# branch after a broad `git add .`.
#
# Overrides:
#   git commit --no-verify
#   CODY_SWITCH_SKIP_FEATURE_GUARD=1 git commit -m "..."
#   [skip-feature-guard] in the commit message body for editor commits/amends
#
# cody-switch:managed:pre-commit:v1

set -e

if [ "${CODY_SWITCH_SKIP_FEATURE_GUARD:-}" = "1" ]; then
    exit 0
fi

git_dir="$(git rev-parse --git-dir 2>/dev/null)" || exit 0
git_top="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
git_common="$(git rev-parse --git-common-dir 2>/dev/null)"

if [ -f "$git_top/.git" ] && [ -n "$git_common" ]; then
    main_repo="$(cd "$git_top" && cd "$git_common/.." && pwd)"
else
    main_repo="$git_top"
fi

case "$git_top" in
    "$main_repo/.codex/worktrees/"*)
        active="$(basename "$git_top")"
        ;;
    *)
        tracker="$main_repo/.codex-current-feature"
        [ -f "$tracker" ] || exit 0
        active="$(cat "$tracker" 2>/dev/null)"
        ;;
esac
[ -n "$active" ] || exit 0

msg_file="$git_dir/COMMIT_EDITMSG"
if [ -f "$msg_file" ] && grep -q "\[skip-feature-guard\]" "$msg_file" 2>/dev/null; then
    exit 0
fi

leaked=""
while IFS= read -r path; do
    [ -z "$path" ] && continue
    case "$path" in
        .codex/features/archived/*|.codex/features/lessons-global.md) continue ;;
        .codex/features/*/*)
            rest="${path#.codex/features/}"
            feature="${rest%%/*}"
            if [ "$feature" != "$active" ]; then
                leaked="${leaked}  ${feature}: ${path}"$'\n'
            fi
            ;;
    esac
done < <(git diff --cached --name-only --diff-filter=ACDMRTU 2>/dev/null)

if [ -n "$leaked" ]; then
    {
        echo ""
        echo "cody-switch pre-commit: cross-feature edits blocked"
        echo "  Active feature: '${active}'"
        echo "  Staged paths under non-active feature(s):"
        printf '%s' "$leaked"
        echo "  These edits belong on the owning feature's base branch, not this branch."
        echo ""
        echo "  Overrides:"
        echo "    git commit --no-verify"
        echo "    CODY_SWITCH_SKIP_FEATURE_GUARD=1 git commit -m \"...\""
        echo "    add [skip-feature-guard] to commit message body (editor commits only)"
        echo ""
    } >&2
    exit 1
fi

exit 0
