#!/usr/bin/env bash
set -euo pipefail

if ! command -v gh >/dev/null 2>&1; then
  echo "gh command is required." >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUN_ISSUE_SCRIPT="${ROOT_DIR}/scripts/run_issue.sh"
REPO="${REPO:-$(gh repo view --json nameWithOwner --jq .nameWithOwner)}"
LABEL="${LABEL:-autocodex}"
POLL_INTERVAL="${POLL_INTERVAL:-60}"
STATE_DIR="${STATE_DIR:-${ROOT_DIR}/.codex-worker}"
LOCK_DIR="${STATE_DIR}/lock"
PROCESSED_DIR="${STATE_DIR}/processed"
LOG_DIR="${STATE_DIR}/logs"

mkdir -p "${PROCESSED_DIR}" "${LOG_DIR}"

if ! mkdir "${LOCK_DIR}" 2>/dev/null; then
  echo "issue worker lock exists (${LOCK_DIR}); another worker is running."
  exit 0
fi

cleanup() {
  rmdir "${LOCK_DIR}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "starting issue worker for ${REPO} (label=${LABEL}, interval=${POLL_INTERVAL}s)"

while true; do
  handled_any=false

  # Always stay on main before scanning the next issue.
  if [[ -n "$(git status --porcelain)" ]]; then
    echo "worktree is dirty; skip polling until clean." >&2
    sleep "${POLL_INTERVAL}"
    continue
  fi
  if [[ "$(git branch --show-current)" != "main" ]]; then
    if ! git checkout main >/dev/null 2>&1; then
      echo "failed to checkout main; retry later." >&2
      sleep "${POLL_INTERVAL}"
      continue
    fi
  fi
  if ! git pull --ff-only origin main >/dev/null 2>&1; then
    echo "failed to update main; retry later." >&2
    sleep "${POLL_INTERVAL}"
    continue
  fi

  NEXT_ISSUE="$(gh issue list \
    --repo "${REPO}" \
    --state open \
    --label "${LABEL}" \
    --limit 100 \
    --json number \
    --jq 'sort_by(.number) | .[].number' | while read -r n; do
      [[ -z "${n}" ]] && continue
      [[ -f "${PROCESSED_DIR}/${n}.done" ]] && continue
      echo "${n}"
      break
    done)"

  if [[ -n "${NEXT_ISSUE}" ]]; then
    handled_any=true
    LOG_FILE="${LOG_DIR}/issue-${NEXT_ISSUE}-$(date +%Y%m%d%H%M%S).log"
    echo "processing issue #${NEXT_ISSUE}"
    if "${RUN_ISSUE_SCRIPT}" "${NEXT_ISSUE}" >"${LOG_FILE}" 2>&1; then
      touch "${PROCESSED_DIR}/${NEXT_ISSUE}.done"
      echo "issue #${NEXT_ISSUE} completed"
    else
      run_status=$?
      if [[ "${run_status}" -eq 20 ]]; then
        echo "issue #${NEXT_ISSUE} waiting for reviewer response"
      else
        echo "issue #${NEXT_ISSUE} failed. see ${LOG_FILE}" >&2
        # Best-effort return to main after failure.
        if [[ -z "$(git status --porcelain)" ]]; then
          git checkout main >/dev/null 2>&1 || true
        fi
      fi
    fi
  fi
  if [[ "${handled_any}" != "true" ]]; then
    echo "poll: no new autocodex issues"
  fi

  sleep "${POLL_INTERVAL}"
done
