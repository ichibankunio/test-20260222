#!/usr/bin/env bash
set -euo pipefail

ISSUE_NUMBER="${1:-}"
if [[ -z "${ISSUE_NUMBER}" ]]; then
  echo "usage: $0 <issue-number>" >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "gh command is required." >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq command is required." >&2
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "curl command is required." >&2
  exit 1
fi
if ! command -v codex >/dev/null 2>&1; then
  echo "codex command is required." >&2
  exit 1
fi
if ! command -v git >/dev/null 2>&1; then
  echo "git command is required." >&2
  exit 1
fi

REPO="${REPO:-$(gh repo view --json nameWithOwner --jq .nameWithOwner)}"
BASE_BRANCH="${BASE_BRANCH:-main}"
BRANCH="codex/issue-${ISSUE_NUMBER}"
TMP_DIR="$(mktemp -d)"
PROMPT_FILE="${TMP_DIR}/prompt.txt"
RESPONSE_FILE="${TMP_DIR}/response.txt"
PLAN_PROMPT_FILE="${TMP_DIR}/plan_prompt.txt"
PLAN_RESPONSE_FILE="${TMP_DIR}/plan_response.txt"
IMAGE_DIR="${TMP_DIR}/images"
STATE_DIR="${STATE_DIR:-$(pwd)/.codex-worker}"
QUESTION_STATE_DIR="${STATE_DIR}/issue-questions"
QUESTION_STATE_FILE="${QUESTION_STATE_DIR}/${ISSUE_NUMBER}.json"
CODEX_IMAGE_ARGS=()

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

if [[ -n "$(git status --porcelain)" ]]; then
  echo "worktree is dirty. commit/stash changes before running." >&2
  exit 1
fi

if gh pr list --repo "${REPO}" --state open --head "${BRANCH}" --json number --jq 'length > 0' | grep -q true; then
  echo "open PR already exists for ${BRANCH}; skipping."
  exit 0
fi

mkdir -p "${QUESTION_STATE_DIR}"
mkdir -p "${IMAGE_DIR}"

RAW_TITLE="$(gh issue view "${ISSUE_NUMBER}" --repo "${REPO}" --json title --jq .title)"
RAW_BODY="$(gh issue view "${ISSUE_NUMBER}" --repo "${REPO}" --json body --jq .body)"
ISSUE_URL="$(gh issue view "${ISSUE_NUMBER}" --repo "${REPO}" --json url --jq .url)"
PR_TITLE="${RAW_TITLE}"
ANSWER_CONTEXT=""

collect_issue_images() {
  local url idx ext out
  idx=0
  while IFS= read -r url; do
    [[ -z "${url}" ]] && continue
    case "${url}" in
      *.png|*.jpg|*.jpeg|*.gif|*.webp|*.bmp|*.svg|*user-images.githubusercontent.com/*|*github.com/user-attachments/*)
        ;;
      *)
        continue
        ;;
    esac

    idx=$((idx + 1))
    ext="$(printf '%s' "${url}" | sed -E 's/.*(\.[A-Za-z0-9]{2,5})(\?.*)?$/\1/')"
    if [[ ! "${ext}" =~ ^\.[A-Za-z0-9]{2,5}$ ]]; then
      ext=".img"
    fi
    out="${IMAGE_DIR}/issue-${ISSUE_NUMBER}-${idx}${ext}"
    if curl -fsSL "${url}" -o "${out}"; then
      CODEX_IMAGE_ARGS+=("-i" "${out}")
    fi
  done < <(
    printf '%s\n' "${RAW_BODY}" \
      | perl -nE '
          while(/\!\[[^\]]*\]\((https?:\/\/[^)\s]+)\)/g){say $1}
          while(/<(https?:\/\/[^>]+)>/g){say $1}
          while(/\b(https?:\/\/\S+)/g){say $1}
        ' \
      | sed -E 's/[)>.,]+$//' \
      | awk '!seen[$0]++'
  )
}

run_codex_exec() {
  local output_file="$1"
  local prompt_file="$2"
  codex exec --full-auto -C "$(pwd)" -o "${output_file}" "${CODEX_IMAGE_ARGS[@]}" - < "${prompt_file}"
}

collect_issue_images

if [[ -f "${QUESTION_STATE_FILE}" ]]; then
  ASKED_AT="$(jq -r '.asked_at' "${QUESTION_STATE_FILE}")"
  ASKED_COMMENT_ID="$(jq -r '.asked_comment_id // empty' "${QUESTION_STATE_FILE}")"
  ANSWER_JSON="$(
    gh api "repos/${REPO}/issues/${ISSUE_NUMBER}/comments?per_page=100" \
      | jq -cr --arg asked_at "${ASKED_AT}" --arg asked_comment_id "${ASKED_COMMENT_ID}" '
          map(
            select(.created_at > $asked_at)
            | select((.id|tostring) != $asked_comment_id)
            | select((.body // "") | test("^\\s*@codex\\b"; "i"))
          )
          | last // empty
        '
  )"
  if [[ -z "${ANSWER_JSON}" ]]; then
    echo "waiting for reviewer response on issue #${ISSUE_NUMBER}"
    exit 20
  fi
  ANSWER_CONTEXT="$(printf '%s' "${ANSWER_JSON}" | jq -r '.body')"
  rm -f "${QUESTION_STATE_FILE}"
fi

cat > "${PLAN_PROMPT_FILE}" <<EOF
You are preparing implementation for GitHub Issue #${ISSUE_NUMBER} in ${REPO}.

Issue title:
${RAW_TITLE}

Issue body:
${RAW_BODY}

Latest reviewer answer on this issue (if any):
${ANSWER_CONTEXT}

If the issue is clear enough, output exactly:
READY_TO_IMPLEMENT

If you need clarification before implementation, output exactly:
QUESTION_FOR_ISSUE: <question>

Do not output anything else.
EOF

run_codex_exec "${PLAN_RESPONSE_FILE}" "${PLAN_PROMPT_FILE}"

QUESTION_TEXT="$(sed -n 's/^QUESTION_FOR_ISSUE:[[:space:]]*//p' "${PLAN_RESPONSE_FILE}" | head -n 1)"
if [[ -n "${QUESTION_TEXT}" ]]; then
  QUESTION_COMMENT_BODY="$(cat <<EOF
Codex question before implementation:

${QUESTION_TEXT}

Please reply on this issue. Worker will resume after your answer.
EOF
)"
  QUESTION_COMMENT_JSON="$(
    jq -n --arg body "${QUESTION_COMMENT_BODY}" '{body: $body}' \
      | gh api "repos/${REPO}/issues/${ISSUE_NUMBER}/comments" --input -
  )"
  printf '%s' "${QUESTION_COMMENT_JSON}" | jq '{asked_at: .created_at, asked_comment_id: .id, question: .body}' > "${QUESTION_STATE_FILE}"
  echo "asked clarification question for issue #${ISSUE_NUMBER}; waiting for response"
  exit 20
fi

git fetch origin "${BASE_BRANCH}"
git checkout "${BASE_BRANCH}"
git pull --ff-only origin "${BASE_BRANCH}"
git checkout -B "${BRANCH}"

cat > "${PROMPT_FILE}" <<EOF
You are working in repository ${REPO}.
Implement GitHub Issue #${ISSUE_NUMBER}.

Issue title:
${RAW_TITLE}

Issue body:
${RAW_BODY}

Latest reviewer answer on issue (if any):
${ANSWER_CONTEXT}

Requirements:
- Implement only what is necessary to satisfy this issue.
- Do not commit or push directly.
- Do not checkout or switch to another branch. Stay on ${BRANCH}.
- Never modify any file or directory outside this repository.
- Only modify files under: game/, mobile/, web/, docs/
- Additionally, only these root files may be modified: go.mod, go.sum, main.go, README.md, .gitignore
- Do not modify any other paths/files.
- Be extremely careful with shell commands executed on this PC.
- Never run destructive commands (e.g. rm -rf, git reset --hard, git clean -fd, force-push, or anything that can delete/overwrite data).
- Run relevant checks (at minimum: go test ./... and make wasm if applicable).
- Keep changes small and reviewable.
- Update docs if behavior or usage changed.
- In your final response, include a concise implementation summary for the PR comment.
- If you need reviewer input before further changes, add a line:
  QUESTION_FOR_REVIEWER: <your question>
EOF

run_codex_exec "${RESPONSE_FILE}" "${PROMPT_FILE}"

# Guard: ensure commit always happens on the issue branch.
if [[ "$(git branch --show-current)" != "${BRANCH}" ]]; then
  if git show-ref --verify --quiet "refs/heads/${BRANCH}"; then
    git checkout "${BRANCH}"
  else
    git checkout -b "${BRANCH}" "origin/${BRANCH}" 2>/dev/null || git checkout -B "${BRANCH}"
  fi
fi

if [[ -z "$(git status --porcelain)" ]]; then
  gh issue comment "${ISSUE_NUMBER}" --repo "${REPO}" --body "Codex ran for this issue but produced no file changes."
  git checkout "${BASE_BRANCH}"
  git branch -D "${BRANCH}"
  exit 0
fi

git add -A
git commit -m "feat: resolve issue #${ISSUE_NUMBER}"
git push -u origin "${BRANCH}"

PR_BODY="$(cat <<EOF
Automated implementation for #${ISSUE_NUMBER}.

Source issue:
- ${ISSUE_URL}

Closes #${ISSUE_NUMBER}
EOF
)"

PR_URL="$(gh pr create \
  --repo "${REPO}" \
  --base "${BASE_BRANCH}" \
  --head "${BRANCH}" \
  --title "${PR_TITLE}" \
  --body "${PR_BODY}")"

PR_NUMBER="$(printf '%s\n' "${PR_URL}" | sed -E 's#.*/pull/([0-9]+).*#\1#')"
RESPONSE_TEXT="$(cat "${RESPONSE_FILE}")"
if [[ -n "${RESPONSE_TEXT}" ]]; then
  gh pr comment "${PR_NUMBER}" --repo "${REPO}" --body $'Codex initial response:\n\n'"${RESPONSE_TEXT}"
fi

QUESTION_TEXT="$(sed -n 's/^QUESTION_FOR_REVIEWER:[[:space:]]*//p' "${RESPONSE_FILE}" | head -n 1)"
if [[ -n "${QUESTION_TEXT}" ]]; then
  gh pr comment "${PR_NUMBER}" --repo "${REPO}" --body $'Codex question for reviewer:\n\n'"${QUESTION_TEXT}"$'\n\nPlease answer in this PR with `@codex ...`.'
fi

gh issue comment "${ISSUE_NUMBER}" --repo "${REPO}" --body "Codex started and opened a PR: ${PR_URL}"
echo "created PR: ${PR_URL}"

# Return to base branch so the worker is ready for the next issue.
git checkout "${BASE_BRANCH}"
git pull --ff-only origin "${BASE_BRANCH}"
