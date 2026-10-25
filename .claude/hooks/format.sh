#!/usr/bin/env bash
# PostToolUse hook for Edit/Write: formats the touched file so the agent never fights the linter on style.
set -uo pipefail

path=$(jq -r '.tool_input.file_path // ""')
[[ -f "$path" ]] || exit 0
# Fixtures are intentionally unformatted or "broken" (unused imports); never touch them.
[[ "$path" == */testdata/* ]] && exit 0

case "$path" in
  *.go)
    if command -v goimports >/dev/null 2>&1; then
      goimports -w "$path"
    elif command -v gofmt >/dev/null 2>&1; then
      gofmt -w "$path"
    fi
    ;;
  *.tf)
    command -v terraform >/dev/null 2>&1 && terraform fmt "$path" >/dev/null
    ;;
esac

exit 0
