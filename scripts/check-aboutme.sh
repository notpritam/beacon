#!/usr/bin/env bash
# ABOUTME: Verifies each given .go file has two `// ABOUTME:` comment lines near the top.
# ABOUTME: Used by the pre-commit gate; exits non-zero and lists offenders if any are missing.

if [ -z "$BASH_VERSION" ]; then exec bash "$0" "$@"; fi

bad=0
for f in "$@"; do
  case "$f" in *.go) ;; *) continue ;; esac
  [ -f "$f" ] || continue
  # Count ABOUTME comment lines within the first 6 lines (allows a //go:build constraint first).
  count="$(head -n 6 "$f" | grep -c '^// ABOUTME:')"
  if [ "$count" -lt 2 ]; then
    echo "  missing // ABOUTME: header → $f"
    bad=1
  fi
done

exit "$bad"
