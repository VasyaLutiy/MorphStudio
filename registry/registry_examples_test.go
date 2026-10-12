package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/queue"
)

func rgOpen(t *testing.T, base string) *Registry {
	t.Helper()
	r, err := Open(base)
	if err != nil {
		t.Fatalf("Open(%q): %v", base, err)
	}
	if r == nil {
		t.Fatalf("Open(%q): nil registry", base)
	}
	return r
}

func rgZeta() Project {
	return Project{
		Name:      "zeta",
		Language:  "go",
		RepoURL:   "https://github.com/o/zeta",
		Dir:       "/p/zeta",
		Caps:      queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5},
		CreatedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC),
	}
}

func rgAlpha() Project {
	return Project{
		Name:      "alpha",
		Language:  "python",
		RepoURL:   "https://github.com/o/alpha",
		Dir:       "/p/alpha",
		Caps:      queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5},
		CreatedAt: time.Date(2026, 10, 8, 15, 1, 0, 0, time.UTC),
	}
}

func rgSeed(t *testing.T) (*Registry, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "state")
	r := rgOpen(t, base)
	if err := r.Add(rgZeta()); err != nil {
		t.Fatalf("Add zeta: %v", err)
	}
	if err := r.Add(rgAlpha()); err != nil {
		t.Fatalf("Add alpha: %v", err)
	}
	return r, base
}

func rgDecodeProjects(t *testing.T, path string) []Project {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return projects
}

func rgNames(ps []Project) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Name
	}
	return out
}

func rgMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Mode() & 0o777
}

func TestProjectRegistryExample1(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	r, err := Open(base)
	testhelp.Equal(t, "Open error", err, nil)
	if r == nil {
		t.Fatal("Open returned nil registry")
	}
	if _, statErr := os.Stat(base); statErr != nil {
		t.Fatalf("stat base: %v", statErr)
	}
	testhelp.Equal(t, "List", r.List(), []Project{})
	got, err := r.Get("demo")
	testhelp.Equal(t, "Get demo project", got, Project{})
	testhelp.Equal(t, "Get demo error", err, ErrNotFound)
	testhelp.Equal(t, "Base", r.Base(), base)

	testhelp.Equal(t, "ErrNotFound message", ErrNotFound.Error(), "project not found")
	testhelp.Equal(t, "ErrExists message", ErrExists.Error(), "project exists")
	testhelp.Equal(t, "ErrBadName message", ErrBadName.Error(), "bad project name")
	testhelp.Equal(t, "ErrNoState message", ErrNoState.Error(), "no state")
}

func TestProjectRegistryExample2(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	r := rgOpen(t, base)
	testhelp.Equal(t, "Add zeta", r.Add(rgZeta()), nil)
	testhelp.Equal(t, "Add alpha", r.Add(rgAlpha()), nil)
	testhelp.Equal(t, "List names", rgNames(r.List()), []string{"alpha", "zeta"})
	decoded := rgDecodeProjects(t, filepath.Join(base, "projects.json"))
	testhelp.Equal(t, "decoded projects", decoded, r.List())

	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("readdir %s: %v", base, err)
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	testhelp.Equal(t, "base entries", names, []string{"projects.json"})
}

func TestProjectRegistryExample3(t *testing.T) {
	r, _ := rgSeed(t)
	testhelp.Equal(t, "Add alpha again", r.Add(Project{Name: "alpha"}), ErrExists)
	testhelp.Equal(t, "Add Alpha", r.Add(Project{Name: "Alpha"}), ErrBadName)
	testhelp.Equal(t, "Add -x", r.Add(Project{Name: "-x"}), ErrBadName)
	testhelp.Equal(t, "Add 41 a", r.Add(Project{Name: strings.Repeat("a", 41)}), ErrBadName)
	testhelp.Equal(t, "List still 2", rgNames(r.List()), []string{"alpha", "zeta"})

	// The same shape admits other names, e.g. "beta" and "m-1".
	rr := rgOpen(t, filepath.Join(t.TempDir(), "state"))
	testhelp.Equal(t, "Add beta error", rr.Add(Project{Name: "beta"}), nil)
	testhelp.Equal(t, "Add m-1 error", rr.Add(Project{Name: "m-1"}), nil)
}

func TestProjectRegistryExample4(t *testing.T) {
	r, base := rgSeed(t)
	r2 := rgOpen(t, base)
	first := r.List()
	second := r2.List()
	testhelp.Equal(t, "first List not nil", first == nil, false)
	testhelp.Equal(t, "first List", first, []Project{rgAlpha(), rgZeta()})
	for i := range first {
		testhelp.Equal(t, "reopened project", second[i], first[i])
	}
	testhelp.Equal(t, "second List", second, first)

	g, err := r2.Get("zeta")
	testhelp.Equal(t, "Get zeta error", err, nil)
	testhelp.Equal(t, "Get zeta", g, rgZeta())
}

func TestProjectRegistryExample5(t *testing.T) {
	r, base := rgSeed(t)
	type S struct {
		N int `json:"n"`
	}
	var out S
	testhelp.Equal(t, "SaveState alpha error", r.SaveState("alpha", S{7}), nil)
	testhelp.Equal(t, "LoadState alpha error", r.LoadState("alpha", &out), nil)
	testhelp.Equal(t, "out", out, S{7})

	var out2 S
	testhelp.Equal(t, "LoadState zeta error", r.LoadState("zeta", &out2), ErrNoState)

	var out3 S
	testhelp.Equal(t, "SaveState nope error", r.SaveState("nope", S{1}), ErrNotFound)
	testhelp.Equal(t, "LoadState nope error", r.LoadState("nope", &out3), ErrNotFound)

	testhelp.Equal(t, "state.json mode", rgMode(t, filepath.Join(base, "alpha", "state.json")), os.FileMode(0o600))
	testhelp.Equal(t, "alpha dir mode", rgMode(t, filepath.Join(base, "alpha")), os.FileMode(0o700))
}

func TestProjectRegistryExample6(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "projects.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Open(base)
	testhelp.Equal(t, "Open registry nil", r == nil, true)
	if err == nil || !strings.HasPrefix(err.Error(), "registry: read ") {
		t.Fatalf("Open error %v: want prefix %q", err, "registry: read ")
	}
}

func TestProjectRegistryExample7(t *testing.T) {
	r := rgOpen(t, filepath.Join(t.TempDir(), "state"))
	if err := r.Add(Project{Name: "alpha", Language: "go", Dir: "/p/alpha"}); err != nil {
		t.Fatal(err)
	}
	p := r.List()
	p[0].Name = "changed"
	p = append(p, Project{Name: "extra"})
	g, _ := r.Get("alpha")
	g.Name = "changed"

	want := []Project{{Name: "alpha", Language: "go", Dir: "/p/alpha"}}
	testhelp.Equal(t, "List", r.List(), want)
	g2, err := r.Get("alpha")
	testhelp.Equal(t, "Get error", err, nil)
	testhelp.Equal(t, "Get", g2, Project{Name: "alpha", Language: "go", Dir: "/p/alpha"})
}

func TestProjectRegistryExample8(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	r := rgOpen(t, base)
	if err := r.Add(Project{Name: "zeta", Language: "go", Dir: "/p/zeta", User: "zeta-ux"}); err != nil {
		t.Fatal(err)
	}
	r2 := rgOpen(t, base)
	g, err := r2.Get("zeta")
	testhelp.Equal(t, "Get zeta error", err, nil)
	testhelp.Equal(t, "Get zeta", g, Project{Name: "zeta", Language: "go", Dir: "/p/zeta", User: "zeta-ux"})
	decoded := rgDecodeProjects(t, filepath.Join(base, "projects.json"))
	testhelp.Equal(t, "decoded projects", decoded, r2.List())

	if err := r2.Add(Project{Name: "plain", Language: "go", Dir: "/p/plain"}); err != nil {
		t.Fatal(err)
	}
	r3 := rgOpen(t, base)
	gp, err := r3.Get("plain")
	testhelp.Equal(t, "Get plain error", err, nil)
	testhelp.Equal(t, "Get plain user", gp.User, "")
	testhelp.Equal(t, "Get plain", gp, Project{Name: "plain", Language: "go", Dir: "/p/plain"})
	decoded2 := rgDecodeProjects(t, filepath.Join(base, "projects.json"))
	testhelp.Equal(t, "decoded projects again", decoded2, r3.List())
}
