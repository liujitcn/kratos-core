package kratoscore

import (
	"testing"

	"github.com/liujitcn/kratos-core/internal/server"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
)

func TestNewServersSkipsInProcessMCP(t *testing.T) {
	inProcessServer := mcpserver.NewServer()
	standaloneServer := mcpserver.NewServer()
	servers := newServers(nil, nil, &server.MCPRuntime{Server: inProcessServer, InProcess: true}, nil, nil, nil)
	if len(servers) != 0 {
		t.Fatalf("in-process MCP should not be added as an independent server, got %d", len(servers))
	}
	servers = newServers(nil, nil, &server.MCPRuntime{Server: standaloneServer}, nil, nil, nil)
	if len(servers) != 1 || servers[0] != standaloneServer {
		t.Fatalf("standalone MCP should be added as an independent server")
	}
}
