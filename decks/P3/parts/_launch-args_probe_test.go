package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"morphstudio/internal/testhelp"
)

var pLAFirstTen = []string{"-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose", "--include-hook-events", "--permission-prompt-tool", "stdio", "--dangerously-skip-permissions"}

func pLAWith(rest ...string) []string {
	return append(append([]string{}, pLAFirstTen...), rest...)
}

func pLAMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Errorf("stat %s: %v", path, err)
		return 0
	}
	return st.Mode() & 0o777
}

func TestProbeLaunchArgsExample1(t *testing.T) {
	got := Args(Launch{Bin: "claude", Dir: "/home/morph/projects/demo", SessionID: "11111111-2222-3333-4444-555555555555", BudgetUSD: 30, MCPConfigPath: "/tmp/demo-mcp.json"})
	testhelp.Equal(t, "example 1 Args", got, pLAWith("--session-id", "11111111-2222-3333-4444-555555555555", "--max-budget-usd", "30", "--mcp-config", "/tmp/demo-mcp.json"))
	// variant: other values, Bin and Dir never in the argv
	got = Args(Launch{Bin: "/opt/claude", Dir: "/srv/p", SessionID: "abc", BudgetUSD: 0.05, MCPConfigPath: "/s/q-mcp.json"})
	testhelp.Equal(t, "example 1 Args (budget 0.05)", got, pLAWith("--session-id", "abc", "--max-budget-usd", "0.05", "--mcp-config", "/s/q-mcp.json"))
}

func TestProbeLaunchArgsExample2(t *testing.T) {
	got := Args(Launch{SessionID: "s-2", Resume: true, BudgetUSD: 12.5, Model: "opus", MCPConfigPath: "/x/p-mcp.json", Extra: []string{"--effort", "high"}})
	testhelp.Equal(t, "example 2 Args", got, pLAWith("--resume", "s-2", "--max-budget-usd", "12.5", "--model", "opus", "--mcp-config", "/x/p-mcp.json", "--effort", "high"))
	// variant: resume with a model only; a budget of 100.25
	got = Args(Launch{SessionID: "r-9", Resume: true, Model: "sonnet"})
	testhelp.Equal(t, "example 2 Args (resume, model only)", got, pLAWith("--resume", "r-9", "--model", "sonnet"))
	got = Args(Launch{SessionID: "b", BudgetUSD: 100.25, Extra: []string{"--x"}})
	testhelp.Equal(t, "example 2 Args (budget 100.25, extra)", got, pLAWith("--session-id", "b", "--max-budget-usd", "100.25", "--x"))
}

func TestProbeLaunchArgsExample3(t *testing.T) {
	got := Args(Launch{SessionID: "s-3"})
	testhelp.Equal(t, "example 3 Args", got, pLAWith("--session-id", "s-3"))
	testhelp.Equal(t, "example 3 length", len(got), 12)
	// variant: a negative budget is not passed
	testhelp.Equal(t, "example 3 Args (negative budget)", Args(Launch{SessionID: "n", BudgetUSD: -1}), pLAWith("--session-id", "n"))
}

func TestProbeLaunchArgsExample4(t *testing.T) {
	want := []string{"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1"}
	testhelp.Equal(t, "example 4 Env (launch of example 1)", Env(Launch{Bin: "claude", Dir: "/home/morph/projects/demo", SessionID: "11111111-2222-3333-4444-555555555555", BudgetUSD: 30, MCPConfigPath: "/tmp/demo-mcp.json"}), want)
	testhelp.Equal(t, "example 4 Env (zero Launch)", Env(Launch{}), want)
	testhelp.Equal(t, "example 4 Env (extra)", Env(Launch{SessionID: "s-2", Resume: true, BudgetUSD: 12.5, Model: "opus", MCPConfigPath: "/x/p-mcp.json", Extra: []string{"--effort", "high"}}), want)
	// variant: a caller appending to one result does not change the next
	e := Env(Launch{})
	if len(e) > 0 {
		e[0] = "CHANGED=1"
	}
	testhelp.Equal(t, "example 4 Env (a fresh slice each call)", Env(Launch{}), want)
}

func TestProbeLaunchArgsExample5(t *testing.T) {
	got := string(MCPConfigJSON("http://127.0.0.1:7080/mcp/demo/session", "tok-1"))
	testhelp.Equal(t, "example 5 MCPConfigJSON", got, `{"mcpServers":{"morphd":{"type":"http","url":"http://127.0.0.1:7080/mcp/demo/session","headers":{"Authorization":"Bearer tok-1"}}}}`+"\n")
	got = string(MCPConfigJSON("http://127.0.0.1:9/mcp/x/session", "t-77"))
	testhelp.Equal(t, "example 5 MCPConfigJSON (other url and token)", got, `{"mcpServers":{"morphd":{"type":"http","url":"http://127.0.0.1:9/mcp/x/session","headers":{"Authorization":"Bearer t-77"}}}}`+"\n")
}

func TestProbeLaunchArgsExample6(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "mcp")
	path, err := WriteMCPConfig(dir, "demo", "http://127.0.0.1:7181/mcp/demo/session", "tok-2")
	testhelp.Equal(t, "example 6 path", path, filepath.Join(dir, "demo-mcp.json"))
	testhelp.Equal(t, "example 6 error", err, error(nil))
	b, _ := os.ReadFile(filepath.Join(dir, "demo-mcp.json"))
	testhelp.Equal(t, "example 6 file content", string(b), `{"mcpServers":{"morphd":{"type":"http","url":"http://127.0.0.1:7181/mcp/demo/session","headers":{"Authorization":"Bearer tok-2"}}}}`+"\n")
	testhelp.Equal(t, "example 6 file mode", pLAMode(t, filepath.Join(dir, "demo-mcp.json")), os.FileMode(0o600))
	testhelp.Equal(t, "example 6 dir mode", pLAMode(t, dir), os.FileMode(0o700))
	// variant: another project in an existing dir; an existing 0644 file is rewritten and left 0600
	dir2 := t.TempDir()
	old := filepath.Join(dir2, "p-7-mcp.json")
	if err := os.WriteFile(old, []byte(strings.Repeat("x", 500)), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err = WriteMCPConfig(dir2, "p-7", "http://h/mcp/p-7/session", "tok-3")
	testhelp.Equal(t, "example 6 path (project p-7)", path, old)
	testhelp.Equal(t, "example 6 error (project p-7)", err, error(nil))
	b, _ = os.ReadFile(old)
	testhelp.Equal(t, "example 6 rewritten content", string(b), `{"mcpServers":{"morphd":{"type":"http","url":"http://h/mcp/p-7/session","headers":{"Authorization":"Bearer tok-3"}}}}`+"\n")
	testhelp.Equal(t, "example 6 existing file left 0600", pLAMode(t, old), os.FileMode(0o600))
}

func TestProbeLaunchArgsExample7(t *testing.T) {
	dir := testhelp.WriteFile(t, "f", "x")
	path, err := WriteMCPConfig(dir, "demo", "http://h", "tok")
	testhelp.Equal(t, "example 7 path", path, "")
	if err == nil {
		t.Errorf("example 7 error: nil, want one beginning %q", "claude: mcp config: ")
	} else if !strings.HasPrefix(err.Error(), "claude: mcp config: ") {
		t.Errorf("example 7 error: %q does not begin %q", err.Error(), "claude: mcp config: ")
	}
}
