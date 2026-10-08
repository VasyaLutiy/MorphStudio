// Package claude launches the claude CLI and describes the MCP configuration
// file handed to it.
package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Launch describes one claude session: the binary, its working directory, the
// session id, the budget, the model, the MCP config file and any extra argv
// entries passed verbatim.
type Launch struct {
	Bin           string
	Dir           string
	SessionID     string
	Resume        bool
	BudgetUSD     float64
	Model         string
	MCPConfigPath string
	Extra         []string
}

// Args returns the argv of the claude process for l. Bin and Dir are not part
// of the argv.
func Args(l Launch) []string {
	args := []string{
		"-p",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--include-hook-events",
		"--permission-prompt-tool", "stdio",
		"--dangerously-skip-permissions",
	}
	if l.Resume {
		args = append(args, "--resume", l.SessionID)
	} else {
		args = append(args, "--session-id", l.SessionID)
	}
	if l.BudgetUSD > 0 {
		args = append(args, "--max-budget-usd", strconv.FormatFloat(l.BudgetUSD, 'f', -1, 64))
	}
	if l.Model != "" {
		args = append(args, "--model", l.Model)
	}
	if l.MCPConfigPath != "" {
		args = append(args, "--mcp-config", l.MCPConfigPath)
	}
	args = append(args, l.Extra...)
	return args
}

// Env returns the environment entries added to the child process. Auto-memory
// is disabled in every phase session, whatever the Launch holds.
func Env(l Launch) []string {
	return []string{"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1"}
}

type mcpServer struct {
	Type    string            `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type mcpConfig struct {
	MCPServers map[string]mcpServer `json:"mcpServers"`
}

// MCPConfigJSON returns the MCP configuration granting claude access to the
// morphd HTTP server, carrying the bearer token for this session.
func MCPConfigJSON(url, token string) []byte {
	cfg := mcpConfig{
		MCPServers: map[string]mcpServer{
			"morphd": {
				Type: "http",
				URL:  url,
				Headers: map[string]string{
					"Authorization": "Bearer " + token,
				},
			},
		},
	}
	b, _ := json.Marshal(cfg)
	return append(b, '\n')
}

// WriteMCPConfig writes MCPConfigJSON(url, token) to <dir>/<project>-mcp.json
// and returns that path.
func WriteMCPConfig(dir, project, url, token string) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("claude: mcp config: %w", err)
	}
	path := filepath.Join(dir, project+"-mcp.json")
	if err := os.WriteFile(path, MCPConfigJSON(url, token), 0o600); err != nil {
		return "", fmt.Errorf("claude: mcp config: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return "", fmt.Errorf("claude: mcp config: %w", err)
	}
	return path, nil
}
