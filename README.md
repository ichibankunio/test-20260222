# test-20260222

`flib` のシーンベース構成で、スマホ向け縦持ち 9:16 (144x256) の最小テンプレートです。

## Included

- 画像ロード: `game/assets/images`
- フォントロード: `game/assets/fonts`
- `text/v2` の `GoTextFace` で `Hello codex` 表示
- 画面下部テキストに `hello codex!!!` 表示
- SE/BGMロード: `game/assets/se`, `game/assets/bgm`
- JSONロード: `game/assets/data`
- ゲーム画面の配色はゲームボーイ風8bitパレット
- 敵弾描画は `kage` シェーダー + 三角形バッチで実行（高密度時の負荷を軽減）
- 被弾時とコイン取得時に `kage` シェーダーのパーティクルエフェクトを再生
- コイン取得で飛散したピクセルが背景に蓄積し、`game/assets/images/bishonen.png` の美男子画像が徐々に完成するプロトタイプを追加

## Run

```bash
go run .
```

- 操作:
  - 自機（丸）はタップ/ドラッグ（スワイプ）またはマウスドラッグで移動
  - 敵弾（丸）に当たるとゲームオーバー
  - 当たり判定は自機と敵弾の丸同士で判定
  - コイン役の丸を集めるとゲージが増加
  - コイン取得で `ART%` が上昇（背景のポートレート構築率）
  - ゲージ満タンでステージクリアして次ステージへ進行
  - ステージは 1〜10 の固定パターン（数列・対称・回転ベース）で循環
  - 弾幕パターン実装は `game/danmaku_stage*.go` の 1 ステージ 1 ファイル構成
  - 後半ステージには軽い追尾のホーミング弾が出現
  - ゲームオーバー中にタップ / クリック / `SPACE` / `↑` / `W`: 現在ステージをリトライ
  - `ESC`: 終了

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

## Gameplay Spec Note

- 目標体験:
  - 弾幕を避けてコイン回収
  - 回収したコインがピクセル飛散
  - 飛散ピクセルが背景に蓄積し、美男子ポートレートを構築
- 今回のプロトタイプ範囲:
  - `coin pickup -> 飛散ピクセル` の追加
  - 飛散ピクセルのターゲット座標への吸着・定着
  - 背景ポートレート段階表示
  - `ART xx%` 進行表示
  - ステージリトライ時もアート進行は保持
- 今後の主要タスク:
  - ドットアート素材運用（表情差分含む）
  - コイン色と蓄積色の連動ルール
  - パーツ解放やボーナス等の進行ゲーム化
  - 弾幕視認性を崩さない背景表示チューニング
  - アート進行のセーブ/ロード
  - 完成時演出や共有導線の拡張

## GitHub Actions

- `.github/workflows/pages.yml`:
  - push to `main` -> build WASM -> deploy GitHub Pages
- `.github/workflows/pr-preview.yml`:
  - PR open/update -> deploy Pages preview -> comment preview URL to PR
- `.github/workflows/cleanup-merged-branch.yml`:
  - merged PR (head branch starts with `codex/`) -> auto delete branch

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
- `issue_worker` and `pr_worker` share a worktree lock, so only one worker performs git mutations at a time.
- Worker console logs are prefixed with `[issue_worker]` or `[pr_worker]`.
- Workers post an immediate "started" comment on Issue/PR when Codex begins handling a request.

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

1. Copy `scripts/launchd/com.ichibankunio.test-20260222.issue-worker.plist`
   to `~/Library/LaunchAgents/`
2. Replace `__REPO_PATH__` with your local repository path
3. Load service:

```bash
launchctl unload ~/Library/LaunchAgents/com.ichibankunio.test-20260222.issue-worker.plist 2>/dev/null || true
launchctl load ~/Library/LaunchAgents/com.ichibankunio.test-20260222.issue-worker.plist
```

## Mobile binding

```bash
ebitenmobile bind -v -target ios -o ./ios/Mobile.xcframework ./mobile
# or
# ebitenmobile bind -v -target android -javapkg com.example.mobilegametemplate -o ./android/test-20260222.aar ./mobile
```
