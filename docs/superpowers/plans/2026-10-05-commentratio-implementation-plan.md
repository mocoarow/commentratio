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

作成するファイル: `internal/judge/doc.go`、`judge.go`、`judge_test.go`

```go
type Limits struct {
    FreeLines int
    MaxLines  int
    MaxRatio  float64
}

const ratioEpsilon = 1e-9

func Limit(code int, l Limits) int // min(MaxLines, max(FreeLines, int(math.Floor(float64(code)*MaxRatio + ratioEpsilon))))
func TooMany(comment, code int, l Limits) bool
func RequiresDoc(code, requireFrom int, exported, hasDoc bool) bool // requireFrom > 0 && code >= requireFrom && exported && !hasDoc
```

先に書くテスト:

- `Test_Limit_shouldReturnSpecExample_whenCodeLinesVary`: テーブル。3→3、10→3、20→4、50→10、100→15（デフォルトの func-doc）
- `Test_Limit_shouldReturnFreeLines_whenCodeIsZero`
- `Test_Limit_shouldReturnFreeLines_whenRatioIsZero`
- `Test_Limit_shouldNotUnderflow_whenProductIsInexact`: 100×0.29 → 29
- `Test_TooMany_shouldReturnFalse_whenCommentEqualsLimit`: テーブル。free で決まる（code 10, comment 3）、ratio で決まる（code 20, comment 4）、max で決まる（code 100, comment 15）
- `Test_TooMany_shouldReturnTrue_whenCommentExceedsLimitByOne`: 上の 3 ケースで comment 4 / 5 / 16
- `Test_RequiresDoc_shouldReturnTrue_whenCodeEqualsRequireFrom`: 10 / 10
- `Test_RequiresDoc_shouldReturnFalse_whenNotRequired`: テーブル。code 9 / 10、requireFrom 0、非公開、doc あり

---

### (d) Analyzer への組み込み

作成するファイル: `analyzer.go`、`check.go`、`flags.go`、`analyzer_test.go`、`flags_test.go`、`testdata/src/...`

```go
const Name = "commentratio"

func NewAnalyzer(s Settings) (*analysis.Analyzer, error) // Validate → runner → Flags 登録 → Run

type runner struct{ settings *Settings }

func (r *runner) run(pass *analysis.Pass) (any, error) // 毎回 Validate する（フラグは NewAnalyzer の後に Parse されるため）

type fileContext struct {
    pass  *analysis.Pass
    tf    *token.File
    lines linecount.Lines
    file  *ast.File
}

func (c fileContext) line(p token.Pos) int { return c.tf.PositionFor(p, false).Line }
```

- 行番号は `PositionFor(p, false)` で取る。`//line` で補正された行番号を使うと、自前の走査結果とずれるため

run の流れ:

1. 生成ファイル（`ast.IsGenerated(f)`）は飛ばす
2. `tf := pass.Fset.File(f.Pos())`、`pass.ReadFile(tf.Name())`（エラーは `"read %s: %w"` でラップ）、`linecount.Classify`
3. `f.Decls` だけを走査する。FuncLit とローカルな宣言は外側の関数本体の一部として扱う

メッセージの書式は定数にする:

```go
const (
    msgDocTooLong     = "doc comment of %s is too long: %d comment lines for %d code lines (max %d)"
    msgDocMissing     = "%s has no doc comment: %d code lines (doc required from %d)"
    msgBodyTooMany    = "comments in body of %s are too many: %d comment lines for %d code lines (max %d)"
    msgFileTooMany    = "file comments are too many: %d comment lines for %d code lines (max %d)"
)
```

#### func-doc

- コード範囲: `d.Pos()`（`func` キーワード。Doc は含まない）から、Body があれば `Body.End()`、なければ `Type.End()` まで
- doc のコメント行数: `Comment(line(Doc.Pos()), line(Doc.End()))`
- 本体内のコメント行はコード行に数えない。doc 範囲と本体範囲は重ならない
- GoDoc あり: 数えるべき doc 行が 1 行以上（ディレクティブだけの doc は GoDoc なし）
- 公開判定: `d.Name.IsExported()`。非公開型の公開メソッドも対象
- 名前: 関数は `F`、メソッドは `T.M`
- 位置: 多すぎは `Doc.List[0].Pos()`、GoDoc なしは `d.Name.Pos()`

#### func-body

- Body が nil なら飛ばす。範囲は `line(Body.Lbrace)`〜`line(Body.Rbrace)`
- `{` の行の行末コメントも本体のコメントとして数える
- 位置: 範囲内で最初の「数える」コメント。`f.Comments` を走査し、`CommentLineFlags` のいずれかが true のものを選ぶ

#### decl-doc

- import 以外の GenDecl が対象
- `GenDecl.Doc` がある場合: GenDecl 全体（`d.Pos()`〜`d.End()`）で多すぎを判定する
- 括弧付き宣言の spec に Doc がある場合（ValueSpec / TypeSpec の Doc）: その spec の範囲で判定する。GenDecl の doc とは別に判定する
- 括弧なしの宣言では、パーサーが doc を GenDecl.Doc に付ける
- GoDoc なしは spec ごとに判定する。条件: GenDecl にも spec にも数えるべき doc 行がない、spec に公開名がある（ValueSpec の Names のいずれか、または TypeSpec.Name）、`Code(spec 範囲) >= require-from`。位置は最初の公開名
- struct フィールドや interface メソッドのコメントは、コメントだけの行なのでコード行に数えない
- 名前: 最初の spec の最初の識別子名

#### file

- `pkgLine := line(f.Package)`
- コード行: `Code(1, N)`。コメント行: `Comment(pkgLine, N)`
- package 句より前のコメントがすべて除かれるので、ライセンスヘッダーとパッケージの GoDoc の両方が除外される。package 行の行末コメントは数える
- 位置: `f.Package`

#### flags.go

```go
func registerRuleFlags(fs *flag.FlagSet, prefix string, r *Rule)       // <prefix>.enabled / .free-lines / .max-ratio / .max-lines
func registerDocRuleFlags(fs *flag.FlagSet, prefix string, r *DocRule) // 上記に加えて <prefix>.require-from
```

- フラグはネストした Rule のフィールドを直接指すポインタに束縛する。キー名は定数（`keyFreeLines` など）
- `func-body` と `file` には `require-from` を登録しない（指定すると flag パッケージの未定義エラー）

#### testdata の方針

- 各テストでは対象外の Rule を `Enabled=false` にし、他の対象の警告が混ざらないようにする
- 少ない行数で境界を作れるよう、テスト用の設定を使う（free 2 / ratio 0.5 / max 4 / require-from 5）
- `// want` は doc の 1 行目の末尾に `// Foo does x. // want "..."` の形で書き、コメント行数を増やさない。識別子の行や package 行に書く場合は、行末コメントとして 1 行増えることを前提に期待値を設計する

先に書くテスト:

- `Test_NewAnalyzer_shouldReturnErrInvalidSettings_whenSettingsAreInvalid`
- `Test_NewAnalyzer_shouldNotReportFuncDoc_whenDocIsWithinLimit`: free / ratio / max の 3 要素それぞれでちょうど上限、段落区切りの `//`、ディレクティブを含む doc
- `Test_NewAnalyzer_shouldReportFuncDoc_whenDocExceedsLimit`: 3 要素それぞれで +1、メソッド
- `Test_NewAnalyzer_shouldReportMissingFuncDoc_whenExportedFuncReachesRequireFrom`: ちょうど require-from、公開メソッド、ディレクティブだけの doc
- `Test_NewAnalyzer_shouldNotReportMissingFuncDoc_whenNotRequired`: require-from−1 行、非公開、`_`、本体のない関数
- `Test_NewAnalyzer_shouldNotReportDeclDoc_whenDocIsWithinLimit`
- `Test_NewAnalyzer_shouldReportDeclDoc_whenDocExceedsLimit`: 括弧なし type、括弧付きで GenDecl に doc、括弧付きで spec に doc
- `Test_NewAnalyzer_shouldReportMissingDeclDoc_whenExportedSpecReachesRequireFrom`
- `Test_NewAnalyzer_shouldNotReportMissingDeclDoc_whenGroupOrSpecHasDoc`: GenDecl だけに doc、spec だけに doc、非公開だけの spec
- `Test_NewAnalyzer_shouldNotReportFuncBody_whenCommentsAreWithinLimit`
- `Test_NewAnalyzer_shouldReportFuncBody_whenCommentsExceedLimit`: 行末コメントを含む
- `Test_NewAnalyzer_shouldNotReportFile_whenOnlyHeaderAndPackageDocAreLong`: 長いライセンスヘッダーと長いパッケージ doc を持つ doc.go
- `Test_NewAnalyzer_shouldReportFile_whenCommentsExceedLimit`
- `Test_NewAnalyzer_shouldSkipFile_whenFileIsGenerated`
- `Test_NewAnalyzer_shouldCheckFile_whenFileIsTestFile`
- `Test_NewAnalyzer_shouldNotReport_whenRuleIsDisabled`: テーブル。4 対象それぞれ
- `Test_NewAnalyzer_shouldRegisterFlag_whenCreated`: テーブル。18 個のフラグ名を `Lookup`
- `Test_NewAnalyzer_shouldApplyFlagValue_whenFlagIsSet`: `Flags.Set("func-doc.max-lines", "2")` の後に analysistest で確認
- `Test_Analyzer_Run_shouldReturnErrInvalidSettings_whenFlagValueIsInvalid`: `Set("func-doc.free-lines", "-1")` の後に Run。検証は pass に触る前に行う
- `Test_Rule_jsonTags_shouldMatchFlagKeys`: reflect で json タグとフラグキー定数のずれを検知する
- `Test_NewAnalyzer_shouldReportSpecMessages_whenDefaultSettings`: デフォルト値で、仕様のメッセージ例と同じ文字列になる

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
