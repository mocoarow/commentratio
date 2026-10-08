# commentratio 設計

## 目的

コードの量に見合わないコメントを検出する Go の Linter。10 行の関数に 10 行の GoDoc、100 行のファイルに 100 行のコメント、といった過剰なコメントを警告する。あわせて、一定以上の長さがある公開 API に GoDoc がない場合も警告する。

既存の Linter（maxcomments、commentlen、commentsize、godoc-lint）は、GoDoc の行数を対象コードの行数との比率で制限できないため自作する。

## チェック対象

| 対象 | 設定キー | 「多すぎ」 | 「GoDoc なし」 |
|---|---|---|---|
| 関数・メソッドの GoDoc | `func-doc` | ✅ | ✅（公開識別子のみ） |
| 型・定数・変数の GoDoc | `decl-doc` | ✅ | ✅（公開識別子のみ） |
| 関数本体内のコメント | `func-body` | ✅ | — |
| ファイル全体のコメント | `file` | ✅ | — |

本体内のコメントとファイル全体に下限を設けないのは、警告を消すための水増しコメントを誘発しないため。

## 行数の数え方

### コード行

対象範囲のうち、空行とコメントだけの行を除いた行数。

| 対象 | 範囲 |
|---|---|
| `func-doc` | `func` キーワードから閉じ `}` まで（シグネチャを含む）。本体のない関数宣言はシグネチャのみ |
| `decl-doc` | GoDoc が `GenDecl` に付いている場合は `GenDecl` 全体、`const ( ... )` の中の個別の spec に付いている場合はその spec のみ |
| `func-body` | 本体の `{` から `}` まで |
| `file` | ファイル全体 |

### コメント行

中身のあるコメントを含む行数。以下は数えない。

- 中身のない `//` 行（段落区切り）、および複数行の `/* */` の中の空行と、装飾用の `*` だけの行
- ディレクティブ: `//go:`、`//nolint`、`//lint:`、`//line ` で始まるコメント
- `file` の計算でのみ: package 句より前にあり GoDoc として付いていないコメント（ライセンスヘッダー）と、パッケージの GoDoc。`doc.go` が常に「多すぎ」になるのを防ぐため

行末コメント（`x := 1 // note`）のある行は、コード行とコメント行の両方に数える。

複数行の raw string は、中の空行も含めてすべてコード行に数える。

### 対象外のファイル

- 生成ファイル（`ast.IsGenerated` が true）はチェックしない
- `_test.go` はチェックする

## 判定

### 多すぎ

```
上限 = min(max-lines, max(free-lines, floor(code_lines × max-ratio)))
comment_lines > 上限 なら警告
```

- `free-lines`: この行数以内なら比率に関係なく常に OK。短い関数でも 1〜数行の GoDoc を書けるようにする
- `max-ratio`: コード行に対するコメント行の比率の上限
- `max-lines`: コードがどれだけ長くても超えてはいけない絶対的な上限

### GoDoc なし

```
code_lines >= require-from かつ 識別子が公開されている かつ GoDoc がない なら警告
```

- `require-from` が 0 のときはチェックしない
- 数えるべき GoDoc の行が 0 行なら「GoDoc なし」とみなす。ディレクティブ（`//nolint` など）だけが付いている場合も GoDoc なし
- 公開の判定は `ast.IsExported`（識別子名）で行う。メソッドはメソッド名で判定する
- `decl-doc` では、`GenDecl` と spec のどちらかに GoDoc があれば「GoDoc あり」とみなす。spec 内に公開識別子が 1 つでもあれば公開とみなす

## 設定

### デフォルト値

| 対象 | free-lines | max-ratio | max-lines | require-from |
|---|---|---|---|---|
| `func-doc` | 3 | 0.2 | 15 | 10 |
| `decl-doc` | 3 | 0.2 | 15 | 10 |
| `func-body` | 2 | 0.2 | 10 | — |
| `file` | 5 | 0.3 | 200 | — |

`file` の比率を緩めにしているのは、ファイル全体の数字に GoDoc も含まれるため。数値は初期案であり、実装後に実際のコードベースで警告数を見て調整する。

`func-doc` の上限の例:

| 関数の行数 | 上限 | 決めている要素 |
|---|---|---|
| 3 | 3 | free-lines |
| 10 | 3 | free-lines |
| 20 | 4 | max-ratio |
| 50 | 10 | max-ratio |
| 100 | 15 | max-lines |

### 設定の形

```yaml
# .golangci.yml
linters:
  settings:
    custom:
      commentratio:
        type: module
        settings:
          func-doc:
            enabled: true
            free-lines: 3
            max-ratio: 0.2
            max-lines: 15
            require-from: 10
          decl-doc:
            enabled: true
            free-lines: 3
            max-ratio: 0.2
            max-lines: 15
            require-from: 10
          func-body:
            enabled: true
            free-lines: 2
            max-ratio: 0.2
            max-lines: 10
          file:
            enabled: true
            free-lines: 5
            max-ratio: 0.3
            max-lines: 200
```

- 省略したキーはデフォルト値になる
- `enabled: false` でその対象のチェックを無効にする
- `func-body` と `file` に `require-from` を指定した場合は設定エラー
- 単体コマンドでは `-func-doc.max-ratio=0.2` のようなフラグで同じ値を渡す

### 設定の検証

以下の場合は `ErrInvalidSettings` を返し、解析を始めない。

- `free-lines`、`max-lines`、`require-from` が負
- `max-ratio` が負
- `free-lines > max-lines`

## 警告メッセージ

英語で、コメントの先頭位置（GoDoc なしの場合は識別子の位置）に出す。

```
doc comment of ParseUser is too long: 10 comment lines for 10 code lines (max 3)
doc comment of User is too long: 20 comment lines for 30 code lines (max 6)
ParseUser has no doc comment: 20 code lines (doc required from 10)
comments in body of ParseUser are too many: 6 comment lines for 12 code lines (max 2)
file comments are too many: 40 comment lines for 100 code lines (max 30)
```

`file` の位置は `package` キーワードの位置とする（1 行目がライセンスヘッダーでも `//nolint` を置けるように）。

メッセージ中の名前は、関数は `F`、メソッドは `T.M`（ポインタと型パラメータを除いた受信者型名）、まとめた宣言は最初の spec の最初の識別子名とする。

個別の抑制は golangci-lint の `//nolint:commentratio` を使う。単体コマンド用の独自の抑制コメントは作らない。

## 構成

```
github.com/mocoarow/commentratio
├── analyzer.go            # NewAnalyzer(Settings) (*analysis.Analyzer, error)。ファイルごとの処理の流れ
├── settings.go            # Settings / Rule / DocRule / DefaultSettings() / Validate() / DecodeSettings() / ErrInvalidSettings
├── check.go               # 4 つの対象ごとのチェック
├── flags.go               # 単体コマンド用のフラグ登録
├── internal/linecount/    # 行の分類と範囲の行数カウント
├── internal/judge/        # 上限・GoDoc 必須の判定（副作用のない関数のみ）
├── plugin/                # golangci-lint module plugin
└── cmd/commentratio/      # 単体コマンド（singlechecker）
```

### 処理の流れ

1. ファイルごとに `go/scanner`（`ScanComments` モード）でトークンを 1 回走査し、各行について「コードがある」「数えるべきコメントがある」を記録する
2. 行ごとの値の累積和を作り、任意の行範囲のコード行数・コメント行数を O(1) で求められるようにする
3. AST を走査して、関数宣言・GenDecl・関数本体の範囲を取り出し、範囲ごとに判定する
4. ファイル全体は、除外対象（ライセンスヘッダー、パッケージの GoDoc）の行を差し引いて判定する

### `init()` の例外

golangci-lint の module plugin は `func init() { register.Plugin("commentratio", New) }` による登録が必須。コーディング規約では `init()` は禁止だが、代替手段がないため `plugin/` パッケージに限り例外とし、その理由をコード内のコメントに書く。

## テスト

- `internal/linecount`: 行の分類をテーブルテストで検証する（中身のない `//`、ディレクティブ、行末コメント、複数行の `/* */`、ライセンスヘッダー）
- 判定: 上限ちょうどは OK、1 行超えると警告、という境界を `free-lines`・`max-ratio`・`max-lines`・`require-from` のそれぞれで検証する。非公開識別子は GoDoc なしでも警告しないことを検証する
- 設定: 不正な値ごとに `ErrInvalidSettings` が返ることを `ErrorIs` で検証する。デフォルト値を `shouldReturnDefaults` のテストで固定する
- Analyzer 全体: `analysistest` で testdata 内の `// want` と照合する。生成ファイルを無視すること、`_test.go` をチェックすることも含める

## 範囲外

- コメントの内容（文法、TODO の有無など）のチェック
- struct フィールドや interface メソッドに付いたコメントのチェック。これらはコメントだけの行なので `decl-doc` のコード行には数えず、`file` のコメント行にだけ数える
- 単体コマンド用の独自の抑制コメント
