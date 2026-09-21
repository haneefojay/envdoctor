# Release process

EnvDoctor releases follow [Semantic Versioning](https://semver.org/). A tagged
push (`vMAJOR.MINOR.PATCH`) is picked up by the
`.github/workflows/release.yml` pipeline, which runs the full QA gate, packages
every documented target, writes checksums, and opens a **draft** GitHub
Release for a human to review and publish.

## Compatibility surfaces

These behaviors are part of EnvDoctor's public contract and must not change
within a MAJOR version:

- **Contract semantics** - the EnvDoctor Contract Profile: supported keywords,
  type interpretation, required/default/secret rules, enum/const matching.
- **Diagnostic codes** - the stable identifiers such as `ENV_MISSING`,
  `CONTRACT_INVALID`, `ENV_FILE_INVALID`. Messages may evolve; codes may not.
- **Machine-readable output** - the `--json` shape (`{"valid", "diagnostics"}`
  with stable field structures and deterministic ordering).
- **Exit codes** - `0` valid, `1` validation failure, `2` usage/contract/source
  error, `3` unexpected internal error.

Breaking a compatibility surface requires a MAJOR version bump and a changelog
entry explaining the migration.

## Versioning

- The current version lives in `cmd/envdoctor/main.go` as `var version`.
- `envdoctor version` must always report the latest released version: the
  default value is what `go install github.com/haneefojay/envdoctor/cmd/envdoctor@<tag>`
  embeds, because `go install` cannot pass `-ldflags`. The release pipeline
  fails if the tag and the embedded default disagree.
- Tagged releases are additionally built with `-X main.version=<tag>` injected
  so distributed artifacts embed the exact tag.

## Release checklist

1. Move the `[Unreleased]` section in `CHANGELOG.md` to a new
   `## vX.Y.Z` section (Keep a Changelog format).
2. Bump `var version` in `cmd/envdoctor/main.go` to `X.Y.Z` (no `v`).
3. Commit and push. CI runs on the commit.
4. Tag: `git tag -a vX.Y.Z -m "vX.Y.Z"` and `git push origin vX.Y.Z`.
5. The Release workflow builds packages and opens a draft release.
6. Review the artifacts and changelog notes, then publish the draft.

## Artifacts

The `tools/release` packager builds every target with CGO disabled and a fixed
member timestamp, so identical inputs produce byte-identical archives:

```text
dist/envdoctor_X.Y.Z_linux_amd64.tar.gz
dist/envdoctor_X.Y.Z_linux_arm64.tar.gz
dist/envdoctor_X.Y.Z_windows_amd64.zip
dist/envdoctor_X.Y.Z_windows_arm64.zip
dist/envdoctor_X.Y.Z_darwin_amd64.tar.gz
dist/envdoctor_X.Y.Z_darwin_arm64.tar.gz
dist/SHA256SUMS
dist/NOTES.md
```

Each archive contains a single `envdoctor_X.Y.Z_<os>_<arch>/` directory with
the `envdoctor` (or `envdoctor.exe`) binary, the `README.md`, and the
`LICENSE`. `SHA256SUMS` lists every archive in `sha256sum -c` format.

## Verifying a release

```text
sha256sum -c SHA256SUMS
# or, per file:
sha256sum -c SHA256SUMS --ignore-missing
```

Extract and check the version:

```text
tar xzf envdoctor_vX.Y.Z_linux_amd64.tar.gz
./envdoctor_X.Y.Z_linux_amd64/envdoctor version   # -> envdoctor X.Y.Z
```

## Local packaging

Run the packager from the repository root. It uses the `go` command from PATH:

```text
go run ./tools/release -version vX.Y.Z -notes CHANGELOG.md -out dist
```

Flags: `-version` (required, semantic), `-out` (default `dist`), `-notes` (a
Keep-a-Changelog file; writes `<out>/NOTES.md` with the version's section).
Run `go run ./tools/release -h` for the full help text.