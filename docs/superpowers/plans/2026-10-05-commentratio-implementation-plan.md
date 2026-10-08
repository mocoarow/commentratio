# commentratio 実装計画

仕様: [2026-10-05-commentratio-design.md](../specs/2026-10-05-commentratio-design.md)

## 前提

- モジュールパス: `github.com/mocoarow/commentratio`
- 提供形態: 単体コマンド（singlechecker とフラグ）、golangci-lint module plugin、Go API `NewAnalyzer(Settings)`
- 進め方: 各フェーズでテストを先に書く（RED → GREEN → リファクタリング）。カバレッジ 80% 以上
- 境界テストの基準: 上限ちょうどは OK、1 行超えると警告

## 確定した判断

計画作成時に確認し、決定した事項。仕様書にも反映済み。

| # | 項目 | 決定 |
|---|---|---|
| 1 | `file` の警告位置 | `package` キーワードの位置（1 行目がライセンスヘッダーでも `//nolint` を置けるように） |
| 2 | メッセージ中の名前 | 関数は `F`、メソッドは `T.M`（ポインタと型パラメータを除いた受信者型名）。まとめた宣言は最初の spec の最初の識別子名 |
| 3 | 細かい数え方 | `/* */` 内の装飾用 `*` だけの行は数えない。複数行 raw string 内の空行はコード行に数える |
| 4 | ディレクティブだけの doc | GoDoc なしとして扱う（数えるべき doc 行が 0 行なら GoDoc なし） |
| 5 | 設定のデコード | `register.DecodeSettings` を使わず、自前の `DecodeSettings` を実装する |

仕様からの設計上の微修正:

- 判定ロジックを `internal/judge` に切り出す（import の循環を避けるため、値をプリミティブで受け取る）
- `DocRule` 型を追加し、`require-from` は GoDoc 用の設定だけが持つ
- 比率の計算に `ratioEpsilon = 1e-9` を加える（100×0.29 が 28.999… になって上限が 1 小さくなるのを防ぐ）

## フェーズ

| # | 内容 | 規模 |
|---|---|---|
| a | go.mod と設定 | 小 |
| b | `internal/linecount`: 行の分類と範囲の行数 | 中 |
| c | `internal/judge`: 上限と GoDoc 必須の判定 | 小 |
| d | Analyzer への組み込み、4 対象のチェック、フラグ | 大 |
| e | 単体コマンド | 小 |
| f | golangci-lint plugin | 小 |
| g | README、`.golangci.yml`、`Taskfile.yml` | 小 |
| h | cocotola-1.26 での調整 | 中 |

見込み: 本体コード約 600 行、テストと testdata 約 1000 行。

---

### (a) go.mod と設定

実装済み。挙動は `settings_test.go` を参照。

---

### (b) internal/linecount

実装済み。挙動は `internal/linecount/linecount_test.go` を参照。

---

### (c) internal/judge

実装済み。挙動は `internal/judge/judge_test.go` を参照。

---

### (d) Analyzer への組み込み

実装済み。挙動は `analyzer_test.go`、`flags_test.go`、`testdata/src/` を参照。

---

### (e) cmd/commentratio

作成するファイル: `cmd/commentratio/doc.go`、`main.go`

- `commentratio.NewAnalyzer(commentratio.DefaultSettings())` が失敗したら `slog.Error("create analyzer", "error", err)` の後に `os.Exit(1)`（`os.Exit` は main の中だけ）
- 成功したら `singlechecker.Main(a)`
- テストは書かない（フラグの挙動は (d) で検証済み）。カバレッジ計測の対象外にする

---

### (f) plugin

作成するファイル: `plugin/doc.go`、`plugin.go`、`plugin_test.go`、`.custom-gcl.yml.example`

```go
// golangci-lint の module plugin は init での登録が必須で、代替手段がないため例外的に init を使う。
func init() { register.Plugin(commentratio.Name, New) } //nolint:gochecknoinits

func New(settings any) (register.LinterPlugin, error) // commentratio.DecodeSettings → fmt.Errorf("decode settings: %w", err)

type Plugin struct{ settings commentratio.Settings }

func (p Plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) // NewAnalyzer。エラーは "build analyzer: %w"
func (p Plugin) GetLoadMode() string                           // register.LoadModeSyntax
```

テスト:

- `Test_New_shouldReturnPlugin_whenSettingsAreValid`
- `Test_New_shouldReturnErrInvalidSettings_whenSettingsAreInvalid`
- `Test_Plugin_BuildAnalyzers_shouldReturnCommentratioAnalyzer_whenCalled`
- `Test_Plugin_GetLoadMode_shouldReturnSyntax_whenCalled`

`.custom-gcl.yml.example`: `version`（golangci-lint のバージョン）と `plugins: [{module: github.com/mocoarow/commentratio, import: github.com/mocoarow/commentratio/plugin, version: v0.1.0}]`。ローカル開発用に `path: .` の例も載せる。

---

### (g) README、golangci 設定、Taskfile

- README:
  - 単体コマンドのインストールと全フラグの一覧
  - `golangci-lint custom` の手順
  - `.golangci.yml` の例（`linters.enable: [commentratio]` と settings ブロック）
  - 判定式と行数の数え方の要約
  - `//nolint:commentratio` の置き場所
  - 制約: cgo のファイルは生成ファイル扱いでチェックされない
- `.golangci.yml`（v2）: cocotola-1.26 の設定をもとに作る
- `Taskfile.yml`（cocotola-1.26 に合わせる）: `fmt`、`lint`、`test`（`go test -race -coverprofile`）、`cover`（80% 未満で失敗）、`build`、`custom-gcl`、`check`（fmt / vet / lint / test）、`calibrate`

---

### (h) cocotola-1.26 での調整

1. `go build -o bin/commentratio ./cmd/commentratio`
2. cocotola-1.26 の `Taskfile.yml` にある GO_PROJECTS の各ディレクトリで、`GOFLAGS=-tags=small,medium,large bin/commentratio -json ./...` を実行する（go.work 経由ではなくモジュールごと）
3. メッセージの種類（4 対象 × 多すぎ / GoDoc なし）ごとに件数を集計し、各種類から 10 件ずつ誤検知かどうかを確認する
4. フラグで値を変えながら件数の変化を表にまとめる
5. 決めた値を仕様の表、`DefaultSettings`、`Test_DefaultSettings_shouldReturnDefaults_whenCalled` に同時に反映する

cocotola-1.26 側のファイルは変更しない。

## リスクと対策

| リスク | 対策 |
|---|---|
| `// want` が行末コメントとして数えられ、期待値がずれる | doc 行の末尾に `// ... // want` 形式で書く。識別子行や package 行に書く場合は 1 行増える前提で設計する |
| analysistest がテストファイルを読むとき、テスト用バリアントで警告が重複する | (d) の最初に `_test.go` 用の小さな testdata で確認する。重複したら testdata を分ける |
| `pass.ReadFile` が x/tools のバージョンによって使えない | 最新の x/tools に固定する。使えなければ `os.ReadFile(tf.Name())` に切り替える |
| cgo のファイルが生成ファイル扱いになる | README に制約として書く |
| golangci-lint から渡る settings の型が想定と違う | `DecodeSettings` が JSON を経由するので吸収できる。(f) の後に実際の custom バイナリで確認する |
| 浮動小数点の誤差で上限が 1 小さくなる | `ratioEpsilon` を入れ、テストで固定する |

## 品質ゲート

各フェーズの完了時に以下を通す。

- `gofmt` / `goimports`
- `go vet ./...`
- `golangci-lint run`
- `go test -race ./...`、カバレッジ 80% 以上
