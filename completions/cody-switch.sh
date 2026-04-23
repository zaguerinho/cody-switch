#compdef cody-switch
# Tab completion for cody-switch (bash + zsh)
#
# Installation:
#   source this file from .bashrc or .zshrc, OR
#   run `cody-switch install` to set it up automatically.

# --- Shared helpers ---

# Walk up from $PWD to find the nearest .git directory (or .git file for worktrees)
_cody_switch_git_root() {
    local dir="$PWD"
    while [ "$dir" != "/" ]; do
        if [ -d "$dir/.git" ]; then
            echo "$dir"
            return 0
        elif [ -f "$dir/.git" ]; then
            # Inside a worktree — resolve to main repo
            local git_common_dir
            git_common_dir="$(cd "$dir" && git rev-parse --git-common-dir 2>/dev/null)"
            if [ -n "$git_common_dir" ]; then
                (cd "$dir" && cd "$git_common_dir/.." && pwd)
                return 0
            fi
            echo "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

# List feature names from .codex/features/*/ (excluding "archived")
_cody_switch_features() {
    local root
    root="$(_cody_switch_git_root)" || return
    local features_dir="$root/.codex/features"
    [ -d "$features_dir" ] || return
    for d in "$features_dir"/*/; do
        [ -d "$d" ] || continue
        local name="$(basename "$d")"
        [ "$name" = "archived" ] && continue
        [ -f "$d/AGENTS.md" ] && echo "$name"
    done | sort
}

# List archived feature names
_cody_switch_archived() {
    local root
    root="$(_cody_switch_git_root)" || return
    local archived_dir="$root/.codex/features/archived"
    [ -d "$archived_dir" ] || return
    for d in "$archived_dir"/*/; do
        [ -d "$d" ] || continue
        local name="$(basename "$d")"
        [ -f "$d/AGENTS.md" ] && echo "$name"
    done | sort
}

_cody_switch_prompts() {
    local script_dir
    script_dir="$(readlink -f "$(command -v cody-switch)" 2>/dev/null || echo "")"
    script_dir="$(dirname "$script_dir")"
    [ -d "$script_dir/global-commands" ] || return
    for f in "$script_dir"/global-commands/*.md; do
        [ -f "$f" ] || continue
        basename "$f" .md
    done | sort
}

_CODY_SWITCH_SUBCOMMANDS="list ls current new create blank fork peek show cat archive unarchive delete rm merge pin-session pin promote-lesson promote promote-audit audit init doctor save sync open prompt help install uninstall update"

# --- Bash completion ---

if type complete &>/dev/null; then

_cody_switch_bash() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Check if --with appears anywhere before current word
    local has_with=0
    local i
    for ((i=1; i < COMP_CWORD; i++)); do
        if [[ "${COMP_WORDS[$i]}" == "--with" || "${COMP_WORDS[$i]}" == --with=* ]]; then
            has_with=1
            break
        fi
    done

    # After --with flag: complete feature names
    if [ "$prev" = "--with" ] || [ "$has_with" -eq 1 ]; then
        local features
        features="$(_cody_switch_features)"
        COMPREPLY=( $(compgen -W "$features" -- "$cur") )
        return
    fi

    case $COMP_CWORD in
        1)
            # First arg: subcommands + feature names (for direct switching)
            local features
            features="$(_cody_switch_features)"
            COMPREPLY=( $(compgen -W "$_CODY_SWITCH_SUBCOMMANDS $features" -- "$cur") )
            ;;
        *)
            local cmd="${COMP_WORDS[1]}"
            case "$cmd" in
                list|ls)
                    COMPREPLY=( $(compgen -W "--all -a" -- "$cur") )
                    ;;
                blank|-b)
                    COMPREPLY=( $(compgen -W "--branch --worktree --workflow" -- "$cur") )
                    ;;
                fork|-f)
                    if [[ "$cur" == -* ]]; then
                        COMPREPLY=( $(compgen -W "--without-docs --without-tasks --branch --worktree" -- "$cur") )
                    else
                        local features archived
                        features="$(_cody_switch_features)"
                        archived="$(_cody_switch_archived)"
                        COMPREPLY=( $(compgen -W "$features $archived" -- "$cur") )
                    fi
                    ;;
                merge|-m)
                    if [[ "$cur" == -* ]]; then
                        COMPREPLY=( $(compgen -W "--delete" -- "$cur") )
                    elif [ "$prev" = "into" ]; then
                        local features
                        features="$(_cody_switch_features)"
                        COMPREPLY=( $(compgen -W "$features" -- "$cur") )
                    else
                        local features archived
                        features="$(_cody_switch_features)"
                        archived="$(_cody_switch_archived)"
                        COMPREPLY=( $(compgen -W "$features $archived into --delete" -- "$cur") )
                    fi
                    ;;
                promote-lesson|promote)
                    COMPREPLY=( $(compgen -W "--user --project" -- "$cur") )
                    ;;
                install)
                    if [ "$prev" = "--skill" ]; then
                        local script_dir extras=""
                        script_dir="$(readlink -f "$(command -v cody-switch)" 2>/dev/null || echo "")"
                        script_dir="$(dirname "$script_dir")"
                        if [ -d "$script_dir/global-skills-extra" ]; then
                            for d in "$script_dir/global-skills-extra"/*/; do
                                [ -d "$d" ] && [ -f "$d/SKILL.md" ] && extras="$extras $(basename "$d")"
                            done
                        fi
                        COMPREPLY=( $(compgen -W "$extras" -- "$cur") )
                    else
                        COMPREPLY=( $(compgen -W "--force --extras --skill" -- "$cur") )
                    fi
                    ;;
                prompt)
                    local prompts
                    prompts="$(_cody_switch_prompts)"
                    COMPREPLY=( $(compgen -W "list $prompts" -- "$cur") )
                    ;;
                init)
                    if [[ "$cur" == -* ]]; then
                        COMPREPLY=( $(compgen -W "--template --force" -- "$cur") )
                    elif [ "$prev" = "--template" ]; then
                        COMPREPLY=( $(compgen -W "node typescript python go rust java ruby php c-cpp make generic" -- "$cur") )
                    fi
                    ;;
                peek|show|cat|archive|delete|rm)
                    local features
                    features="$(_cody_switch_features)"
                    COMPREPLY=( $(compgen -W "$features" -- "$cur") )
                    ;;
                unarchive)
                    local archived
                    archived="$(_cody_switch_archived)"
                    COMPREPLY=( $(compgen -W "$archived" -- "$cur") )
                    ;;
                doctor|health|diagnose)
                    COMPREPLY=( $(compgen -W "--fix" -- "$cur") )
                    ;;
                sync|open)
                    local features
                    features="$(_cody_switch_features)"
                    COMPREPLY=( $(compgen -W "$features" -- "$cur") )
                    ;;
                *)
                    # Default switch context: offer --with and feature names
                    if [[ "$cur" == -* ]]; then
                        COMPREPLY=( $(compgen -W "--with --json --output=json" -- "$cur") )
                    else
                        local features
                        features="$(_cody_switch_features)"
                        COMPREPLY=( $(compgen -W "$features" -- "$cur") )
                    fi
                    ;;
            esac
            ;;
    esac
}

complete -o default -F _cody_switch_bash cody-switch

# --- Zsh completion ---

elif type compdef &>/dev/null; then

_cody_switch_zsh() {
    local -a features subcommands archived

    # Populate features
    features=("${(@f)$(_cody_switch_features)}")

    subcommands=(
        'list:List features with status indicators'
        'current:Show active feature and branch'
        'new:Save current AGENTS.md as named context'
        'blank:Create blank feature context'
        'fork:Fork from an existing feature'
        'peek:View a feature'\''s AGENTS.md without switching'
        'archive:Archive a feature'
        'unarchive:Restore an archived feature'
        'delete:Delete a feature and all artifacts'
        'merge:Merge a feature into another'
        'pin-session:Pin current session as checkpoint'
        'promote-lesson:Promote lessons to global file'
        'promote-audit:Audit all features and bulk-promote lessons'
        'init:Initialize project for AI-assisted development'
        'doctor:Diagnose feature health'
        'save:Save active feature context to storage'
        'sync:Sync worktree context back to storage'
        'open:Print path to worktree directory'
        'prompt:Print bundled prompt templates'
        'help:Show help text'
    )

    # Check if --with is already in the line
    local has_with=0
    local i
    for ((i=2; i < CURRENT; i++)); do
        if [[ "${words[$i]}" == "--with" || "${words[$i]}" == --with=* ]]; then
            has_with=1
            break
        fi
    done

    # After --with: complete feature names
    if [[ "${words[$((CURRENT-1))]}" == "--with" ]] || [ "$has_with" -eq 1 ]; then
        _describe 'feature' features
        return
    fi

    case $CURRENT in
        2)
            # First argument: subcommands + feature names for direct switching
            _describe 'command' subcommands
            _describe 'feature' features
            ;;
        *)
            local cmd="${words[2]}"
            case "$cmd" in
                list|ls)
                    _arguments '--all[Include archived features]' '-a[Include archived features]'
                    ;;
                blank|-b)
                    _arguments \
                        '--branch[Also create and checkout a git branch]' \
                        '--worktree[Create with dedicated git worktree]' \
                        '--workflow[Append workflow.md template]'
                    ;;
                fork|-f)
                    archived=("${(@f)$(_cody_switch_archived)}")
                    _arguments \
                        '--without-docs[Skip copying docs folder]' \
                        '--without-tasks[Create fresh tasks scaffold instead of copying]' \
                        '--branch[Also create and checkout a git branch]' \
                        '--worktree[Create with dedicated git worktree]' \
                        '*:feature:_describe "feature" features -- archived'
                    ;;
                merge|-m)
                    archived=("${(@f)$(_cody_switch_archived)}")
                    _arguments \
                        '--delete[Delete source instead of archiving]' \
                        '*:feature:_describe "feature" features -- archived'
                    ;;
                promote-lesson|promote)
                    _arguments \
                        '--user[Promote to user-level lessons file]' \
                        '--project[Promote to project-level lessons file]'
                    ;;
                install)
                    _arguments \
                        '--force[Overwrite existing config, commands, and skills]' \
                        '--extras[Also install optional extra skills]' \
                        '--skill[Install a specific extra skill by name]:skill:->extra_skill'
                    if [[ "$state" == "extra_skill" ]]; then
                        local -a extra_skills
                        local script_dir
                        script_dir="$(readlink -f "$(command -v cody-switch)" 2>/dev/null || echo "")"
                        script_dir="$(dirname "$script_dir")"
                        if [ -d "$script_dir/global-skills-extra" ]; then
                            for d in "$script_dir/global-skills-extra"/*/; do
                                [ -d "$d" ] && [ -f "$d/SKILL.md" ] && extra_skills+=("$(basename "$d")")
                            done
                        fi
                        _describe 'extra skill' extra_skills
                    fi
                    ;;
                prompt)
                    local prompts
                    prompts=("${(@f)$(_cody_switch_prompts)}")
                    _describe 'prompt' prompts
                    ;;
                init)
                    _arguments \
                        '--template[Use specific template]:template:(node typescript python go rust java ruby php c-cpp make generic)' \
                        '--force[Reinitialize (backs up existing AGENTS.md)]'
                    ;;
                peek|show|cat|archive|delete|rm)
                    _describe 'feature' features
                    ;;
                unarchive)
                    archived=("${(@f)$(_cody_switch_archived)}")
                    _describe 'archived feature' archived
                    ;;
                sync|open)
                    _describe 'feature' features
                    ;;
                *)
                    # Direct switch: offer --with flag and features
                    _arguments '--with[Add reference context]:feature:->ref_feature'
                    if [[ "$state" == "ref_feature" ]]; then
                        _describe 'feature' features
                    else
                        _describe 'feature' features
                    fi
                    ;;
            esac
            ;;
    esac
}

compdef _cody_switch_zsh cody-switch
# When autoloaded via fpath, run now to produce completions
_cody_switch_zsh "$@"

fi
