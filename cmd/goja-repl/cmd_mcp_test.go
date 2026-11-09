package main

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/replmcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPHelpDocumentsSafeDefault(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"mcp", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute mcp help: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{"--http", "--allow-remote", "--persistent", "safe"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("mcp help missing %q:\n%s", want, rendered)
		}
	}
}

func TestMCPModuleMiddlewareDefaultsToSafe(t *testing.T) {
	available := []string{"crypto", "events", "exec", "fs", "os", "path", "time", "timer"}
	selected := func(opts *rootOptions) string {
		return strings.Join(mcpModuleMiddleware(opts)(func(a []string) []string { return a })(available), ",")
	}
	if got := selected(&rootOptions{}); strings.Contains(got, "fs") || strings.Contains(got, "exec") || !strings.Contains(got, "path") {
		t.Fatalf("default selection = %s", got)
	}
	if got := selected(&rootOptions{EnableModules: []string{"fs", "path"}}); got != "fs,path" {
		t.Fatalf("enabled selection = %s", got)
	}
	if got := selected(&rootOptions{EnableModules: []string{"fs"}, SafeMode: true}); strings.Contains(got, "fs") {
		t.Fatalf("safe mode must win over enable: %s", got)
	}
	if got := selected(&rootOptions{DisableModules: []string{"path"}}); strings.Contains(got, "path") || strings.Contains(got, "fs") {
		t.Fatalf("disable selection = %s", got)
	}
}

func connectMCPCommandServer(t *testing.T, opts *rootOptions, transport func(*mcp.Server) mcp.Transport) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	command := newMCPCommand(&bytes.Buffer{}, opts)
	server, app, store, err := command.newMCPServer(ctx, mcpSettings{})
	if err != nil {
		t.Fatalf("new mcp server: %v", err)
	}
	t.Cleanup(func() { _ = closeAppAndStore(app, store) })
	client := mcp.NewClient(&mcp.Implementation{Name: "goja-repl-mcp-test"}, nil)
	session, err := client.Connect(ctx, transport(server), nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func mcpEval(t *testing.T, session *mcp.ClientSession, code string) (string, bool) {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "eval", Arguments: map[string]any{"code": code}})
	if err != nil {
		t.Fatalf("eval %q: %v", code, err)
	}
	return result.Content[0].(*mcp.TextContent).Text, result.IsError
}

func TestMCPCommandServerIsSafeByDefault(t *testing.T) {
	session := connectMCPCommandServer(t, &rootOptions{}, func(server *mcp.Server) mcp.Transport {
		serverTransport, clientTransport := mcp.NewInMemoryTransports()
		if _, err := server.Connect(context.Background(), serverTransport, nil); err != nil {
			t.Fatalf("server connect: %v", err)
		}
		return clientTransport
	})
	if text, isErr := mcpEval(t, session, `require("path").join("a", "b")`); isErr || text != `"a/b"` {
		t.Fatalf("path: isError=%v text=%q", isErr, text)
	}
	if text, isErr := mcpEval(t, session, `require("fs")`); !isErr || !strings.Contains(text, "Invalid module") {
		t.Fatalf("fs should be unavailable: isError=%v text=%q", isErr, text)
	}
}

func TestMCPCommandServerOverStreamableHTTP(t *testing.T) {
	session := connectMCPCommandServer(t, &rootOptions{EnableModules: []string{"fs"}}, func(server *mcp.Server) mcp.Transport {
		ts := httptest.NewServer(replmcp.NewStreamableHTTPHandler(server, nil))
		t.Cleanup(ts.Close)
		return &mcp.StreamableClientTransport{Endpoint: ts.URL}
	})
	if text, isErr := mcpEval(t, session, `const n = 40; n + 2`); isErr || text != "42" {
		t.Fatalf("eval over http: isError=%v text=%q", isErr, text)
	}
	if text, isErr := mcpEval(t, session, `typeof require("fs").readFileSync`); isErr || text != `"function"` {
		t.Fatalf("explicitly enabled fs: isError=%v text=%q", isErr, text)
	}
}
