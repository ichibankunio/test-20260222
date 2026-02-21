#!/usr/bin/env bash
set -euo pipefail

for cmd in gh codex git jq base64; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "${cmd} command is required." >&2
    exit 1
  fi
done

decode_base64() {
  if base64 --decode >/dev/null 2>&1 <<<""; then
    base64 --decode
  else
    base64 -D
  fi
}

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO="${REPO:-$(gh repo view --json nameWithOwner --jq .nameWithOwner)}"
POLL_INTERVAL="${POLL_INTERVAL:-60}"
TRIGGER_PREFIX="${TRIGGER_PREFIX:-@codex}"
STATE_DIR="${STATE_DIR:-${ROOT_DIR}/.codex-worker}"
LOCK_DIR="${STATE_DIR}/pr-lock"
WORKTREE_LOCK_DIR="${STATE_DIR}/worktree-lock"
PROCESSED_DIR="${STATE_DIR}/processed-comments"
LOG_DIR="${STATE_DIR}/logs"
BASE_BRANCH="${BASE_BRANCH:-main}"

mkdir -p "${PROCESSED_DIR}" "${LOG_DIR}"

if ! mkdir "${LOCK_DIR}" 2>/dev/null; then
  echo "pr worker lock exists (${LOCK_DIR}); another worker is running."
  exit 0
fi

cleanup() {
  rmdir "${WORKTREE_LOCK_DIR}" >/dev/null 2>&1 || true
  rmdir "${LOCK_DIR}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

log() {
  echo "[pr_worker] $*"
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

ensure_main_ready() {
  if [[ -n "$(git status --porcelain)" ]]; then
    log "worktree is dirty; skip loop until clean."
    return 1
  fi
  if [[ "$(git branch --show-current)" != "${BASE_BRANCH}" ]]; then
    git checkout "${BASE_BRANCH}" >/dev/null 2>&1 || return 1
  fi
  git pull --ff-only origin "${BASE_BRANCH}" >/dev/null 2>&1 || return 1
  return 0
}

run_codex_for_comment() {
  local pr_number="$1"
  local comment_id="$2"
  local comment_url="$3"
  local comment_body="$4"
  local head_branch="$5"
  local mode="$6"

  local tmp_dir prompt_file response_file log_file
  tmp_dir="$(mktemp -d)"
  prompt_file="${tmp_dir}/prompt.txt"
  response_file="${tmp_dir}/response.txt"
  log_file="${LOG_DIR}/pr-${pr_number}-comment-${comment_id}-$(date +%Y%m%d%H%M%S).log"

  if [[ "${mode}" == "apply" && "${head_branch}" == "${BASE_BRANCH}" ]]; then
    gh pr comment "${pr_number}" --repo "${REPO}" --body "Safety stop: this PR head branch is \`${BASE_BRANCH}\`. Auto-commit is disabled to avoid direct commits to base branch."
    return 0
  fi

  cat >"${prompt_file}" <<EOF
You are working in repository ${REPO}.
A new PR comment requires action.

PR number: #${pr_number}
Head branch: ${head_branch}
Comment URL: ${comment_url}
Comment body:
${comment_body}

Mode: ${mode}

Instructions:
- If mode is "reply-only", do not modify files; provide a concise reviewer response.
- If mode is "apply", implement the requested change if needed, run relevant checks, and summarize what changed.
- If mode is "apply", do not checkout or switch to any branch other than ${head_branch}.
- Never modify any file or directory outside this repository.
- Only modify files under: game/, mobile/, web/, docs/
- Additionally, only these root files may be modified: go.mod, go.sum, main.go, README.md, .gitignore
- Do not modify any other paths/files.
- Be extremely careful with shell commands executed on this PC.
- Never run destructive commands (e.g. rm -rf, git reset --hard, git clean -fd, force-push, or anything that can delete/overwrite data).
- Keep the response concise and specific to the comment.
EOF

  if [[ "${mode}" == "apply" ]]; then
    git fetch origin "${head_branch}"
    if git show-ref --verify --quiet "refs/heads/${head_branch}"; then
      git checkout "${head_branch}"
    else
      git checkout -b "${head_branch}" "origin/${head_branch}"
    fi
    git pull --ff-only origin "${head_branch}"
  else
    git checkout "${BASE_BRANCH}" >/dev/null 2>&1 || true
  fi

  set +e
  codex exec --full-auto -C "$(pwd)" -o "${response_file}" - <"${prompt_file}" 2>&1 \
    | tee "${log_file}" \
    | while IFS= read -r line; do log "pr #${pr_number} comment #${comment_id} | ${line}"; done
  codex_status=${PIPESTATUS[0]}
  set -e
  if [[ "${codex_status}" -ne 0 ]]; then
    gh pr comment "${pr_number}" --repo "${REPO}" --body "Codex failed while processing [this comment](${comment_url}). See worker logs."
    git checkout "${BASE_BRANCH}" >/dev/null 2>&1 || true
    git pull --ff-only origin "${BASE_BRANCH}" >/dev/null 2>&1 || true
    rm -rf "${tmp_dir}"
    return 1
  fi

  local commit_note=""
  if [[ "${mode}" == "apply" ]] && [[ -n "$(git status --porcelain)" ]]; then
    # Guard: ensure commits always go to the PR head branch.
    if [[ "$(git branch --show-current)" != "${head_branch}" ]]; then
      if git show-ref --verify --quiet "refs/heads/${head_branch}"; then
        git checkout "${head_branch}"
      else
        git checkout -b "${head_branch}" "origin/${head_branch}" 2>/dev/null || git checkout -B "${head_branch}"
      fi
    fi
    if [[ "$(git branch --show-current)" == "${BASE_BRANCH}" ]]; then
      gh pr comment "${pr_number}" --repo "${REPO}" --body "Safety stop: detected pending changes on \`${BASE_BRANCH}\`. No commit was created. Please clean up local branch state, then retry."
      git checkout "${BASE_BRANCH}" >/dev/null 2>&1 || true
      return 0
    fi
    git add -A
    git commit -m "fix: address PR #${pr_number} comment #${comment_id}"
    git push origin "${head_branch}"
    commit_note="$(git rev-parse --short HEAD)"
  fi

  local response_text
  response_text="$(cat "${response_file}")"
  if [[ -n "${commit_note}" ]]; then
    response_text="${response_text}"$'\n\n'"Updated branch \`${head_branch}\` at commit \`${commit_note}\`."
  fi
  gh pr comment "${pr_number}" --repo "${REPO}" --body "${response_text}"

  git checkout "${BASE_BRANCH}" >/dev/null 2>&1 || true
  git pull --ff-only origin "${BASE_BRANCH}" >/dev/null 2>&1 || true
  rm -rf "${tmp_dir}"
}

log "starting for ${REPO} (prefix=${TRIGGER_PREFIX}, interval=${POLL_INTERVAL}s)"

while true; do
  handled_any=false
  acquire_worktree_lock

  if ! ensure_main_ready; then
    release_worktree_lock
    sleep "${POLL_INTERVAL}"
    continue
  fi

  while IFS= read -r pr_number; do
    [[ -z "${pr_number}" ]] && continue
    head_branch="$(gh pr view "${pr_number}" --repo "${REPO}" --json headRefName --jq .headRefName)"

    while IFS= read -r row; do
      [[ -z "${row}" ]] && continue
      json="$(printf '%s' "${row}" | decode_base64)"
      comment_id="$(printf '%s' "${json}" | jq -r '.id')"
      [[ -f "${PROCESSED_DIR}/${comment_id}.done" ]] && continue

      comment_url="$(printf '%s' "${json}" | jq -r '.html_url')"
      comment_body="$(printf '%s' "${json}" | jq -r '.body')"
      first_line="$(printf '%s\n' "${comment_body}" | head -n 1)"

      mode="apply"
      if [[ "${first_line}" == "${TRIGGER_PREFIX} reply"* ]]; then
        mode="reply-only"
      fi

      log "processing PR #${pr_number} comment #${comment_id} (${mode})"
      handled_any=true
      if run_codex_for_comment "${pr_number}" "${comment_id}" "${comment_url}" "${comment_body}" "${head_branch}" "${mode}"; then
        touch "${PROCESSED_DIR}/${comment_id}.done"
      else
        log "failed PR #${pr_number} comment #${comment_id}"
      fi
    done < <(
      {
        gh api "repos/${REPO}/issues/${pr_number}/comments?per_page=100"
        gh api "repos/${REPO}/pulls/${pr_number}/comments?per_page=100"
      } | jq -cr --arg pfx "${TRIGGER_PREFIX}" '
        .[]?
        | select(.user.login != "github-actions[bot]")
        | select((.body // "") | startswith($pfx))
        | {id, html_url, body}
        | @base64
      '
    )
  done < <(gh pr list --repo "${REPO}" --state open --limit 100 --json number --jq '.[].number')
  if [[ "${handled_any}" != "true" ]]; then
    log "poll: no new @codex PR comments"
  fi
  release_worktree_lock

  sleep "${POLL_INTERVAL}"
done
