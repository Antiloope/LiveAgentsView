// Package pilotmcp implements "lav pilot-mcp", a tiny stdio MCP server that
// Claude Code spawns as the target of --permission-prompt-tool. Live-tested
// against the real `claude` CLI (2.1.258): headless stream-json print mode
// never sends a permission decision over its main control channel — the only
// way a driver gets asked at all is by registering one MCP tool named
// approval_prompt via --mcp-config and pointing --permission-prompt-tool at
// it (mcp__<serverName>__approval_prompt). This process exists only to be
// that tool: each call it receives is relayed, one connection per call, to
// whichever pilot-runner (internal/pilotrunner) owns the real child process,
// over the same Unix domain control socket the daemon itself dials
// (internal/pilotwire) — so the actual approve/deny decision still comes from
// wherever the daemon is (or, if nothing is attached, a fail-closed deny; see
// runner.go's handling of an unrelayable permission_request).
package pilotmcp

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/pilotwire"
)

// Run is cmd/lav's entry point for `lav pilot-mcp <flags>`.
func Run(args []string) error {
	fs := flag.NewFlagSet("pilot-mcp", flag.ExitOnError)
	sock := fs.String("sock", "", "the session's pilot-runner control socket")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sock == "" {
		return fmt.Errorf("pilot-mcp: --sock is required")
	}
	return serve(*sock, os.Stdin, os.Stdout)
}

type rpcRequest struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var approvalPromptTool = map[string]any{
	"name":        "approval_prompt",
	"description": "Ask LiveAgentsView's dashboard whether a tool call may proceed",
	"inputSchema": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tool_name":   map[string]any{"type": "string"},
			"input":       map[string]any{"type": "object"},
			"tool_use_id": map[string]any{"type": "string"},
		},
		"required": []any{"tool_name", "input"},
	},
}

// serve reads one JSON-RPC 2.0 request per line from in and writes one
// response per line to out — the newline-delimited framing Claude Code's own
// stdio MCP transport uses (confirmed live). Each request is handled in its
// own goroutine so a slow approval_prompt (waiting on a human) never blocks a
// concurrent one for a different tool call in the same turn; out is shared,
// so writes to it are serialized.
func serve(sockPath string, in io.Reader, out io.Writer) error {
	var writeMu sync.Mutex
	sc := pilotwire.NewScanner(in)
	for sc.Scan() {
		var req rpcRequest
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			continue
		}
		go func(req rpcRequest) {
			resp, ok := handle(sockPath, req)
			if !ok {
				return
			}
			writeMu.Lock()
			defer writeMu.Unlock()
			_ = pilotwire.Encode(out, resp)
		}(req)
	}
	return sc.Err()
}

// handle answers one JSON-RPC request. The second return value is false for
// a notification (no id in the request), which per the spec gets no reply at
// all.
func handle(sockPath string, req rpcRequest) (rpcResponse, bool) {
	if req.ID == nil {
		return rpcResponse{}, false
	}
	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &params)
		version := params.ProtocolVersion
		if version == "" {
			version = "2025-06-18"
		}
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "lav-pilot-mcp", "version": "1"},
		}}, true
	case "tools/list":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"tools": []any{approvalPromptTool},
		}}, true
	case "tools/call":
		return handleToolCall(sockPath, req)
	default:
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "method not found"}}, true
	}
}

func handleToolCall(sockPath string, req rpcRequest) (rpcResponse, bool) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil || params.Name != "approval_prompt" {
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "unknown tool"}}, true
	}
	var args struct {
		ToolName  string          `json:"tool_name"`
		Input     json.RawMessage `json:"input"`
		ToolUseID string          `json:"tool_use_id"`
	}
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid arguments"}}, true
	}

	approve := askRunner(sockPath, args.ToolUseID, args.ToolName, args.Input)
	decision := map[string]any{"behavior": "deny", "message": "denied from LiveAgentsView"}
	if approve {
		decision = map[string]any{"behavior": "allow", "updatedInput": json.RawMessage(args.Input)}
	}
	text, _ := json.Marshal(decision)
	return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
		"content": []any{map[string]any{"type": "text", "text": string(text)}},
	}}, true
}

// askRunner relays one permission decision request to the pilot-runner
// holding sockPath and blocks for its answer. Any failure to reach it or get
// a well-formed answer back (no daemon attached to ask, the runner is gone,
// a malformed reply) fails closed — deny — rather than hanging the child's
// turn forever or silently allowing a tool call nobody actually approved.
func askRunner(sockPath, requestID, toolName string, input json.RawMessage) bool {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return false
	}
	defer conn.Close()
	if err := pilotwire.Encode(conn, pilotwire.ClientMsg{
		Op: "permission_request", RequestID: requestID, ToolName: toolName, Input: input,
	}); err != nil {
		return false
	}
	sc := pilotwire.NewScanner(conn)
	if !sc.Scan() {
		return false
	}
	var msg pilotwire.ServerMsg
	if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
		return false
	}
	return msg.Approve
}
