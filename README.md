# envtop

Kubernetes 上のアプリケーションが参照している環境変数を一覧表示する TUI ツール。

ConfigMap / Secret / SealedSecret を横断して、Deployment や StatefulSet の環境変数を確認できます。

## Features

- **2行レイアウトの TUI**: 上段に Namespace / Apps、下段に Environment Variables を表示
- **ConfigMap/Secret/SealedSecret 横断表示**: env / envFrom を解決して一覧表示
- **インクリメンタル検索**: `/` キーで Namespace / Apps / Env をリアルタイム絞り込み
- **セキュアな Secret 表示**: デフォルトではハッシュ値のみ表示、確認プロンプト後に Reveal
- **Seal 機能**: kubeseal と連携して Secret 値を暗号化
- **Namespace 間 Diff**: 同一アプリの環境変数を namespace 間で比較
- **クリップボード対応**: Reveal / Seal 結果を `c` キーでコピー

## Installation

### From source

```bash
go install github.com/ginbear/k8s-envtop@latest
```

### Build locally

```bash
git clone https://github.com/ginbear/k8s-envtop.git
cd k8s-envtop
go build -o envtop .
```

## Usage

```bash
envtop
```

kubeconfig (`~/.kube/config` または `KUBECONFIG` 環境変数) を使用して、現在のコンテキストに接続します。

## Key Bindings

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | ペイン切替 |
| `↑` / `k` | 上に移動 |
| `↓` / `j` | 下に移動 |
| `←` / `h` | 左ペインへ |
| `→` / `l` | 右ペインへ |
| `Enter` | 選択確定（次のペインへ移動） |
| `/` | インクリメンタル検索 |
| `r` | Secret を Reveal（確認後表示） |
| `s` | Seal（kubeseal で暗号化） |
| `d` | Diff モード（namespace 間比較） |
| `c` | クリップボードにコピー（Reveal/Seal 結果画面） |
| `Esc` | 戻る / キャンセル |
| `q` | 終了 |

## Display Format

### Environment Variables

| Column | Description |
|--------|-------------|
| NAME | 環境変数名 |
| SOURCE | 参照元（`cm/name` or `sec/name`） |
| KIND | ConfigMap / Secret / SealedSecret |
| VALUE | 値（Secret はハッシュ表示） |

### Secret Values

Secret / SealedSecret の値はデフォルトで以下の形式で表示されます：

```
HASH: ab12cd34  len=32 sealed
```

- `HASH`: SHA256 ハッシュの先頭 8 文字
- `len`: 値の長さ
- `sealed`: SealedSecret 由来の場合に表示

## Reveal Feature

`r` キーで Secret の値を表示できます。

1. 表示形式を選択（Base64 / Plain Text）
2. 確認プロンプトで "OK" と入力
3. 値が 30 秒間表示される
4. `c` キーでクリップボードにコピー可能

**Note**: `ENVTOP_DISABLE_REVEAL=1` を設定すると Reveal 機能を無効化できます。

## Seal Feature

`s` キーで kubeseal を使って Secret 値を暗号化できます。

1. Secret 名を入力（Secret/SealedSecret 選択時は自動入力）
2. 暗号化したい平文を入力
3. Enter で実行（実行コマンドがプレビュー表示されます）
4. 暗号化された値が表示される
5. `c` キーでクリップボードにコピー

暗号化された値は SealedSecret の YAML に貼り付けて使用できます。

**Note**: kubeseal コマンドがインストールされている必要があります。

## Diff Mode

`d` キーで namespace 間の環境変数を比較できます。

| Status | Description |
|--------|-------------|
| SAME | 値が一致 |
| VALUE_DIFF | 値が異なる |
| ONLY_IN_A | 比較元のみに存在 |
| ONLY_IN_B | 比較先のみに存在 |

Secret の比較はハッシュ値で行われるため、中身を見ずに差分を確認できます。

## Requirements

- Go 1.21+
- Kubernetes cluster with read access
- kubeconfig configured
- kubeseal (Seal 機能を使用する場合)

### Clipboard Support

クリップボード機能は以下のコマンドを使用します：

| OS | Command |
|----|---------|
| macOS | `pbcopy` |
| Linux | `xclip` or `xsel` |
| Windows | `clip` |

### Required RBAC Permissions

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: envtop-reader
rules:
- apiGroups: [""]
  resources: ["namespaces", "configmaps", "secrets"]
  verbs: ["get", "list"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list"]
- apiGroups: ["bitnami.com"]
  resources: ["sealedsecrets"]
  verbs: ["get", "list"]
```

## License

MIT

## Acknowledgements

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes client
