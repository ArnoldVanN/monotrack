#!/bin/bash
set -euo pipefail

COMMAND=${1:-"tag bump"}
BASE=${2:-}
HEAD=${3:-}
ARGS=${4:-}
CONFIG=${5:-}

CMD=(monotrack)

git config --global --add safe.directory /repo

read -r -a COMMAND_ARRAY <<< "$COMMAND"
CMD+=("${COMMAND_ARRAY[@]}")

if [ -n "$ARGS" ]; then
  read -r -a ARGS_ARRAY <<< "$ARGS"
  CMD+=("${ARGS_ARRAY[@]}")
fi
CMD+=(--base "$BASE" --head "$HEAD")

if [ -n "$CONFIG" ]; then
  CMD+=(--config "$CONFIG")
fi

{
  printf 'Executing command: '
  printf '%q ' "${CMD[@]}"
  printf '\n'
} >&2

exec "${CMD[@]}"
