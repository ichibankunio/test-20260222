#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./scripts/bootstrap_template.sh --owner <github-owner> --repo <new-repo>

Example:
  ./scripts/bootstrap_template.sh --owner myname --repo my-new-game
EOF
}

OWNER=""
REPO=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --owner)
      OWNER="${2:-}"
      shift 2
      ;;
    --repo)
      REPO="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown arg: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ -z "${OWNER}" || -z "${REPO}" ]]; then
  usage
  exit 1
fi

if ! command -v perl >/dev/null 2>&1; then
  echo "perl is required." >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}"

OLD_OWNER="ichibankunio"
OLD_REPO="mobile-game-template"
OLD_MODULE="github.com/${OLD_OWNER}/${OLD_REPO}"
NEW_MODULE="github.com/${OWNER}/${REPO}"

replace_literal() {
  local old="$1"
  local new="$2"
  shift 2
  OLD_LITERAL="${old}" NEW_LITERAL="${new}" perl -i -pe '
    BEGIN {
      $old = $ENV{"OLD_LITERAL"};
      $new = $ENV{"NEW_LITERAL"};
    }
    s/\Q$old\E/$new/g;
  ' "$@"
}

echo "bootstrapping template..."
echo "  old module: ${OLD_MODULE}"
echo "  new module: ${NEW_MODULE}"

FILES=(
  "go.mod"
  "main.go"
  "cmd/game/main.go"
  "mobile/mobile.go"
  "README.md"
  "game/assets/data/sample.json"
  "game/scene_main.go"
)

replace_literal "${OLD_MODULE}" "${NEW_MODULE}" "${FILES[@]}"
replace_literal "${OLD_REPO}" "${REPO}" "${FILES[@]}"

# Rename launchd template file and update label/path references.
OLD_PLIST="scripts/launchd/com.${OLD_OWNER}.${OLD_REPO}.issue-worker.plist"
NEW_PLIST="scripts/launchd/com.${OWNER}.${REPO}.issue-worker.plist"
if [[ -f "${OLD_PLIST}" ]]; then
  mv "${OLD_PLIST}" "${NEW_PLIST}"
fi
if [[ -f "${NEW_PLIST}" ]]; then
  replace_literal "com.${OLD_OWNER}.${OLD_REPO}.issue-worker" "com.${OWNER}.${REPO}.issue-worker" "${NEW_PLIST}"
fi
replace_literal "com.${OLD_OWNER}.${OLD_REPO}.issue-worker.plist" "com.${OWNER}.${REPO}.issue-worker.plist" "README.md"

echo "done."
echo "next:"
echo "  1) review changes: git diff"
echo "  2) run checks: go test ./... && make wasm"
echo "  3) commit bootstrap updates"
