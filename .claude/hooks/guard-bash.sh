#!/usr/bin/env bash
# PreToolUse hook for Bash: blocks actions that CLAUDE.md reserves for the pipeline.
# Exit code 2 blocks the call and sends stderr back to the agent.
set -euo pipefail

cmd=$(jq -r '.tool_input.command // ""')

block() {
  echo "Bloqueado pelo harness: $1. Veja CLAUDE.md (Esteira) e HARNESS.md." >&2
  exit 2
}

# Terraform: only fmt, validate, init, plan, show, output, providers are allowed locally.
if grep -Eq '(^|[[:space:];&|])terraform[[:space:]]+(-chdir=[^[:space:]]+[[:space:]]+)?(apply|destroy|import|taint|untaint|state[[:space:]]+(rm|mv|push|replace-provider)|force-unlock)' <<<"$cmd"; then
  block "terraform que altera estado só roda na esteira"
fi

# Publishing images.
if grep -Eq '(^|[[:space:];&|])docker[[:space:]]+(image[[:space:]]+)?push' <<<"$cmd"; then
  block "publicar imagem só pela esteira"
fi

# Mutating AWS CLI calls and secret reads.
if grep -Eq '(^|[[:space:];&|])aws[[:space:]]+[a-z0-9-]+[[:space:]]+(create|delete|put|update|terminate|remove|batch-delete|tag|untag|run|start|stop|send|publish)-?' <<<"$cmd"; then
  block "chamadas da AWS CLI que alteram recursos não são permitidas fora da esteira"
fi
if grep -Eq '(^|[[:space:];&|])aws[[:space:]]+s3[[:space:]]+(cp|mv|rm|sync|rb|mb)' <<<"$cmd"; then
  block "escrita no S3 não é permitida fora da esteira"
fi
if grep -Eq 'secretsmanager[[:space:]]+get-secret-value' <<<"$cmd"; then
  block "segredos não são lidos pelo agente"
fi

# Git Flow (ADR 0002): work happens on feature/*, release/x.y.z and hotfix/x.y.z; develop and main change only through PRs.
branch=$(git -C "${CLAUDE_PROJECT_DIR:-.}" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")

if [[ "$branch" == "main" || "$branch" == "develop" ]] &&
  grep -Eq '(^|[[:space:];&|])git[[:space:]]+(commit|merge|cherry-pick|revert|am|rebase)([[:space:]]|$)' <<<"$cmd"; then
  block "você está na $branch; use feature/* (da develop), release/x.y.z (da develop) ou hotfix/x.y.z (da main). Back-merge é por PR"
fi

# Merges into develop and main are done by the pipeline, never by hand.
if grep -Eq '(^|[[:space:];&|])gh[[:space:]]+pr[[:space:]]+merge' <<<"$cmd"; then
  block "merge de PR é sempre feito pelo administrador"
fi

# Push: only feature/* branches, never forced.
while IFS= read -r segment; do
  [[ -n "$segment" ]] || continue
  if grep -Eq '(--force|--force-with-lease|--mirror|--all|--delete|(^|[[:space:]])-f([[:space:]]|$)|(^|[[:space:]])-d([[:space:]]|$))' <<<"$segment"; then
    block "force push, --mirror, --all e remoção de branch remota não são permitidos"
  fi
  # Positional args after "push": first is the remote, the rest are refspecs.
  read -r -a args <<<"$(sed -E 's/^.*git[[:space:]]+push//' <<<"$segment")"
  positional=()
  for a in "${args[@]+"${args[@]}"}"; do
    [[ "$a" =~ ^[0-9]*[\<\>] ]] && break # shell redirection (2>&1, >log): no more git args
    [[ "$a" == -* ]] || positional+=("$a")
  done
  refs=("${positional[@]:1}")
  [[ ${#refs[@]} -gt 0 ]] || refs=("$branch")
  for ref in "${refs[@]}"; do
    target="${ref##*:}"
    target="${target#refs/heads/}"
    [[ "$target" == "HEAD" ]] && target="$branch"
    if ! [[ "$target" == feature/?* || "$target" =~ ^(release|hotfix)/[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      block "push só em feature/*, release/x.y.z ou hotfix/x.y.z (alvo: ${target:-?}); develop e main mudam só por PR"
    fi
  done
done < <(grep -oE '(^|[;&|][[:space:]]*)git[[:space:]]+push[^;&|]*' <<<"$cmd" | sed -E 's/^[;&|][[:space:]]*//')

exit 0
