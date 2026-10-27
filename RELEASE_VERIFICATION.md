# Release verification — fastgate v0.1.1

Verified locally on October 27, 2026, Linux x86-64. Toolchains: Python 3.14.8, Rust 1.98.1, Node.js 26.8.1, Go 1.27.1 and Mojo 1.1.0, as applicable. CI covers additional versions/platforms where configured; consult the workflow run for the release commit rather than an older green run.

## Checks

```sh
go test -race -v ./...
go vet ./...
go build .
```

**Local result: PASS.** 6 Go tests, race detector, vet, executable build and triage demo. Local HTTP fixtures verify preserved request fields/authentication, streamed response bytes, status/errors, explicit tool choice, structured content and rejected upstream URLs.

The Python build command requires the `build` package; Unity checks also require `mypy` and `ruff`. Native Mojo experiments are optional and require Mojo 1.1.0. CLI acceptance used disposable fixtures and no real customer data.

## Release changes

Preserve provider request fields and streaming; retain explicit tool choices and unknown tools; validate upstream URLs; bind to localhost; add HTTP regressions.

The README and repository summary now explain the supported behavior, installation, and limits without unverified speed or security promises.

## Limits

No paid/live provider requests were used. Keyword filtering can affect task quality. Go does not call the separate Mojo kernel; savings are illustrative estimates.

Git history was checked for common credential/key patterns. Matches in Envghost were synthetic scanner fixtures; no live credentials were identified by that scan. This is a limited pattern check, not a comprehensive security certification.
