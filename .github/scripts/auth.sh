#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────────────────────────
# .github/scripts/auth.sh
# Shared command-parsing and authorization gate for OpenCode workflows.
#
# Usage:
#   .github/scripts/auth.sh <comment_body>
#
# Outputs (via GITHUB_OUTPUT):
#   IS_OC_COMMAND=true|false
#   SUBCOMMAND=review|implement|task|discuss|none
#   TASK_ARGS=<text after the subcommand, if any>
# ─────────────────────────────────────────────────────────────────────

COMMENT_BODY="${1:-}"

# Normalize: trim leading/trailing whitespace
TRIMMED=$(echo "$COMMENT_BODY" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

IS_OC="false"
SUBCOMMAND="none"
TASK_ARGS=""

if [[ "$TRIMMED" =~ ^/oc ]]; then
  IS_OC="true"

  if [[ "$TRIMMED" =~ ^/oc[[:space:]]+implement[[:space:]]+(.*) ]]; then
    SUBCOMMAND="implement"
    TASK_ARGS="${BASH_REMATCH[1]}"
  elif [[ "$TRIMMED" =~ ^/oc[[:space:]]+task[[:space:]]+(.*) ]]; then
    SUBCOMMAND="task"
    TASK_ARGS="${BASH_REMATCH[1]}"
  elif [[ "$TRIMMED" =~ ^/oc[[:space:]]+task[[:space:]]*$ ]]; then
    SUBCOMMAND="task"
  elif [[ "$TRIMMED" =~ ^/oc[[:space:]]+review($|[[:space:]]) ]]; then
    SUBCOMMAND="review"
  elif [[ "$TRIMMED" =~ ^/oc[[:space:]]*$ ]] || [[ "$TRIMMED" =~ ^/oc$ ]]; then
    SUBCOMMAND="discuss"
  else
    # /oc with unrecognized subcommand — default to discuss
    SUBCOMMAND="discuss"
  fi
fi

echo "IS_OC_COMMAND=$IS_OC" >> "$GITHUB_OUTPUT"
echo "SUBCOMMAND=$SUBCOMMAND" >> "$GITHUB_OUTPUT"

# Multi-line args via heredoc delimiter
if [[ -n "$TASK_ARGS" ]]; then
  {
    echo "TASK_ARGS<<OCAUTH_EOF"
    echo "$TASK_ARGS"
    echo "OCAUTH_EOF"
  } >> "$GITHUB_OUTPUT"
else
  echo "TASK_ARGS=" >> "$GITHUB_OUTPUT"
fi
