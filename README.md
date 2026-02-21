# mobile-game-template

`flib` のシーンベース構成で、スマホ向け縦持ち 16:9 (720x1280) の最小テンプレートです。

## Included

- 画像ロード: `game/assets/images`
- フォントロード: `game/assets/fonts`
- `text/v2` の `GoTextFace` で `Hello codex` 表示
- 画面下部テキストに `hello codex!!!` 表示
- SE/BGMロード: `game/assets/se`, `game/assets/bgm`
- JSONロード: `game/assets/data`

## Run

```bash
go run .
```

## Build all

```bash
go build ./...
```

## Bootstrap After Creating From Template

Run once after creating a new repository from this template:

```bash
./scripts/bootstrap_template.sh --owner <github-owner> --repo <new-repo>
```

This updates module/import paths, visible app/repo name strings, and launchd plist naming.

## WASM (GitHub Pages)

```bash
make wasm
make serve
```

- `make wasm` builds `docs/game.wasm` and copies `web/index.html`, `web/wasm_exec.js`.
- `make serve` starts local preview at `http://localhost:8080`.

## GitHub Actions

- `.github/workflows/pages.yml`:
  - push to `main` -> build WASM -> deploy GitHub Pages
- `.github/workflows/pr-preview.yml`:
  - PR open/update -> deploy Pages preview -> comment preview URL to PR

## New Repo Checklist

After creating a new repo from this template, run the following:

1. Run bootstrap:
```bash
./scripts/bootstrap_template.sh --owner <github-owner> --repo <new-repo>
```
2. Validate and commit:
```bash
go test ./...
make wasm
git add -A
git commit -m "Bootstrap from template"
```
3. GitHub `Settings > Pages`:
   - set `Build and deployment` to `GitHub Actions`
4. GitHub `Settings > Actions > General`:
   - set `Workflow permissions` to `Read and write`
5. If using PR preview deploys:
   - check `Settings > Environments > github-pages` branch restriction rules
6. (Recommended) Branch protection for `main`:
   - require PR before merge
   - require required status checks
7. On worker machine:
   - `gh auth login`
   - `codex login`
   - start workers with `./scripts/start_workers.sh`

## Local Codex Worker (Issue -> PR)

This template uses a local always-on machine (e.g. home Mac mini) to process issues with Codex CLI.
GitHub Actions is not used for Codex execution.

### Prerequisites

```bash
gh auth login
codex login
```

### Run once for a single issue

```bash
./scripts/run_issue.sh 123
```

### Run as polling worker

```bash
LABEL=autocodex POLL_INTERVAL=60 ./scripts/issue_worker.sh
```

- Only open issues with label `autocodex` are picked up.
- Worker creates `codex/issue-<number>` branch, commits, pushes, and opens a PR.
- Worker state/logs are saved under `.codex-worker/`.
- Before scanning each next issue, worker guarantees checkout/update of `main` (or waits if worktree is dirty).
- If clarification is needed before implementation, worker posts a question on the issue and waits.
- After you reply on the issue, worker resumes and continues implementation.

### PR comment worker (optional)

```bash
POLL_INTERVAL=60 TRIGGER_PREFIX=@codex ./scripts/pr_worker.sh
```

- Watches open PR comments and review comments.
- Reacts only to comments starting with `@codex`.
- `@codex reply ...` -> reply only, no code changes.
- `@codex ...` -> may edit PR branch, commit, push, and comment back.
- `issue_worker` posts an initial Codex response to each created PR.
- If initial response contains `QUESTION_FOR_REVIEWER: ...`, answer in PR with `@codex ...` and `pr_worker` continues from there.

### Start both workers together

```bash
./scripts/start_workers.sh
```

- Starts both `issue_worker` and `pr_worker` in one command.
- Stop both workers with `Ctrl+C`.

### macOS launchd (optional)

1. Copy `scripts/launchd/com.ichibankunio.mobile-game-template.issue-worker.plist`
   to `~/Library/LaunchAgents/`
2. Replace `__REPO_PATH__` with your local repository path
3. Load service:

```bash
launchctl unload ~/Library/LaunchAgents/com.ichibankunio.mobile-game-template.issue-worker.plist 2>/dev/null || true
launchctl load ~/Library/LaunchAgents/com.ichibankunio.mobile-game-template.issue-worker.plist
```

## Mobile binding

```bash
ebitenmobile bind -v -target ios -o ./ios/Mobile.xcframework ./mobile
# or
# ebitenmobile bind -v -target android -javapkg com.example.mobilegametemplate -o ./android/mobile-game-template.aar ./mobile
```
