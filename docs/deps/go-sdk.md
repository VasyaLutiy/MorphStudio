# go-sdk v1.8.0 — the digest for mcpserver

v1.8.0, vendored (Go 1.25). Signatures copied from `vendor/github.com/modelcontextprotocol/go-sdk/mcp/*.go`; every
JSON line measured on v1.8.0 at P0 (08.10). Build: `GOFLAGS=-mod=vendor GOPROXY=off`.

## Import and server

```go
import "github.com/modelcontextprotocol/go-sdk/mcp"

type Implementation struct { Name string `json:"name"`; Version string `json:"version"`; … }
func NewServer(impl *Implementation, options *ServerOptions) *Server  // options may be nil
s := mcp.NewServer(&mcp.Implementation{Name: "morphd", Version: "0.1.0"}, nil)
```

## Tools

```go
func AddTool[In, Out any](s *Server, t *Tool, h ToolHandlerFor[In, Out])       // panics on a bad schema
type ToolHandlerFor[In, Out any] func(_ context.Context, request *CallToolRequest, input In) (result *CallToolResult, output Out, _ error)
type CallToolRequest = ServerRequest[*CallToolParamsRaw]  // .Session, .Params, .Extra
type CallToolResult struct { Content []Content `json:"content"`; StructuredContent any; IsError bool; … }
type TextContent struct { Text string; Meta Meta; Annotations *Annotations }   // Content is an interface: &TextContent{…}
```

- The input schema is inferred from `In` (a struct or a map): the `json` tag names the property, the `jsonschema` tag
  is its description; a field without `omitempty` is `required`; `additionalProperties: false`.
- **Empty input**: `In` = `struct{}` (a named `type Empty struct{}`): schema `{"type":"object","additionalProperties":false}`;
  `"arguments":{}` and a missing `arguments` both work.
- `Out` a struct → `outputSchema` inferred, `structuredContent` = the value, and when `Content` is nil the SDK fills it
  with one `TextContent` holding the JSON of `Out` (keys sorted). Return `nil` as the result to get that.
- `Out` = `any` → no output schema; return `&CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "…"}}}`.
- A handler `error` → `isError: true`, its text in `content[0].text` (not a JSON-RPC error); so is input failing the schema.

```go
mcp.AddTool(s, &mcp.Tool{Name: "status", Description: "the project state"},
	func(ctx context.Context, req *mcp.CallToolRequest, in Empty) (*mcp.CallToolResult, Status, error) {
		return nil, Status{State: "busy", Phase: "P18"}, nil
	})
```

## HTTP (Streamable HTTP)

```go
func NewStreamableHTTPHandler(getServer func(*http.Request) *Server, opts *StreamableHTTPOptions) *StreamableHTTPHandler
type StreamableHTTPOptions struct { Stateless bool; JSONResponse bool; … }
h := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server { return srv }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
```

Stateless: no `initialize` needed; POST only (GET, DELETE → 405, `Allow: POST`); needs `Content-Type:
application/json` (else 415) and `Accept: application/json, text/event-stream` (`application/json` alone → 400
`Accept must contain both 'application/json' and 'text/event-stream'`). Answer: `200 application/json`, one object.

Measured `tools/call`, request then response, byte for byte:

```
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"status","arguments":{}}}
{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"phase\":\"P18\",\"state\":\"busy\"}"}],"structuredContent":{"phase":"P18","state":"busy"}}}
```

With `Out` = `any` and own content: `{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"sent: go"}]}}`.
A missing required argument: `{"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"validating \"arguments\": validating root: required: missing properties: [\"text\"]"}],"isError":true}}`.
`tools/list` result: `ttlMs`, `cacheScope`, `tools`. Compare decoded fields, never whole bytes.

## In-memory client (for tests)

```go
func NewInMemoryTransports() (*InMemoryTransport, *InMemoryTransport)
func (s *Server) Connect(ctx context.Context, t Transport, opts *ServerSessionOptions) (*ServerSession, error)
func NewClient(impl *Implementation, options *ClientOptions) *Client
func (c *Client) Connect(ctx context.Context, t Transport, opts *ClientSessionOptions) (cs *ClientSession, err error)
func (cs *ClientSession) ListTools(ctx context.Context, params *ListToolsParams) (*ListToolsResult, error)   // params may be nil
func (cs *ClientSession) CallTool(ctx context.Context, params *CallToolParams) (*CallToolResult, error)
func (cs *ClientSession) Close() error
type CallToolParams struct { Name string; Arguments any; … }   // json "name", "arguments,omitempty"
```

```go
ct, st := mcp.NewInMemoryTransports()
ss, _ := srv.Connect(ctx, st, nil)
cs, _ := mcp.NewClient(&mcp.Implementation{Name: "pm", Version: "0"}, nil).Connect(ctx, ct, nil)
r, _ := cs.CallTool(ctx, &mcp.CallToolParams{Name: "status", Arguments: map[string]any{}})
// r.IsError false, r.Content[0].(*mcp.TextContent).Text == `{"phase":"P18","state":"busy"}`,
// r.StructuredContent == map[string]any{"phase":"P18","state":"busy"}
cs.Close(); ss.Wait()
```

`ListTools` returns the tools sorted by name.
