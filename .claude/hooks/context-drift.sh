#!/usr/bin/env bash
# Fires after an edit to a context-affecting path (see CLAUDE.md, "Keeping this
# file true"). Reminds the agent to reconcile the docs that describe that code.
#
# Contract: PostToolUse hook. Exit 2 surfaces stderr to the agent without
# blocking anything — the tool has already run. Any other outcome is silent.
set -uo pipefail

payload=$(cat)

# The hook payload is JSON. Prefer jq; fall back to a grep so a missing jq
# degrades to "no reminder" rather than a spurious one on every edit.
if command -v jq >/dev/null 2>&1; then
  file=$(printf '%s' "$payload" | jq -r '.tool_input.file_path // empty')
else
  file=$(printf '%s' "$payload" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | cut -d'"' -f4)
fi

[ -n "$file" ] || exit 0

root=${CLAUDE_PROJECT_DIR:-}
[ -n "$root" ] || exit 0

# Match on the path relative to the project root. Matching on a trailing glob
# would fire for api/internal/transport/http/v1/events/, a directory that merely
# shares a name with the contracts module.
case "$file" in
  "$root"/*) rel=${file#"$root"/} ;;
  *) exit 0 ;;
esac

# A test does not describe the design; it checks it. And editing the docs is the
# fix, not the trigger.
case "$rel" in
  *_test.go|CLAUDE.md|.claude/*) exit 0 ;;
esac

case "$rel" in
  events/*.go \
  |outbox/*.go|outbox/*/*.go \
  |subscribers/internal/handler/*.go \
  |platform/amqp/*.go|platform/postgres/*.go \
  |go.work|*/go.mod \
  |*/migrations/*.sql)
    ;;
  *) exit 0 ;;
esac

cat >&2 <<EOF
Context-affecting change: $rel

This path can invalidate CLAUDE.md, .claude/agents/*.md, or .claude/skills/*/SKILL.md.
Before reporting this work complete, run the sync-context skill, or confirm the docs
still describe reality:

  rg -n -f .claude/retired-symbols.txt CLAUDE.md .claude/agents .claude/skills

Empty output is the pass condition. Retired a symbol? Add it to that file.
EOF
exit 2
