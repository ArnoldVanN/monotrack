#!/bin/bash
set -euo pipefail

git config --global --add safe.directory /repo

exec monotrack "$@"
