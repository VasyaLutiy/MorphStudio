//go:build tools

// Package deps pins the module's declared dependencies so `go mod tidy` and `go mod vendor` keep them before any
// package of the record imports them.
package deps

import (
	_ "github.com/modelcontextprotocol/go-sdk/mcp"
)
