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
| e | 単体コマンド | 小 | 済（#6） |
| f | golangci-lint plugin | 小 | 済（#6）。v0.1.0 のタグ済み |
| g | README、`.golangci.yml`、`Taskfile.yml` | 小 | 済（#2、#6）。`custom-gcl` と `calibrate` タスクは不要と判断（2026-10-12） |
| h | cocotola-1.26 での調整 | 中 | 済（#8）。v0.1.1 |
| i | テスト関数を GoDoc なしの判定から外す | 小 | 済（#7） |
| j | 非公開の型のメソッドを GoDoc なしの判定から外す | 小 | 済（#8）。v0.1.1 |
| k | `file` で宣言の GoDoc の要約行を数えない・`ignore-patterns` | 小 | 済（#9、v0.1.2） |

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

`Taskfile.yml` の `custom-gcl`、`calibrate` タスクは作らない（2026-10-12 決定）。custom-gcl の手順は README にあり、値の調整は (h) で済んでいる。

---

### (h) cocotola-1.26 での調整

cocotola-1.26 の各 Go モジュールで `GOFLAGS=-tags=small,medium,large bin/commentratio -json ./...` を実行し、誤検知を見て値を決める。cocotola-1.26 側のファイルは変更しない。決めた値と理由は仕様の「デフォルト値」を参照。

---

### (i) テスト関数の除外

(e)〜(f) の後、このリポジトリ自身にかけたときに、規約どおりのテスト関数が GoDoc なしで警告されたため追加（2026-10-09 決定）。挙動は `testdata/src/testfuncs`、`testfuncslookalike`、`testfuncsnontest`、`testfile` を参照。

---

### (j) 非公開の型のメソッド

(h) で「GoDoc なし」の 4 件がすべて非公開の型の公開名メソッドだったため追加（2026-10-11 決定）。golint・revive の exported ルールと同じ扱いで、公開の型に埋め込まれて昇格するメソッドも判定しない。挙動は `testdata/src/funcdoc*` を参照。

---

### (k) GoDoc の要約行と ignore-patterns

cocotola-1.26 で、コメントを削っても残った 7 件がすべて必須のコメントだったため追加（2026-10-11 決定）。revive が求める 1 行 GoDoc が並ぶファイルと、規約で必須のテストの区切りコメント（given/when/then）。後者はプロジェクト固有なので、利用側が正規表現で指定する。挙動は `testdata/src/filedocsummary*`、`ignorepatterns` を参照。

---

## 品質ゲート

各フェーズの完了時に以下を通す。

- `gofmt` / `goimports`
- `go vet ./...`
- `golangci-lint run`
- `go test -race ./...`、カバレッジ 80% 以上
