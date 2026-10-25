#!/usr/bin/env bash
# PreToolUse hook for Read/Edit/Write: keeps the agent away from secrets and Terraform state.
set -euo pipefail

path=$(jq -r '.tool_input.file_path // ""')
name=$(basename "$path")

case "$name" in
  .env | .env.* | *.tfstate | *.tfstate.* | *.pem | *.key | *.tfvars.secret)
    echo "Bloqueado pelo harness: $name pode conter segredos ou state. Segredos ficam no Secrets Manager." >&2
    exit 2
    ;;
esac

exit 0
