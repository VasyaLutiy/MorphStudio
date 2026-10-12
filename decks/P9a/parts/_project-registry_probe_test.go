package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/queue"
)

var pPRZeta = Project{Name: "zeta", Language: "go", RepoURL: "https://github.com/o/zeta", Dir: "/p/zeta", Caps: queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}, CreatedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}
var pPRAlpha = Project{Name: "alpha", Language: "python", RepoURL: "https://github.com/o/alpha", Dir: "/p/alpha", Caps: queue.Caps{ClaudeUSD: 12, Hours: 2, ExecutorUSD: 4}, CreatedAt: time.Date(2026, 10, 8, 15, 1, 0, 0, time.UTC)}

func pPROpen(t *testing.T, what, base string) *Registry {
	t.Helper()
	r, err := Open(base)
	if err != nil {
		t.Fatalf("%s: Open(%s) error %v, want nil", what, base, err)
	}
	if r == nil {
		t.Fatalf("%s: Open(%s) returned a nil *Registry", what, base)
	}
	return r
}

func pPRNames(ps []Project) []string {
	out := []string{}
	for _, p := range ps {
		out = append(out, p.Name)
	}
	return out
}

func pPRDir(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Errorf("read dir %s: %v", dir, err)
		return nil
	}
	out := []string{}
	for _, e := range es {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

func pPRMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Errorf("stat %s: %v", path, err)
		return 0
	}
	return st.Mode() & 0o777
}

// pPRTwo is the registry after example 2: zeta then alpha added.
func pPRTwo(t *testing.T, what string) (*Registry, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "state")
	r := pPROpen(t, what, base)
	if err := r.Add(pPRZeta); err != nil {
		t.Fatalf("%s: Add(zeta) error %v, want nil", what, err)
	}
	if err := r.Add(pPRAlpha); err != nil {
		t.Fatalf("%s: Add(alpha) error %v, want nil", what, err)
	}
	return r, base
}

func TestProbeProjectRegistryExample1(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	r := pPROpen(t, "example 1", base)
	st, err := os.Stat(base)
	testhelp.Equal(t, "example 1 base directory exists", err == nil && st.IsDir(), true)
	testhelp.Equal(t, "example 1 base directory mode", pPRMode(t, base), os.FileMode(0o700))
	l := r.List()
	testhelp.Equal(t, "example 1 List is non-nil", l != nil, true)
	testhelp.Equal(t, "example 1 List", l, []Project{})
	p, err := r.Get("demo")
	testhelp.Equal(t, "example 1 Get(demo) project", p, Project{})
	testhelp.Equal(t, "example 1 Get(demo) is ErrNotFound", errors.Is(err, ErrNotFound), true)
	testhelp.Equal(t, "example 1 Base", r.Base(), base)
	// variant: another base name, nested two levels deep
	base2 := filepath.Join(t.TempDir(), "x", "reg-7")
	r2 := pPROpen(t, "example 1 variant", base2)
	testhelp.Equal(t, "example 1 variant Base", r2.Base(), base2)
	testhelp.Equal(t, "example 1 sentinel texts", []string{ErrNotFound.Error(), ErrExists.Error(), ErrBadName.Error(), ErrNoState.Error()}, []string{"project not found", "project exists", "bad project name", "no state"})
}

func TestProbeProjectRegistryExample2(t *testing.T) {
	r, base := pPRTwo(t, "example 2")
	l := r.List()
	testhelp.Equal(t, "example 2 List names", pPRNames(l), []string{"alpha", "zeta"})
	testhelp.Equal(t, "example 2 List", l, []Project{pPRAlpha, pPRZeta})
	b, err := os.ReadFile(filepath.Join(base, "projects.json"))
	if err != nil {
		t.Fatalf("example 2: read projects.json: %v", err)
	}
	var file []Project
	testhelp.Equal(t, "example 2 projects.json decodes", json.Unmarshal(b, &file), error(nil))
	testhelp.Equal(t, "example 2 projects.json equals List", file, l)
	want, _ := json.MarshalIndent(l, "", "  ")
	testhelp.Equal(t, "example 2 projects.json is MarshalIndent with two spaces", strings.TrimRight(string(b), "\n"), string(want))
	testhelp.Equal(t, "example 2 base holds only projects.json", pPRDir(t, base), []string{"projects.json"})
	// variant: a third project sorts between them; the file follows
	m := Project{Name: "m-1", Language: "typescript", Dir: "/srv/m-1", Caps: queue.Caps{ClaudeUSD: 7.5, Hours: 1, ExecutorUSD: 2}, CreatedAt: time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)}
	testhelp.Equal(t, "example 2 variant Add(m-1)", r.Add(m), error(nil))
	testhelp.Equal(t, "example 2 variant List", r.List(), []Project{pPRAlpha, m, pPRZeta})
	b, _ = os.ReadFile(filepath.Join(base, "projects.json"))
	file = nil
	_ = json.Unmarshal(b, &file)
	testhelp.Equal(t, "example 2 variant projects.json", file, []Project{pPRAlpha, m, pPRZeta})
	testhelp.Equal(t, "example 2 variant base holds only projects.json", pPRDir(t, base), []string{"projects.json"})
}

func TestProbeProjectRegistryExample3(t *testing.T) {
	r, _ := pPRTwo(t, "example 3")
	testhelp.Equal(t, "example 3 Add(alpha) is ErrExists", errors.Is(r.Add(Project{Name: "alpha"}), ErrExists), true)
	for _, n := range []string{"Alpha", "-x", strings.Repeat("a", 41)} {
		testhelp.Equal(t, "example 3 Add("+n+") is ErrBadName", errors.Is(r.Add(Project{Name: n}), ErrBadName), true)
	}
	testhelp.Equal(t, "example 3 List still 2", pPRNames(r.List()), []string{"alpha", "zeta"})
	// variant: the name rule's edges on a fresh registry
	r2 := pPROpen(t, "example 3 variant", filepath.Join(t.TempDir(), "state"))
	for _, n := range []string{"", "a_b", "a.b", "-", "ab ", "zéta"} {
		testhelp.Equal(t, "example 3 variant Add("+n+") is ErrBadName", errors.Is(r2.Add(Project{Name: n}), ErrBadName), true)
	}
	for _, n := range []string{strings.Repeat("b", 40), "9-ok", "x"} {
		testhelp.Equal(t, "example 3 variant Add("+n+")", r2.Add(Project{Name: n}), error(nil))
	}
	testhelp.Equal(t, "example 3 variant List", pPRNames(r2.List()), []string{"9-ok", strings.Repeat("b", 40), "x"})
}

func TestProbeProjectRegistryExample4(t *testing.T) {
	r, base := pPRTwo(t, "example 4")
	r2 := pPROpen(t, "example 4 reopen", base)
	testhelp.Equal(t, "example 4 reopened List", r2.List(), r.List())
	g, err := r2.Get("zeta")
	testhelp.Equal(t, "example 4 Get(zeta)", g, pPRZeta)
	testhelp.Equal(t, "example 4 Get(zeta) error", err, error(nil))
	// variant: an Add after the reopen keeps the old entries in the file
	testhelp.Equal(t, "example 4 variant Add(beta)", r2.Add(Project{Name: "beta", Language: "go"}), error(nil))
	r3 := pPROpen(t, "example 4 second reopen", base)
	testhelp.Equal(t, "example 4 variant names after reopen", pPRNames(r3.List()), []string{"alpha", "beta", "zeta"})
}

type pPRState struct {
	N int `json:"n"`
}

func TestProbeProjectRegistryExample5(t *testing.T) {
	r, base := pPRTwo(t, "example 5")
	testhelp.Equal(t, "example 5 SaveState(alpha)", r.SaveState("alpha", pPRState{7}), error(nil))
	var out pPRState
	testhelp.Equal(t, "example 5 LoadState(alpha) error", r.LoadState("alpha", &out), error(nil))
	testhelp.Equal(t, "example 5 LoadState(alpha) value", out, pPRState{7})
	var out2 pPRState
	testhelp.Equal(t, "example 5 LoadState(zeta) is ErrNoState", errors.Is(r.LoadState("zeta", &out2), ErrNoState), true)
	testhelp.Equal(t, "example 5 SaveState(nope) is ErrNotFound", errors.Is(r.SaveState("nope", pPRState{1}), ErrNotFound), true)
	var out3 pPRState
	testhelp.Equal(t, "example 5 LoadState(nope) is ErrNotFound", errors.Is(r.LoadState("nope", &out3), ErrNotFound), true)
	testhelp.Equal(t, "example 5 state.json mode", pPRMode(t, filepath.Join(base, "alpha", "state.json")), os.FileMode(0o600))
	testhelp.Equal(t, "example 5 project dir mode", pPRMode(t, filepath.Join(base, "alpha")), os.FileMode(0o700))
	// variant: a second save replaces the first; no temporary file is left
	testhelp.Equal(t, "example 5 variant SaveState(alpha, 42)", r.SaveState("alpha", pPRState{42}), error(nil))
	out = pPRState{}
	_ = r.LoadState("alpha", &out)
	testhelp.Equal(t, "example 5 variant LoadState(alpha)", out, pPRState{42})
	testhelp.Equal(t, "example 5 variant base/alpha holds only state.json", pPRDir(t, filepath.Join(base, "alpha")), []string{"state.json"})
	testhelp.Equal(t, "example 5 variant zeta has no directory yet", pPRDir(t, base), []string{"alpha", "projects.json"})
}

func TestProbeProjectRegistryExample6(t *testing.T) {
	for _, text := range []string{"{not json", `{"name":"x"}`} {
		base := t.TempDir()
		if err := os.WriteFile(filepath.Join(base, "projects.json"), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		r, err := Open(base)
		testhelp.Equal(t, "example 6 Open("+text+") registry is nil", r == nil, true)
		if err == nil {
			t.Errorf("example 6 Open(%s) error: nil, want one beginning %q", text, "registry: read ")
		} else if !strings.HasPrefix(err.Error(), "registry: read ") {
			t.Errorf("example 6 Open(%s) error: %q does not begin %q", text, err.Error(), "registry: read ")
		}
	}
}

func TestProbeProjectRegistryExample7(t *testing.T) {
	r := pPROpen(t, "example 7", filepath.Join(t.TempDir(), "state"))
	want := Project{Name: "alpha", Language: "go", Dir: "/p/alpha"}
	testhelp.Equal(t, "example 7 Add(alpha)", r.Add(want), error(nil))
	p := r.List()
	if len(p) > 0 {
		p[0].Name = "changed"
	}
	p = append(p, Project{Name: "extra"})
	_ = p
	g, _ := r.Get("alpha")
	g.Name = "changed"
	testhelp.Equal(t, "example 7 List after writes to returned values", r.List(), []Project{want})
	g2, err := r.Get("alpha")
	testhelp.Equal(t, "example 7 Get(alpha) after writes", g2, want)
	testhelp.Equal(t, "example 7 Get(alpha) error", err, error(nil))
	// variant: the slice handed back by one List is not the next one's
	l1 := r.List()
	l2 := r.List()
	if len(l1) > 0 && len(l2) > 0 {
		l1[0].Dir = "/elsewhere"
		testhelp.Equal(t, "example 7 variant two Lists are separate slices", l2[0].Dir, "/p/alpha")
	}
}

// pPRUsers is the registry of example 8: alpha (ann), beta (bob) and core (the operator's, User "").
func pPRUsers(t *testing.T, what string) (*Registry, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "state")
	r := pPROpen(t, what, base)
	for _, p := range []Project{pPRUAlpha, pPRUBeta, pPRUCore} {
		if err := r.Add(p); err != nil {
			t.Fatalf("%s: Add(%s) error %v, want nil", what, p.Name, err)
		}
	}
	return r, base
}

var (
	pPRUAlpha = Project{Name: "alpha", Language: "go", RepoURL: "https://github.com/ann/alpha", Dir: "/home/mu-ann/projects/alpha", Caps: queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}, CreatedAt: time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC), User: "ann"}
	pPRUBeta  = Project{Name: "beta", Language: "python", RepoURL: "https://github.com/bob/beta", Dir: "/home/mu-bob/projects/beta", CreatedAt: time.Date(2026, 10, 12, 9, 5, 0, 0, time.UTC), User: "bob"}
	pPRUCore  = Project{Name: "core", Language: "go", Dir: "/home/morph/projects/core", CreatedAt: time.Date(2026, 10, 12, 9, 10, 0, 0, time.UTC)}
)

func TestProbeProjectRegistryExample8(t *testing.T) {
	r, _ := pPRUsers(t, "example 8")
	testhelp.Equal(t, "example 8 ListUser(ann)", r.ListUser("ann"), []Project{pPRUAlpha})
	testhelp.Equal(t, "example 8 ListUser(bob)", r.ListUser("bob"), []Project{pPRUBeta})
	testhelp.Equal(t, "example 8 ListUser(\"\")", r.ListUser(""), []Project{pPRUCore})
	eve := r.ListUser("eve")
	testhelp.Equal(t, "example 8 ListUser(eve) is non-nil", eve != nil, true)
	testhelp.Equal(t, "example 8 ListUser(eve)", eve, []Project{})
	testhelp.Equal(t, "example 8 List names", pPRNames(r.List()), []string{"alpha", "beta", "core"})
	l := r.ListUser("ann")
	if len(l) > 0 {
		l[0].User = "bob"
	}
	testhelp.Equal(t, "example 8 ListUser(bob) after a write to a returned value", r.ListUser("bob"), []Project{pPRUBeta})
}

func TestProbeProjectRegistryExample9(t *testing.T) {
	_, base := pPRUsers(t, "example 9")
	b, err := os.ReadFile(filepath.Join(base, "projects.json"))
	if err != nil {
		t.Fatalf("example 9: read projects.json: %v", err)
	}
	var raw []map[string]any
	testhelp.Equal(t, "example 9 projects.json decodes", json.Unmarshal(b, &raw), error(nil))
	if len(raw) != 3 {
		t.Fatalf("example 9: projects.json holds %d entries, want 3", len(raw))
	}
	testhelp.Equal(t, "example 9 alpha's user", raw[0]["user"], any("ann"))
	_, has := raw[2]["user"]
	testhelp.Equal(t, "example 9 core has no key user", has, false)
	r2 := pPROpen(t, "example 9 reopen", base)
	g, err := r2.Get("alpha")
	testhelp.Equal(t, "example 9 reopened alpha", g, pPRUAlpha)
	testhelp.Equal(t, "example 9 reopened alpha error", err, error(nil))
	old := t.TempDir()
	if err := os.WriteFile(filepath.Join(old, "projects.json"), []byte(`[{"name":"old","dir":"/p/old"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	r3 := pPROpen(t, "example 9 old file", old)
	g, err = r3.Get("old")
	testhelp.Equal(t, "example 9 a file without user", g, Project{Name: "old", Dir: "/p/old"})
	testhelp.Equal(t, "example 9 a file without user error", err, error(nil))
}
