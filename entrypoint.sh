#!/bin/bash
set -euo pipefail

git config --global --add safe.directory /repo

git config --global user.name "${GIT_AUTHOR_NAME:-github-actions[bot]}"
git config --global user.email "${GIT_AUTHOR_EMAIL:-github-actions[bot]@users.noreply.github.com}"

if [ -n "${GITHUB_TOKEN:-}" ]; then
  git remote set-url origin \
    https://x-access-token:${GITHUB_TOKEN}@github.com/${GITHUB_REPOSITORY}.git
fi

exec monotrack "$@"
