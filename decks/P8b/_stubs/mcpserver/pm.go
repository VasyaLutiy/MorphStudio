package mcpserver

import (
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"morphstudio/control"
)

const ServerName = "morphd"

const ServerVersion = "0.1.0"

type okView struct {
	OK bool `json:"ok"`
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	return nil, nil, errors.New("stub jsonResult")
}

func errorResult(err error) (*mcp.CallToolResult, any, error) {
	return nil, nil, errors.New("stub errorResult")
}

func UserServer(c control.Control) *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: "stub UserServer", Version: ServerVersion}, nil)
}

func PMServer(c control.Control, project string) *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: "stub PMServer", Version: ServerVersion}, nil)
}
