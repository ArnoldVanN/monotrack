#!/bin/bash
set -euo pipefail

COMMAND=${1:-"tag bump"}
BASE=${2:-}
HEAD=${3:-}
ARGS=${4:-}
CONFIG=${5:-}

# Determine base and head SHAs if not provided
if [ -z "$BASE" ] || [ -z "$HEAD" ]; then
  case "$GITHUB_EVENT_NAME" in
    push)
      BASE="${BASE:-$(jq -r '.before // empty' < "$GITHUB_EVENT_PATH")}"
      HEAD="${HEAD:-$(jq -r '.after // empty' < "$GITHUB_EVENT_PATH")}"
      ;;
    pull_request)
      BASE="${BASE:-$(jq -r '.pull_request.base.sha' < "$GITHUB_EVENT_PATH")}"
      HEAD="${HEAD:-$(jq -r '.pull_request.head.sha' < "$GITHUB_EVENT_PATH")}"
      ;;
    workflow_dispatch|schedule)
      # User must provide explicitly
      ;;
  esac
fi

HEAD="${HEAD:-$GITHUB_SHA}"

if [ -z "$BASE" ]; then
  echo "::warning::Unable to determine base SHA."
fi

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
