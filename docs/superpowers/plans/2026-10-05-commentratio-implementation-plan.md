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

| # | 内容 | 規模 | 進捗 |
|---|---|---|---|
| a | go.mod と設定 | 小 | 済（#1） |
| b | `internal/linecount`: 行の分類と範囲の行数 | 中 | 済（#4） |
| c | `internal/judge`: 上限と GoDoc 必須の判定 | 小 | 済（#3） |
| d | Analyzer への組み込み、4 対象のチェック、フラグ | 大 | 済（#5） |
| e | 単体コマンド | 小 | コミット済み、PR 待ち（feat/cmd-plugin） |
| f | golangci-lint plugin | 小 | コミット済み、PR 待ち（feat/cmd-plugin）。マージ後に v0.1.0 のタグを打つ（`.custom-gcl.yml.example` が参照する） |
| g | README、`.golangci.yml`、`Taskfile.yml` | 小 | 一部済（#2）。README の使い方と `build` タスクはコミット済み、PR 待ち（feat/cmd-plugin）。`custom-gcl` と `calibrate` タスクは未 |
| h | cocotola-1.26 での調整 | 中 | 未着手 |
| i | テスト関数を GoDoc なしの判定から外す | 小 | コミット待ち（feat/skip-test-func-doc。feat/cmd-plugin の上に作成） |

見込み: 本体コード約 600 行、テストと testdata 約 1000 行。

---

### (a) go.mod と設定

挙動は `settings_test.go` を参照。

---

### (b) internal/linecount

挙動は `internal/linecount/linecount_test.go` を参照。

---

### (c) internal/judge

挙動は `internal/judge/judge_test.go` を参照。

---

### (d) Analyzer への組み込み

挙動は `analyzer_test.go`、`flags_test.go`、`testdata/src/` を参照。

---

### (e) cmd/commentratio

テストはない（フラグの挙動は (d) のテストで検証している）。

---

### (f) plugin

挙動は `plugin/plugin_test.go` を参照。

---

### (g) README、golangci 設定、Taskfile

残り: `Taskfile.yml` の `custom-gcl`、`calibrate` タスク。

---

### (h) cocotola-1.26 での調整

1. `go build -o bin/commentratio ./cmd/commentratio`
2. cocotola-1.26 の `Taskfile.yml` にある GO_PROJECTS の各ディレクトリで、`GOFLAGS=-tags=small,medium,large bin/commentratio -json ./...` を実行する（go.work 経由ではなくモジュールごと）
3. メッセージの種類（4 対象 × 多すぎ / GoDoc なし）ごとに件数を集計し、各種類から 10 件ずつ誤検知かどうかを確認する
4. フラグで値を変えながら件数の変化を表にまとめる
5. 決めた値を仕様の表、`DefaultSettings`、`Test_DefaultSettings_shouldReturnDefaults_whenCalled` に同時に反映する

cocotola-1.26 側のファイルは変更しない。

### (i) テスト関数の除外

(e)〜(f) の後、このリポジトリ自身にかけたときに、規約どおりのテスト関数が GoDoc なしで警告されたため追加（2026-10-09 決定）。挙動は `testdata/src/testfuncs`、`testfuncslookalike`、`testfuncsnontest`、`testfile` を参照。

---

## 品質ゲート

各フェーズの完了時に以下を通す。

- `gofmt` / `goimports`
- `go vet ./...`
- `golangci-lint run`
- `go test -race ./...`、カバレッジ 80% 以上
