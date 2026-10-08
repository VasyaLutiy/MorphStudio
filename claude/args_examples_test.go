package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"morphstudio/internal/testhelp"
)

func laFirstTen() []string {
	return []string{
		"-p",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--include-hook-events",
		"--permission-prompt-tool", "stdio",
		"--dangerously-skip-permissions",
	}
}

func TestLaunchArgsExample1(t *testing.T) {
	l := Launch{
		Bin:           "claude",
		Dir:           "/home/morph/projects/demo",
		SessionID:     "11111111-2222-3333-4444-555555555555",
		BudgetUSD:     30,
		MCPConfigPath: "/tmp/demo-mcp.json",
	}
	want := append(laFirstTen(),
		"--session-id", "11111111-2222-3333-4444-555555555555",
		"--max-budget-usd", "30",
		"--mcp-config", "/tmp/demo-mcp.json",
	)
	testhelp.Equal(t, "Args(example 1)", Args(l), want)
}

func TestLaunchArgsExample2(t *testing.T) {
	l := Launch{
		SessionID:     "s-2",
		Resume:        true,
		BudgetUSD:     12.5,
		Model:         "opus",
		MCPConfigPath: "/x/p-mcp.json",
		Extra:         []string{"--effort", "high"},
	}
	want := append(laFirstTen(),
		"--resume", "s-2",
		"--max-budget-usd", "12.5",
		"--model", "opus",
		"--mcp-config", "/x/p-mcp.json",
		"--effort", "high",
	)
	testhelp.Equal(t, "Args(example 2)", Args(l), want)
}

func TestLaunchArgsExample3(t *testing.T) {
	l := Launch{SessionID: "s-3"}
	want := append(laFirstTen(), "--session-id", "s-3")
	testhelp.Equal(t, "Args(example 3)", Args(l), want)
}

func TestLaunchArgsExample4(t *testing.T) {
	l1 := Launch{
		Bin:           "claude",
		Dir:           "/home/morph/projects/demo",
		SessionID:     "11111111-2222-3333-4444-555555555555",
		BudgetUSD:     30,
		MCPConfigPath: "/tmp/demo-mcp.json",
	}
	l2 := Launch{}
	l3 := Launch{
		SessionID:     "s-2",
		Resume:        true,
		BudgetUSD:     12.5,
		Model:         "opus",
		MCPConfigPath: "/x/p-mcp.json",
		Extra:         []string{"--effort", "high"},
	}
	want := []string{"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1"}
	testhelp.Equal(t, "Env(example 1 launch)", Env(l1), want)
	testhelp.Equal(t, "Env(zero launch)", Env(l2), want)
	testhelp.Equal(t, "Env(example 2 launch)", Env(l3), want)
}

func TestLaunchArgsExample5(t *testing.T) {
	url := "http://127.0.0.1:7080/mcp/demo/session"
	token := "tok-1"
	got := MCPConfigJSON(url, token)
	want := `{"mcpServers":{"morphd":{"type":"http","url":"http://127.0.0.1:7080/mcp/demo/session","headers":{"Authorization":"Bearer tok-1"}}}}` + "\n"
	testhelp.Equal(t, "MCPConfigJSON(example 5)", string(got), want)
}

func TestLaunchArgsExample6(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "mcp")
	project := "demo"
	url := "http://127.0.0.1:7181/mcp/demo/session"
	token := "tok-2"

	path, err := WriteMCPConfig(dir, project, url, token)
	testhelp.Equal(t, "WriteMCPConfig path", path, filepath.Join(dir, "demo-mcp.json"))
	testhelp.Equal(t, "WriteMCPConfig error", err, nil)
	if err != nil {
		return
	}

	gotFile, err := os.ReadFile(path)
	testhelp.Equal(t, "ReadFile error", err, nil)
	if err != nil {
		return
	}
	wantFile := MCPConfigJSON(url, token)
	testhelp.Equal(t, "file contents", gotFile, wantFile)

	fileInfo, err := os.Stat(path)
	testhelp.Equal(t, "Stat file error", err, nil)
	if err != nil {
		return
	}
	testhelp.Equal(t, "file mode", fileInfo.Mode()&0o777, os.FileMode(0o600))

	dirInfo, err := os.Stat(dir)
	testhelp.Equal(t, "Stat dir error", err, nil)
	if err != nil {
		return
	}
	testhelp.Equal(t, "dir mode", dirInfo.Mode()&0o777, os.FileMode(0o700))
}

func TestLaunchArgsExample7(t *testing.T) {
	dir := testhelp.WriteFile(t, "f", "x")
	project := "demo"
	url := "http://127.0.0.1:7281/mcp/demo/session"
	token := "tok-3"

	path, err := WriteMCPConfig(dir, project, url, token)
	testhelp.Equal(t, "WriteMCPConfig path", path, "")
	testhelp.Equal(t, "WriteMCPConfig error non-nil", err == nil, false)
	if err == nil {
		return
	}
	testhelp.Equal(t, "error prefix", strings.HasPrefix(err.Error(), "claude: mcp config: "), true)
}
