#!/usr/bin/env bash
# Tests for the harness itself: Claude Code hooks and scripts/archcheck.sh.
# Run with `make harness-test`. Needs bash, git and jq; no network, no Go toolchain.
set -uo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
hooks="$root/.claude/hooks"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

pass=0
fail=0
check() { # name, expected, got
  if [[ "$2" == "$3" ]]; then
    pass=$((pass + 1))
  else
    fail=$((fail + 1))
    echo "FAIL: $1 (expected $2, got $3)"
  fi
}

# Runs guard-bash.sh with a command, inside the repo given by CLAUDE_PROJECT_DIR.
bash_hook() { # expected-exit, command
  local got
  got=$(jq -n --arg c "$2" '{tool_input:{command:$c}}' | "$hooks/guard-bash.sh" >/dev/null 2>&1; echo $?)
  check "guard-bash [$(git -C "$CLAUDE_PROJECT_DIR" branch --show-current)] $2" "$1" "$got"
}

file_hook() { # expected-exit, path
  local got
  got=$(jq -n --arg p "$2" '{tool_input:{file_path:$p}}' | "$hooks/guard-files.sh" >/dev/null 2>&1; echo $?)
  check "guard-files $2" "$1" "$got"
}

# --- Throwaway repo: main with develop, like the real one.
repo="$tmp/repo"
git init -q -b main "$repo"
git -C "$repo" -c user.name=t -c user.email=t@t commit -q --allow-empty -m init
git -C "$repo" branch develop
export CLAUDE_PROJECT_DIR="$repo"

# --- guard-bash: actions reserved for the pipeline.
git -C "$repo" switch -q -c feature/x develop
bash_hook 2 "terraform apply"
bash_hook 2 "terraform -chdir=infra/envs/prod apply -auto-approve"
bash_hook 2 "cd infra && terraform destroy"
bash_hook 0 "terraform plan"
bash_hook 0 "terraform validate"
bash_hook 2 "docker push x"
bash_hook 0 "docker build ."
bash_hook 2 "aws dynamodb put-item --table x"
bash_hook 0 "aws sts get-caller-identity"
bash_hook 2 "aws s3 cp a s3://b"
bash_hook 0 "aws s3 ls"
bash_hook 2 "aws secretsmanager get-secret-value --secret-id x"
bash_hook 0 "go test ./..."
bash_hook 0 "git status"

# --- guard-bash: Git Flow (ADR 0002).
bash_hook 0 "git push -u origin feature/x"
bash_hook 0 "git push"
bash_hook 0 "git push origin HEAD"
bash_hook 0 "git add . && git commit -m 'x' && git push -u origin feature/x"
bash_hook 0 "git merge develop"
bash_hook 0 "git push -u origin feature/x 2>&1"
bash_hook 0 "git push -u origin feature/x > /tmp/log 2>&1"
bash_hook 0 "git log --oneline -2 && git push -u origin feature/x 2>&1"
bash_hook 0 "git push origin release/1.2.0"
bash_hook 0 "git push origin hotfix/1.2.1"
bash_hook 2 "git push origin release/1.2"
bash_hook 2 "git push origin release/next"
bash_hook 2 "git push origin hotfix/urgent"
bash_hook 2 "git push origin feature/"
bash_hook 2 "git push origin develop"
bash_hook 2 "git push origin develop 2>&1"
bash_hook 2 "git push origin main"
bash_hook 2 "git push origin feature/x:develop"
bash_hook 2 "git push origin HEAD:refs/heads/main"
bash_hook 2 "git push --force origin feature/x"
bash_hook 2 "git push -f"
bash_hook 2 "git push origin --delete feature/old"
bash_hook 2 "git push --all"
bash_hook 2 "gh pr merge 12 --merge"
bash_hook 0 "gh pr create --base develop"

git -C "$repo" switch -q -c release/1.2.0 develop
bash_hook 0 "git commit -m 'fix: staging bug'"
bash_hook 0 "git push -u origin release/1.2.0"
git -C "$repo" switch -q -c hotfix/1.2.1 main
bash_hook 0 "git push"

git -C "$repo" switch -q develop
bash_hook 2 "git push"
bash_hook 2 "git commit -m x"
bash_hook 2 "git merge main"
bash_hook 0 "git pull"
bash_hook 0 "git switch -c release/1.3.0 develop"

git -C "$repo" switch -q main
bash_hook 2 "git merge release/1.2.0"
bash_hook 2 "git push"
bash_hook 2 "git push origin feature/x:main"
bash_hook 0 "git push origin feature/x"
bash_hook 0 "git switch -c hotfix/1.2.2 main"

# --- guard-files: secrets and state.
file_hook 2 /x/.env
file_hook 2 /x/.env.prod
file_hook 2 /x/terraform.tfstate
file_hook 2 /x/terraform.tfstate.backup
file_hook 2 /x/key.pem
file_hook 0 /x/main.go
file_hook 2 /x/.env.example # conservative on purpose: config is documented in docs/spec, not in .env files

# --- session-context: back-merge warning.
remote="$tmp/remote.git"
git init -q --bare "$remote"
git -C "$repo" remote add origin "$remote"
git -C "$repo" push -q origin main develop
git -C "$repo" fetch -q origin
out=$(cd "$repo" && "$hooks/session-context.sh")
check "session-context: no warning when develop has main" "0" "$(grep -c 'ATENÇÃO' <<<"$out")"
git -C "$repo" -c user.name=t -c user.email=t@t commit -q --allow-empty -m "release merge"
git -C "$repo" push -q origin main
git -C "$repo" fetch -q origin
out=$(cd "$repo" && "$hooks/session-context.sh")
check "session-context: warns when main is ahead of develop" "1" "$(grep -c 'ATENÇÃO' <<<"$out")"

# --- archcheck: every "want" line in the violations fixture is reported; the clean fixture passes.
fixtures="$root/scripts/testdata/archcheck"
out=$(ARCHCHECK_ROOT="$fixtures/violations" "$root/scripts/archcheck.sh")
check "archcheck violations: exit code" "1" "$?"
out=$(sed -E 's#^archcheck: \./#archcheck: #' <<<"$out")
wants=0
while IFS=: read -r file line _; do
  wants=$((wants + 1))
  rel="${file#"$fixtures/violations/"}"
  check "archcheck reports $rel:$line" "1" "$(grep -c "^archcheck: $rel:$line:" <<<"$out")"
done < <(grep -rn 'want:' "$fixtures/violations" --include='*.go')
check "archcheck reports exactly the wanted violations" "$wants" "$(grep -c '^archcheck: .*\.go:[0-9]*:' <<<"$out")"
ARCHCHECK_ROOT="$fixtures/clean" "$root/scripts/archcheck.sh" >/dev/null
check "archcheck clean fixture passes" "0" "$?"

echo "harness-test: $pass passed, $fail failed"
[[ $fail -eq 0 ]]
