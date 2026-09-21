// Package integration exercises the full EnvDoctor pipeline (contract parse,
// environment source, validation, output) against the realistic fixtures under
// testdata/. The package is test-only by design and runs as part of
// `go test ./...` on every CI platform.
package integration
