## Misskey Note/Drive Delete Tool

Misskeyのノートとドライブファイルを一括削除するCLIツールです。

> **警告**: 本ツールはローカル限定投稿での利用を推奨します。連合を含む投稿では他サーバーへ削除リクエストが飛びます。
> 削除間隔を過剰に短くしないでください（デフォルト 30秒 を推奨）。

## セットアップ

1. `.env` を作成（`.env.sample` を参照）
2. `go build` でビルド
3. `./misskeyNotedel [flags]` で実行

## CLI フラグ

| フラグ | 説明 | デフォルト |
|---|---|---|
| `--token` | Misskey APIトークン | (必須) |
| `--host` | Misskeyホスト名 | (必須) |
| **ノート削除** |||
| `--note-older-than` | 指定期間より古いノートのみ削除 (`7d` `12h` `30m` 等) | 無効 |
| `--keep-with-reactions` | リアクション付きノートを保持 (`true/false`) | `false` |
| `--keep-with-renotes` | リノート済みノートを保持 (`true/false`) | `false` |
| `--keep-condition-mode` | 保持判定モード: `or` または `and` | `or` |
| **ドライブ削除** |||
| `--drive-older-than` | 指定期間より古いファイルのみ削除 | 無効 |
| `--drive-mode` | `none`(削除なし) / `all`(全削除) / `unused`(未使用のみ) | `none` |
| `--skip-notes` | ノート削除をスキップしドライブのみ処理 (`true/false`) | `false` |
| **セーフガード** |||
| `--dry-run` | 削除対象の表示のみ（削除しない） | `false` |
| `--yes` | 確認プロンプトをスキップ | `false` |
| `--max-delete` | N件削除で自動停止（`0`=無制限） | `0` |
| `--force` | 既存ロックファイルを無視 | `false` |
| `--delete-interval` | 削除間隔（秒、最小値5） | `30` |
| **ログ** |||
| `--verbose` / `-v` | スキップ理由も表示 | `false` |
| `--quiet` / `-q` | エラーのみ表示 | `false` |

上記フラグはすべて同名の環境変数（例: `KEEP_WITH_REACTIONS=true`）でも設定可能です。
詳細は `.env.sample` を参照してください。

## 使用例

```bash
# 30分以上前のノートを全削除（リアクション付きは残す）
./misskeyNotedel --note-older-than=30m --keep-with-reactions=true --yes

# ドライランモードで対象確認
./misskeyNotedel --note-older-than=7d --drive-mode=all --dry-run

# ドライブのみ未使用ファイルを削除（ノートはそのまま）
./misskeyNotedel --skip-notes=true --drive-mode=unused --yes
```

## 保護対象

- **プロフィール画像（アイコン・バナー）**: ドライブ削除時も自動的に除外されます
- `KEEP_WITH_REACTIONS` / `KEEP_WITH_RENOTES` でリアクション・リノート付きノートを保持可能

## 権限

必要最小限のAPI権限を付与してください：
- ノート削除のみ → 「プロフィール関連」「ノート投稿操作」
- ドライブ削除を含む → 上記に加え「ドライブ閲覧」「ドライブ投稿操作」
