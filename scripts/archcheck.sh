#!/usr/bin/env bash
# Structural checks for the marIA architecture rules in CLAUDE.md.
# Heuristic, grep-based, no Go toolchain required. Each failure prints file:line and the rule.
# To allow a justified exception, put "archcheck:allow <reason>" on the offending line.
# ARCHCHECK_ROOT overrides the directory to check (used by scripts/test-harness.sh with fixtures).
set -uo pipefail

root="${ARCHCHECK_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
cd "$root" || exit 1

services="gateway conversation reminders outbound relay"
fail=0

report() { # file:line, rule
  echo "archcheck: $1: $2"
  fail=1
}

go_files() { # dir
  [[ -d "$1" ]] || return 0
  find "$1" -type f -name '*.go' -not -path '*/vendor/*' -not -path '*/testdata/*'
}

table_owner() {
  case "$1" in
    users | conversations) echo conversation ;;
    reminders) echo reminders ;;
    outbound) echo outbound ;;
    relay) echo relay ;;
  esac
}

for svc in $services; do
  for dir in "internal/$svc" "cmd/$svc"; do
    while IFS= read -r f; do
      [[ -n "$f" ]] || continue

      # Rule 1: a service never imports another service's internal package.
      while IFS=: read -r line content; do
        [[ "$content" == *archcheck:allow* ]] && continue
        other=$(sed -E 's#.*/internal/([a-z]+).*#\1#' <<<"$content")
        if [[ "$other" != "$svc" ]]; then
          report "$f:$line" "internal/$svc must not import internal/$other (use pkg/events or the reminders API)"
        fi
      done < <(grep -nE '"[^"]*/internal/[a-z]+(/[^"]*)?"' "$f")

      # Rule 2: only outbound imports the Meta sender (pkg/meta root). pkg/meta/webhook is free to use.
      if [[ "$svc" != "outbound" ]]; then
        while IFS=: read -r line content; do
          [[ "$content" == *archcheck:allow* ]] && continue
          report "$f:$line" "only outbound may send to Meta (import of pkg/meta)"
        done < <(grep -nE '"[^"]*/pkg/meta"' "$f")
      fi

      # Rule 3: each service only touches its own DynamoDB tables.
      while IFS=: read -r line content; do
        [[ "$content" == *archcheck:allow* ]] && continue
        table=$(sed -E 's#.*"maria-(users|conversations|reminders|outbound|relay)[-"].*#\1#' <<<"$content")
        owner=$(table_owner "$table")
        if [[ -n "$owner" && "$owner" != "$svc" ]]; then
          report "$f:$line" "table maria-$table belongs to $owner, not $svc"
        fi
      done < <(grep -nE '"maria-(users|conversations|reminders|outbound|relay)[-"]' "$f")
    done < <(go_files "$dir")
  done
done

# Rule 4: Meta Graph API host only inside pkg/meta.
while IFS=: read -r f line content; do
  [[ "$content" == *archcheck:allow* ]] && continue
  report "$f:$line" "graph.facebook.com must only appear in pkg/meta"
done < <(grep -rnE 'graph\.facebook\.com' --include='*.go' --exclude-dir=testdata --exclude-dir=vendor . 2>/dev/null | grep -v '^\./pkg/meta/')

# Rule 5: business logic uses an injected clock, not time.Now().
while IFS= read -r f; do
  [[ -n "$f" ]] || continue
  case "$f" in *_test.go | */clock.go | */main.go) continue ;; esac
  while IFS=: read -r line content; do
    [[ "$content" == *archcheck:allow* ]] && continue
    report "$f:$line" "use an injected clock instead of time.Now()"
  done < <(grep -nE 'time\.Now\(\)' "$f")
done < <(go_files internal)

# Rule 6: never log message text or phone numbers without logging.Mask().
while IFS=: read -r f line content; do
  [[ "$content" == *archcheck:allow* || "$content" == *Mask\(* ]] && continue
  report "$f:$line" "log attribute looks like message text or phone; wrap it with logging.Mask()"
done < <(grep -rnE '\.(Info|Warn|Error|Debug|Log)(Context)?\(.*"(phone|to|from|text|body|message|caption)"' --include='*.go' --exclude-dir=testdata internal cmd pkg 2>/dev/null | grep -v '_test\.go:')

# Rule 7: configuration is read from the environment only in internal/<svc>/config.go.
# (forbidigo misses package-level variables; this grep does not.)
while IFS=: read -r f line content; do
  [[ "$content" == *archcheck:allow* ]] && continue
  [[ "$f" =~ ^internal/[a-z]+/config\.go$ ]] && continue
  report "$f:$line" "read environment variables only in internal/<svc>/config.go"
done < <(grep -rnE 'os\.(Getenv|LookupEnv|Environ)\(' --include='*.go' --exclude-dir=testdata internal cmd pkg 2>/dev/null | grep -v '_test\.go:')

if [[ $fail -ne 0 ]]; then
  echo "archcheck: FAILED (see CLAUDE.md, Regras de arquitetura)"
  exit 1
fi
echo "archcheck: ok"
