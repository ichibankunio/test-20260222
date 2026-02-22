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
WORKTREE_LOCK_DIR="${STATE_DIR}/worktree-lock"
PROCESSED_DIR="${STATE_DIR}/processed"
STARTED_DIR="${STATE_DIR}/issue-started"
LOG_DIR="${STATE_DIR}/logs"

mkdir -p "${PROCESSED_DIR}" "${STARTED_DIR}" "${LOG_DIR}"

if ! mkdir "${LOCK_DIR}" 2>/dev/null; then
  echo "issue worker lock exists (${LOCK_DIR}); another worker is running."
  exit 0
fi

cleanup() {
  rmdir "${WORKTREE_LOCK_DIR}" >/dev/null 2>&1 || true
  rmdir "${LOCK_DIR}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

log() {
  echo "[issue_worker] $*"
}

acquire_worktree_lock() {
  while ! mkdir "${WORKTREE_LOCK_DIR}" 2>/dev/null; do
    log "worktree lock busy; waiting..."
    sleep 2
  done
}

release_worktree_lock() {
  rmdir "${WORKTREE_LOCK_DIR}" >/dev/null 2>&1 || true
}

log "starting for ${REPO} (label=${LABEL}, interval=${POLL_INTERVAL}s)"

while true; do
  handled_any=false
  acquire_worktree_lock

  # Always stay on main before scanning the next issue.
  if [[ -n "$(git status --porcelain)" ]]; then
    log "worktree is dirty; skip polling until clean."
    release_worktree_lock
    sleep "${POLL_INTERVAL}"
    continue
  fi
  if [[ "$(git branch --show-current)" != "main" ]]; then
    if ! git checkout main >/dev/null 2>&1; then
      log "failed to checkout main; retry later."
      release_worktree_lock
      sleep "${POLL_INTERVAL}"
      continue
    fi
  fi
  if ! git pull --ff-only origin main >/dev/null 2>&1; then
    log "failed to update main; retry later."
    release_worktree_lock
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
    STARTED_FILE="${STARTED_DIR}/${NEXT_ISSUE}.started"
    log "processing issue #${NEXT_ISSUE}"
    if [[ ! -f "${STARTED_FILE}" ]]; then
      if gh issue comment "${NEXT_ISSUE}" --repo "${REPO}" --body "Codex started working on this issue now." >/dev/null 2>&1; then
        touch "${STARTED_FILE}"
      else
        log "failed to post start comment for issue #${NEXT_ISSUE}"
      fi
    fi
    set +e
    "${RUN_ISSUE_SCRIPT}" "${NEXT_ISSUE}" 2>&1 \
      | tee "${LOG_FILE}" \
      | while IFS= read -r line; do log "issue #${NEXT_ISSUE} | ${line}"; done
    run_status=${PIPESTATUS[0]}
    set -e
    if [[ "${run_status}" -eq 0 ]]; then
      touch "${PROCESSED_DIR}/${NEXT_ISSUE}.done"
      log "issue #${NEXT_ISSUE} completed"
    else
      if [[ "${run_status}" -eq 20 ]]; then
        log "issue #${NEXT_ISSUE} waiting for reviewer response"
      else
        log "issue #${NEXT_ISSUE} failed. see ${LOG_FILE}"
        # Best-effort return to main after failure.
        if [[ -z "$(git status --porcelain)" ]]; then
          git checkout main >/dev/null 2>&1 || true
        fi
      fi
    fi
  fi
  if [[ "${handled_any}" != "true" ]]; then
    log "poll: no new autocodex issues"
  fi
  release_worktree_lock

  sleep "${POLL_INTERVAL}"
done
