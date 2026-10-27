# Fastgate ⚡

**Less tool clutter. More room for the task.**

A local Go reverse proxy that filters recognized tool names using simple prompt keywords. Inspect the routing decision, pass requests to your model provider, and see estimated changes in tool-schema overhead.

## Try it

Requires Go 1.27.1+:

```sh
git clone https://github.com/Buzburg/fastgate.git
cd fastgate
go build -o fastgate .
./fastgate triage "Read package.json and run tests"
./fastgate start --port 8080 --upstream https://api.anthropic.com
```

The proxy binds to **127.0.0.1**. Point a compatible client's base URL at `http://127.0.0.1:8080`; retain its normal provider authentication. For OpenAI-compatible clients, configure the upstream origin and use the client's expected `/v1` base path. Do not duplicate `/v1` in the upstream URL.

## Behavior

- Filters tools for POST `/v1/chat/completions` and `/v1/messages` with a plain-text user message.
- Preserves other JSON fields, authentication headers, response status, and streamed response bytes.
- Passes explicit tool-choice requests and non-text prompts through unchanged.
- Keeps unrecognized tool names and the full catalog when intent is unknown.
- Exposes `/healthz` and `/metrics`; limits rewritten request bodies to 16 MiB.

This is a **heuristic developer beta**. A recognized tool can still be needed later in a task even if the latest prompt does not mention it. Compare completion quality before routing important work through it. Tests use local HTTP fixtures; they do not certify every provider or client version.

The Go proxy does not call `core.mojo` and does not bypass model calls using the standalone gate package. Metrics approximate each tool definition as 120 tokens and use an illustrative $3/million-token rate. They are not tokenizer measurements, actual bills, or guaranteed savings. No universal speedup is claimed.

## Verify

```sh
go test -race ./...
go vet ./...
go build .
```

[Release verification](RELEASE_VERIFICATION.md). MIT licensed. Built by Buzburg.
