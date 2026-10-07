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

作成するファイル: `go.mod`、`doc.go`、`settings.go`、`settings_test.go`

- `go mod init github.com/mocoarow/commentratio`。依存は `golang.org/x/tools`、`github.com/golangci/plugin-module-register`、`github.com/stretchr/testify`
- go 行は `go mod init` が出力したものをそのまま使う（plugin 利用者に最新の Go を強制しないため）

```go
type Rule struct {
    Enabled   bool    `json:"enabled"`
    FreeLines int     `json:"free-lines"`
    MaxRatio  float64 `json:"max-ratio"`
    MaxLines  int     `json:"max-lines"`
}

type DocRule struct {
    Rule
    RequireFrom int `json:"require-from"`
}

type Settings struct {
    FuncDoc  DocRule `json:"func-doc"`
    DeclDoc  DocRule `json:"decl-doc"`
    FuncBody Rule    `json:"func-body"`
    File     Rule    `json:"file"`
}

var ErrInvalidSettings = errors.New("invalid settings")

func DefaultSettings() Settings
func (s Settings) Validate() error // fmt.Errorf("%w: func-doc.free-lines must be >= 0, got %d", ErrInvalidSettings, v)
func DecodeSettings(raw any) (Settings, error)
```

`DecodeSettings` の手順:

1. `json.Marshal(raw)` で JSON にする
2. `DefaultSettings()` で初期化した値に、`DisallowUnknownFields()` を付けた `json.Decoder` でデコードする。値型のネスト構造体なので、データにあるキーだけが上書きされ、省略したキーはデフォルト値（`enabled: true` を含む）のまま残る
3. `Validate` を呼ぶ

デコードエラー（未知のキー、型の不一致）は `ErrInvalidSettings` でラップする。`func-body` と `file` の `require-from` は `Rule` にフィールドがないので、未知のキーとしてエラーになる。

`Validate` は `enabled: false` の Rule にも適用する。

先に書くテスト:

- `Test_DefaultSettings_shouldReturnDefaults_whenCalled`: 仕様の表の全値と `Enabled` を固定する
- `Test_Settings_Validate_shouldReturnNil_whenSettingsAreValid`: テーブル。デフォルト値、すべて 0、`free-lines == max-lines`、`max-ratio = 0`
- `Test_Settings_Validate_shouldReturnErrInvalidSettings_whenValueIsInvalid`: テーブル。4 対象それぞれで free / max / require が負、ratio が負・NaN・+Inf、free > max
- `Test_DecodeSettings_shouldReturnDefaults_whenRawIsNil`
- `Test_DecodeSettings_shouldKeepDefaults_whenKeyIsOmitted`: `func-doc.max-ratio` だけ指定し、他の値と `enabled=true` が残る
- `Test_DecodeSettings_shouldDisableRule_whenEnabledIsFalse`
- `Test_DecodeSettings_shouldApplyValue_whenKeyIsSet`: デフォルトと異なる値（例: 7, 0.45）で確認する
- `Test_DecodeSettings_shouldReturnErrInvalidSettings_whenRequireFromIsSetOnNonDocRule`: テーブル `{func-body, file}`
- `Test_DecodeSettings_shouldReturnErrInvalidSettings_whenKeyIsUnknown`
- `Test_DecodeSettings_shouldReturnErrInvalidSettings_whenTypeMismatches`
- `Test_DecodeSettings_shouldReturnErrInvalidSettings_whenValueIsOutOfRange`

---

### (b) internal/linecount

作成するファイル: `internal/linecount/doc.go`、`linecount.go`、`linecount_test.go`

```go
var ErrScan = errors.New("scan source")

type Lines struct{ code, comment []int } // 行ごとの値の累積和。len = 行数+1、1 始まり

func Classify(filename string, src []byte) (Lines, error)
func (l Lines) Code(from, to int) int    // 閉区間。範囲外は丸め、from > to なら 0
func (l Lines) Comment(from, to int) int
func (l Lines) LineCount() int
func IsDirective(text string) bool
func CommentLineFlags(lit string) []bool // コメントトークンの各行を「数えるか」
```

分類のルール:

- 自前の `token.NewFileSet()` と `scanner.ScanComments` で 1 回だけ走査する（共有の `pass.Fset` は書き換えない）。エラーハンドラでエラーを数え、1 件以上なら `ErrScan` を返す
- 自動挿入のセミコロン（`SEMICOLON` で lit が `"\n"`）は無視する
- その他のトークンは、開始行から `開始行 + strings.Count(lit, "\n")` までをコード行にする。複数行 raw string は空行も含めて全行コード行。raw string は `\r` が除かれるので、行数をバイト長から計算しない
- `//` コメント:
  - `\r` を除いたうえで `IsDirective` に当たれば数えない。対象は `//go:`、`//nolint`、`//lint:`、`//line ` の前方一致で、`//` の直後にスペースがない形に限る
  - `//` を除いて TrimSpace した結果が空なら数えない
  - 行末コメントでもその行のコメントフラグを立てる。コードフラグとは独立
- `/* */` コメント:
  - `\n` で分割し、先頭要素から `/*`、末尾要素から `*/` を除く
  - 各要素を TrimSpace し、先頭の装飾用 `*` を 1 個除く
  - 結果が空でない行だけ数える。`/* */` や `/**/` は数えない
- 行ごとに真偽値で記録するので、同じ行に複数トークンがあっても重複しない

先に書くテスト:

- `Test_Classify_shouldCountCodeLine_whenLineHasToken`
- `Test_Classify_shouldNotCountLine_whenLineIsBlank`
- `Test_Classify_shouldCountCommentLine_whenCommentHasContent`: テーブル。`// text`、`/* text */`、ライセンスヘッダー
- `Test_Classify_shouldNotCountCommentLine_whenCommentIsEmptyOrDirective`: テーブル。`//`、`//   `、`//go:generate x`、`//go:build x`、`//nolint:errcheck`、`//lint:ignore SA1`、`//line a.go:1`、`/* */`
- `Test_Classify_shouldCountCommentLine_whenTextOnlyResemblesDirective`: テーブル。`// go:generate`、`//lines`、`// nolint`
- `Test_Classify_shouldSkipBlankInnerLines_whenBlockCommentSpansLines`: 空行と `*` だけの行を含む
- `Test_Classify_shouldCountBothCodeAndComment_whenLineHasTrailingComment`: テーブル。`x := 1 // note`、`/* a */ x := 1`、複数行 `/* */` の最終行の後ろにコード
- `Test_Classify_shouldCountEveryLineAsCode_whenRawStringSpansLines`
- `Test_Classify_shouldClassifySameAsLF_whenSourceUsesCRLF`
- `Test_Lines_Code_shouldReturnZero_whenRangeIsEmpty`
- `Test_Lines_Code_shouldClampRange_whenRangeExceedsFile`
- `Test_Classify_shouldReturnErrScan_whenSourceHasIllegalCharacter`: NUL など

---

### (c) internal/judge

実装済み。挙動は `internal/judge/judge_test.go` を参照。

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
