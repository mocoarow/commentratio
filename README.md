# commentratio

`commentratio` is a Go static analysis tool (linter) that reports comments that are too long for the code they describe, such as a 10-line GoDoc on a 10-line function. It also reports exported declarations that are long enough to need a GoDoc but have none.

Existing linters can limit the length of a comment, but not its length relative to the code it documents.

> [!NOTE]
> This project is under development. The default values are not tuned yet.

## Usage

### Standalone command

```bash
go install github.com/mocoarow/commentratio/cmd/commentratio@latest
commentratio ./...
commentratio -func-doc.max-lines=20 -file.enabled=false ./...
```

Every setting in [Settings](#settings) is also a flag named `<key>.<setting>`.

### golangci-lint plugin

Copy [.custom-gcl.yml.example](.custom-gcl.yml.example) to `.custom-gcl.yml`, run `golangci-lint custom` to build `./bin/custom-gcl`, and enable the linter in `.golangci.yml`:

```yaml
version: "2"
linters:
  enable:
    - commentratio
  settings:
    custom:
      commentratio:
        type: module
        settings:
          func-doc:
            max-lines: 20
```

To suppress a report, add `//nolint:commentratio` on its own line above the reported line, or as a line of the reported doc comment. A missing GoDoc report, or the file report on the `package` line, can also be suppressed at the end of the reported line.

## Checks

| Target | Key | Too many comments | Missing GoDoc |
| --- | --- | --- | --- |
| GoDoc of functions and methods | `func-doc` | Yes | Yes (exported only) |
| GoDoc of types, constants and variables | `decl-doc` | Yes | Yes (exported only) |
| Comments inside function bodies | `func-body` | Yes | No |
| Comments in the whole file | `file` | Yes | No |

`func-body` and `file` have no lower limit, so that nobody pads code with comments just to silence a warning.

## How lines are counted

- **Code lines**: lines in the target range, excluding blank lines and lines that contain only comments.
- **Comment lines**: lines that contain a comment with content. The following are not counted:
  - empty `//` lines used as paragraph breaks, and blank or `*`-only lines inside `/* */`
  - directives: comments starting with `//go:`, `//nolint`, `//lint:`, or `//line` with a trailing space
  - for `file` only: comments before the `package` clause (license headers) and the package GoDoc
- A line with a trailing comment (`x := 1 // note`) counts as both a code line and a comment line.
- Every line of a multi-line raw string counts as a code line, including blank lines inside it.
- Generated files are skipped. `_test.go` files are checked.
- Files that `import "C"` are skipped, because cgo hands the analyzer generated files instead.

## Rules

### Too many comments

```text
limit = min(max-lines, max(free-lines, floor(code_lines × max-ratio)))
report if comment_lines > limit
```

- `free-lines`: comments up to this many lines are always allowed, whatever the ratio
- `max-ratio`: the maximum ratio of comment lines to code lines
- `max-lines`: the absolute maximum, however long the code is

### Missing GoDoc

```text
report if code_lines >= require-from and the identifier is exported and it has no GoDoc
```

A GoDoc that contains only directives counts as missing. Setting `require-from` to `0` disables this check.

## Settings

| Key | free-lines | max-ratio | max-lines | require-from |
| --- | --- | --- | --- | --- |
| `func-doc` | 3 | 0.2 | 15 | 10 |
| `decl-doc` | 3 | 0.2 | 15 | 10 |
| `func-body` | 2 | 0.2 | 10 | — |
| `file` | 5 | 0.3 | 200 | — |

The table shows the default values. For the golangci-lint plugin, settings go under `linters.settings.custom.commentratio.settings` in `.golangci.yml`. For example, to allow longer function GoDocs and disable the file check:

```yaml
func-doc:
  max-ratio: 0.3
  max-lines: 20
file:
  enabled: false
```

- Omitted keys keep their default values.
- `enabled: false` disables the check for that target.
- Settings are rejected if a value is negative, `max-ratio` is not a finite number, `free-lines` is greater than `max-lines`, a key is unknown, or `require-from` is set on `func-body` or `file`.

## Development

Requires Go 1.26, [golangci-lint](https://golangci-lint.run/) v2 and [Task](https://taskfile.dev/).

```bash
task fmt    # format
task lint   # go vet and golangci-lint
task test   # tests with race detector and coverage
task cover  # tests, failing if coverage is below 80%
task check  # fmt, lint and cover
task build  # build bin/commentratio
```

## License

[MIT](LICENSE)
