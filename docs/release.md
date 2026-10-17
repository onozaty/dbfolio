# リリース手順

GitHub Release の作成は、タグの push をきっかけに GitHub Actions (`.github/workflows/release.yml`) が GoReleaser (`.goreleaser.yaml`) で行う。

## 手順

1. main が最新で、CI が通っていることを確認する
2. 依存を更新した場合は `go run ./scripts/notices` で THIRD_PARTY_NOTICES を更新しておく (CI でも確認される)。go-graphviz を更新した場合は、内包する Graphviz などのバージョンを確認して `third_party/graphviz-wasm/` も更新する
3. バージョンのタグを付けて push する

   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

4. Actions の Release ワークフローが完了したら、GitHub Release の内容 (変更履歴、アーカイブ) を確認する

## 配布物

- 対象: linux / darwin / windows の amd64 / arm64
- アーカイブ: tar.gz (Windows は zip)。バイナリと LICENSE、README.md、THIRD_PARTY_NOTICES を含む
- checksums.txt

変更履歴は GitHub の自動生成を使い、`docs:` / `test:` / `chore:` / `ci:` のコミットは除く。

## ローカルでの確認

```bash
goreleaser check
goreleaser release --snapshot --clean   # dist/ に出力 (git 管理外)
```
