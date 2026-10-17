# リリース手順

GitHub Release の作成は、タグの push をきっかけに GitHub Actions (`.github/workflows/release.yml`) が GoReleaser (`.goreleaser.yaml`) で行う。

## 手順

1. main が最新で、CI が通っていることを確認する
2. 依存を更新した場合は `go run ./scripts/notices` で THIRD_PARTY_NOTICES を更新しておく (CI でも確認される)。go-graphviz を更新した場合は、内包する Graphviz などのバージョンを確認して `third_party/graphviz-wasm/` も更新する
3. バージョンのタグを付けて push する

   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

4. Actions の Release ワークフローが完了したら、GitHub Release の内容 (変更履歴、アーカイブ) を確認する

## Homebrew / Scoop

リリース時に GoReleaser が [onozaty/homebrew-tap](https://github.com/onozaty/homebrew-tap) の Cask (`Casks/dbfolio.rb`) と [onozaty/scoop-bucket](https://github.com/onozaty/scoop-bucket) のマニフェスト (`dbfolio.json`) を更新する。書き込みには、それぞれのリポジトリに push できるトークンを dbfolio のリポジトリの Secrets に登録しておく。

- `HOMEBREW_TAP_TOKEN`: onozaty/homebrew-tap に push できるトークン
- `SCOOP_BUCKET_TOKEN`: onozaty/scoop-bucket に push できるトークン

プレリリースやスナップショットでは更新しない (`skip_upload: auto`)。

## デモサイト

GitHub Release を公開すると、`.github/workflows/pages.yml` が `demo/schema.json` からドキュメントを生成して GitHub Pages (https://onozaty.github.io/dbfolio/) に公開する。手動でも実行できる (Actions の Demo site ワークフロー)。サンプルの再生成方法は [demo/README.md](../demo/README.md) を参照。

## 配布物

- 対象: linux / darwin / windows の amd64 / arm64
- アーカイブ: tar.gz (Windows は zip)。バイナリと LICENSE、README.md、THIRD_PARTY_NOTICES を含む
- checksums.txt

変更履歴は GitHub の自動生成を使い、`docs:` / `test:` / `chore:` / `ci:` のコミットとマージコミットは除く。

## ローカルでの確認

```bash
goreleaser check
goreleaser release --snapshot --clean   # dist/ に出力 (git 管理外)
```
