#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}"

ISSUE_LABEL="${ISSUE_LABEL:-autocodex}"
ISSUE_POLL_INTERVAL="${ISSUE_POLL_INTERVAL:-60}"
PR_POLL_INTERVAL="${PR_POLL_INTERVAL:-60}"
TRIGGER_PREFIX="${TRIGGER_PREFIX:-@codex}"

cleanup() {
  trap - EXIT INT TERM
  [[ -n "${ISSUE_PID:-}" ]] && kill "${ISSUE_PID}" >/dev/null 2>&1 || true
  [[ -n "${PR_PID:-}" ]] && kill "${PR_PID}" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

echo "starting issue worker..."
LABEL="${ISSUE_LABEL}" POLL_INTERVAL="${ISSUE_POLL_INTERVAL}" ./scripts/issue_worker.sh &
ISSUE_PID=$!

echo "starting pr worker..."
POLL_INTERVAL="${PR_POLL_INTERVAL}" TRIGGER_PREFIX="${TRIGGER_PREFIX}" ./scripts/pr_worker.sh &
PR_PID=$!

echo "workers started:"
echo "  issue_worker pid=${ISSUE_PID} label=${ISSUE_LABEL} interval=${ISSUE_POLL_INTERVAL}s"
echo "  pr_worker    pid=${PR_PID} prefix=${TRIGGER_PREFIX} interval=${PR_POLL_INTERVAL}s"
echo "press Ctrl+C to stop both."

while true; do
  if ! kill -0 "${ISSUE_PID}" >/dev/null 2>&1; then
    echo "issue worker exited; shutting down both."
    break
  fi
  if ! kill -0 "${PR_PID}" >/dev/null 2>&1; then
    echo "pr worker exited; shutting down both."
    break
  fi
  sleep 1
done
