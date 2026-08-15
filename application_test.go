package kratoscore

import (
	"net"
	"testing"

	"github.com/go-kratos/kratos/v3"
	"github.com/liujitcn/kratos-core/internal/mcp"
	"github.com/liujitcn/kratos-core/internal/sse"
	"github.com/liujitcn/kratos-kit/bootstrap"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	sseserver "github.com/liujitcn/kratos-kit/transport/sse"
)

func TestNewServersSkipsInProcessMCP(t *testing.T) {
	inProcessServer := mcpserver.NewServer()
	standaloneServer := mcpserver.NewServer()
	servers := newServers(nil, nil, &mcp.Server{Server: inProcessServer, InProcess: true}, nil, nil, nil)
	if len(servers) != 0 {
		t.Fatalf("in-process MCP should not be added as an independent server, got %d", len(servers))
	}
	standaloneMCP := &mcp.Server{Server: standaloneServer}
	servers = newServers(nil, nil, standaloneMCP, nil, nil, nil)
	if len(servers) != 1 || servers[0] != standaloneMCP {
		t.Fatalf("standalone MCP should be added as an independent server")
	}
}

func TestTransportWrappersExposeEndpoints(t *testing.T) {
	mcpWrapper := &mcp.Server{Server: mcpserver.NewServer(mcpserver.WithServerType(mcpserver.ServerTypeStdio))}
	mcpEndpoint, err := mcpWrapper.Endpoint()
	if err != nil {
		t.Fatalf("MCP endpoint should be available: %v", err)
	}
	if mcpEndpoint == nil || mcpEndpoint.Scheme != mcpserver.KindMCP {
		t.Fatalf("unexpected MCP endpoint: %v", mcpEndpoint)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("create SSE listener: %v", err)
	}
	defer listener.Close()
	var lowerSSE *sseserver.Server
	lowerSSE, err = sseserver.NewServer(sseserver.WithListener(listener))
	if err != nil {
		t.Fatalf("create SSE server: %v", err)
	}
	sseWrapper := &sse.Server{Server: lowerSSE}
	sseEndpoint, err := sseWrapper.Endpoint()
	if err != nil {
		t.Fatalf("SSE endpoint should be available: %v", err)
	}
	if sseEndpoint == nil || sseEndpoint.Scheme != sseserver.KindSSE {
		t.Fatalf("unexpected SSE endpoint: %v", sseEndpoint)
	}
}

func TestProviderSetUsesSingleModuleProvider(t *testing.T) {
	var provider func(*bootstrap.Context, Module) (*kratos.App, func(), error) = NewApplicationWithModule
	if provider == nil {
		t.Fatal("single-module application provider is nil")
	}
}
