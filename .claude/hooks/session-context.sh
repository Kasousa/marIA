#!/usr/bin/env bash
# SessionStart hook: prints a short orientation that is added to the agent's context.
set -uo pipefail

root="${CLAUDE_PROJECT_DIR:-.}"
cd "$root" || exit 0

echo "## Estado do harness"
echo "Branch: $(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo '?')"

# Git Flow: warn when main has commits that never came back to develop (missing back-merge).
if git rev-parse -q --verify origin/main >/dev/null && git rev-parse -q --verify origin/develop >/dev/null; then
  missing=$(git rev-list --count origin/develop..origin/main 2>/dev/null || echo 0)
  if [[ "$missing" -gt 0 ]]; then
    echo "ATENÇÃO: origin/main tem $missing commit(s) que não estão na develop. Falta o PR de back-merge main -> develop."
  fi
fi

echo
echo "Planos ativos:"
found=0
for f in docs/plans/active/*.md; do
  [[ -e "$f" ]] || continue
  found=1
  echo "- $f"
done
[[ $found -eq 1 ]] || echo "- nenhum (use /plan-phase)"

if [[ -f docs/features.json ]] && command -v jq >/dev/null 2>&1; then
  echo
  echo "Features por status:"
  jq -r '.features | group_by(.status) | map("- \(.[0].status): \(length)") | .[]' docs/features.json
  echo "Próximas pendentes:"
  jq -r '[.features[] | select(.status == "in_progress" or .status == "todo")][:3][] | "- \(.id) [\(.status)] \(.title)"' docs/features.json
fi

if [[ -f docs/progress.md ]]; then
  echo
  echo "Última entrada do docs/progress.md:"
  awk '/^## [0-9]{4}-/{n++} n==1' docs/progress.md
fi

exit 0
