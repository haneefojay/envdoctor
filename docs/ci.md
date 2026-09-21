# EnvDoctor CI

Continuous integration lives in `.github/workflows/ci.yml`. It runs the same
checks on three operating systems and cross-compiles every documented target.

## Jobs

### `test`

Runs natively on three GitHub-hosted runners:

| Runner          | Architecture |
| --------------- | ------------ |
| `ubuntu-latest` | amd64        |
| `windows-latest`| amd64        |
| `macos-latest`  | arm64        |

The steps on each runner:

1. **gofmt check** - `gofmt -l .` must be empty. It runs under a POSIX shell
   (`shell: bash`) on every runner, including Windows.
2. **go vet** - `go vet ./...`.
3. **build** - `go build ./...`.
4. **test** - `go test ./...`.
5. **race test** - `go test -race ./...`, **only on Linux and macOS**. The
   race detector needs a C toolchain, which GitHub's Windows runners do not
   provide by default.

### `crossbuild`

Compile-only coverage for every documented target. GitHub-hosted runners only
expose amd64 (Linux/Windows) and arm64 (macOS), so the matrix cross-compiles
the remaining combinations with a disabled C toolchain:

| `goos`   | `goarch` | How it is covered     |
| -------- | -------- | --------------------- |
| linux    | amd64    | native (`test` job) + crossbuild |
| linux    | arm64    | crossbuild            |
| windows  | amd64    | native (`test` job) + crossbuild |
| windows  | arm64    | crossbuild            |
| darwin   | amd64    | crossbuild            |
| darwin   | arm64    | native (`test` job) + crossbuild |

Each matrix entry runs `go build ./...` with `GOOS`, `GOARCH`, and
`CGO_ENABLED=0` set.

## Behavioral guarantees CI verifies

Beyond compiling and passing the test suite, CI verifies:

- **Determinism:** golden output, ordering, and line-ending tests assert
  byte-identical human/JSON/generated output on every platform (LF-only).
- **Security:** `cmd/envdoctor/security_test.go` proves realistic secrets never
  reach stdout, stderr, or generated files; `internal/security` statically
  blocks forbidden capabilities in shipping code.
- **Cross-platform posture:** LF-only output, CRLF-tolerant dotenv parsing,
  canonical-case environment names, and no ANSI sequences are all asserted on
  every OS.

## Running the same checks locally

```text
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

Race tests require a C toolchain:

```text
go test -race ./...     # Linux/macOS, or Windows with gcc installed
```