#!/bin/bash
set -euo pipefail

COMMAND=${1:-"tag bump"}
BASE=${2:-}
HEAD=${3:-}
ARGS=${4:-}
CONFIG=${5:-}

# Build command
CMD=(monotrack "$COMMAND")
if [ -n "$ARGS" ]; then
  read -r -a ARGS_ARRAY <<< "$ARGS"
  CMD+=("${ARGS_ARRAY[@]}")
fi
CMD+=(--base "$BASE" --head "$HEAD")

if [ -n "$CONFIG" ]; then
  CMD+=(--config "$CONFIG")
fi

# Run Monotrack
OUTPUT="$("${CMD[@]}")"
echo "$OUTPUT"
