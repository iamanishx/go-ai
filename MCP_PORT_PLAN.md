# MCP Port Plan (Go)

## Goal

Add a Go equivalent of `createMCPClient` from `@ai-sdk/mcp`, modeled against:

- MCP TypeScript SDK: `https://github.com/modelcontextprotocol/typescript-sdk`
- MCP Go SDK: `https://github.com/modelcontextprotocol/go-sdk`

Implementation should prefer wrapping official SDK client/transports over custom protocol code, and integrate cleanly with this repo's existing `agent` + `provider.Tool` model.

No implementation in this step. This file is the execution plan.

Status: initial implementation started on branch `feat/mcp-module-split`.

---

## Reference alignment scope

Two references drive this port:

1. `@ai-sdk/mcp` behavior and ergonomics (`createMCPClient`, tool conversion, lifecycle)
2. Official MCP SDK primitives (TS SDK patterns -> Go SDK primitives)

This means we copy the product behavior from `@ai-sdk/mcp`, but we implement via official MCP Go SDK constructs.

---

## What `@ai-sdk/mcp` does today (reference behavior)

Based on `packages/mcp/src/tool/*` in the `@ai` repo:

1. Provides `createMCPClient(config)` that:
   - Creates transport (SSE, HTTP streamable, or custom transport object)
   - Performs MCP `initialize` handshake
   - Tracks server capabilities
   - Exposes higher-level API methods

2. Exposes client methods:
   - `tools()` -> fetch server tools and convert to AI SDK tools
   - `toolsFromDefinitions()` -> convert already fetched definitions
   - `listTools()`, `listResources()`, `readResource()`, `listResourceTemplates()`
   - experimental prompt methods
   - `onElicitationRequest(...)`
   - `close()`

3. Tool adapter behavior:
   - Converts MCP tool schema -> executable SDK tool
   - On execute, sends `tools/call`
   - Maps MCP result content into model-usable output

4. Transport behavior:
   - `stdio` for local process
   - `sse` transport
   - `http` streamable transport (POST + optional SSE inbound)
   - Uniform callbacks: `onmessage`, `onerror`, `onclose`

5. Reliability behavior:
   - Request id tracking and response handler map
   - Capability checks before requests
   - Typed parsing/validation
   - Proper cleanup on close

---

## Go API target (minimal, clean)

### Package

Create a new package:

- `mcp/`

### Core entry point

- `func CreateMCPClient(ctx context.Context, cfg MCPClientConfig) (*Client, error)`

### Config

- `Transport` (official go-sdk transport instance) OR transport config for convenience wrappers
- `Name` (default `go-ai-mcp-client`)
- `Version` (default `1.0.0`)
- `OnUncaughtError func(error)`
- `Capabilities` (optional, for elicitation later)

### Client methods (phase 1)

- `Tools(ctx context.Context) (map[string]provider.Tool, error)`
- `ListTools(ctx context.Context, cursor string) (..., error)`
- `Close() error`

### Client methods (phase 2)

- `ListResources`, `ReadResource`, `ListResourceTemplates`
- Prompt methods
- Elicitation registration hook

---

## Transport design (Go SDK first)

Primary rule: use official go-sdk transport/client APIs directly where available.

No custom JSON-RPC transport protocol implementation unless go-sdk lacks a required capability.

Planned concrete transports (through go-sdk):

1. `stdio` transport
   - Use go-sdk stdio transport/client components
   - Avoid custom newline framing unless absolutely necessary

2. `sse` transport
   - Use go-sdk SSE transport/client components
   - Reuse SDK event handling/resume semantics if provided

3. `http` streamable transport
   - Use go-sdk streamable HTTP transport/client components
   - Keep session management aligned with go-sdk capabilities

If a transport is not available in go-sdk yet, we either:

1. mark it out-of-scope for MVP, or
2. add a minimal compatibility wrapper with strict TODO to replace with official SDK transport.

---

## Tool conversion into `provider.Tool`

### Mapping

- MCP tool `name` -> `provider.Tool.Name`
- MCP tool `description` -> `provider.Tool.Description`
- MCP input JSON schema -> `provider.Tool.Parameters`
- `provider.Tool.Execute` -> calls MCP `tools/call`

### Result shaping

Initial minimal behavior:

- Return stringified text-first content when MCP returns content blocks
- If structured content exists, return JSON string

Future enhancement:

- Richer typed conversion (text/image/content list) similar to TS `mcpToModelOutput`

---

## Integration with current Go agent

No agent API changes required for phase 1.

Usage target:

1. Create MCP client
2. Fetch tools: `mcpTools := client.Tools(ctx)`
3. Merge with local tools
4. Pass merged list into `agent.CreateToolLoopAgent(...)`

---

## Suggested implementation phases

### Phase 1 (MVP)

- `mcp` package skeleton
- `CreateMCPClient`
- stdio transport via go-sdk
- `ListTools` + `Tools` conversion to `provider.Tool`
- close/cleanup
- example using local filesystem MCP server

### Phase 2

- SSE transport via go-sdk
- HTTP streamable transport via go-sdk
- resources support
- improved error typing

### Phase 3

- prompts support
- elicitation callback support
- stronger schema output handling
- retry/backoff knobs

---

## Error model

Add typed errors in Go package, e.g.:

- `ErrClientClosed`
- `ErrUnsupportedCapability`
- `ErrProtocolVersion`
- `ErrTransport`
- `ErrInvalidServerResponse`

All public methods should return wrapped errors with operation context.

---

## Test plan

1. Unit tests
   - request/response id routing
   - tool conversion correctness
   - close behavior rejects pending requests

2. Transport tests
   - stdio message framing
   - SSE parse loop and reconnect behavior (if implemented)
   - HTTP transport response handling

3. Integration tests
   - run mock/local MCP server
   - fetch tools and execute through `provider.Tool.Execute`
   - agent loop with MCP tool end-to-end

---

## Docs and examples to add once coding starts

1. `README.md`
   - new MCP section
   - minimal stdio example

2. `docs/src/content/docs/`
   - new `mcp/` docs page
   - transport comparison table (stdio vs sse vs http)

3. `examples/`
   - `examples/mcp-stdio/main.go`
   - `examples/mcp-http/main.go` (after phase 2)

---

## Open decisions before implementation

1. Should `mcp` be inside main module now, or split as `go-ai-mcp` later?
2. Do we keep only minimal methods (`Tools`, `ListTools`, `Close`) for first release?
3. For tool results, do we standardize on text-only output initially?
4. Which exact go-sdk version should we pin for transport stability?

Recommended defaults:

- Keep inside this repo first for speed
- Ship MVP with stdio + `Tools`
- Add sse/http next
- Keep output mapping minimal and predictable first
- Pin go-sdk version and avoid floating upgrades in MVP
