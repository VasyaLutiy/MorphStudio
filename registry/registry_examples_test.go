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

func TestProjectRegistryExample1(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	r, err := Open(base)
	testhelp.Equal(t, "Open error", err, nil)
	testhelp.Equal(t, "Open registry is nil", r == nil, false)
	if r == nil {
		t.Fatal("Open returned nil registry")
	}
	info, err := os.Stat(base)
	if err != nil {
		t.Fatalf("Stat %s: %v", base, err)
	}
	testhelp.Equal(t, "base is directory", info.IsDir(), true)
	testhelp.Equal(t, "List", r.List(), []Project{})
	got, err := r.Get("demo")
	testhelp.Equal(t, "Get demo project", got, Project{})
	testhelp.Equal(t, "Get demo error", err, ErrNotFound)
	testhelp.Equal(t, "Base", r.Base(), base)
}

func TestProjectRegistryExample2(t *testing.T) {
	r, base := rgNew(t)
	testhelp.Equal(t, "Add zeta error", r.Add(rgZeta()), nil)
	testhelp.Equal(t, "Add alpha error", r.Add(rgAlpha()), nil)

	list := r.List()
	gotNames := []string{}
	for _, p := range list {
		gotNames = append(gotNames, p.Name)
	}
	testhelp.Equal(t, "List names", gotNames, []string{"alpha", "zeta"})

	data, err := os.ReadFile(filepath.Join(base, "projects.json"))
	if err != nil {
		t.Fatalf("ReadFile projects.json: %v", err)
	}
	var decoded []Project
	testhelp.Equal(t, "decode error", json.Unmarshal(data, &decoded), nil)
	testhelp.Equal(t, "decoded projects", decoded, r.List())

	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", base, err)
	}
	gotEntries := []string{}
	for _, e := range entries {
		gotEntries = append(gotEntries, e.Name())
	}
	testhelp.Equal(t, "base entries", gotEntries, []string{"projects.json"})
}

func TestProjectRegistryExample3(t *testing.T) {
	r, _ := rgSeed(t)
	testhelp.Equal(t, "Add alpha again error", r.Add(Project{Name: "alpha"}), ErrExists)
	testhelp.Equal(t, "Add Alpha error", r.Add(Project{Name: "Alpha"}), ErrBadName)
	testhelp.Equal(t, "Add -x error", r.Add(Project{Name: "-x"}), ErrBadName)
	testhelp.Equal(t, "Add long name error", r.Add(Project{Name: strings.Repeat("a", 41)}), ErrBadName)
	testhelp.Equal(t, "List length", len(r.List()), 2)
}

func TestProjectRegistryExample4(t *testing.T) {
	r1, base := rgSeed(t)

	r2, err := Open(base)
	testhelp.Equal(t, "Open error", err, nil)
	if r2 == nil {
		t.Fatal("Open returned nil registry")
	}

	data, err := json.Marshal(r1.List())
	if err != nil {
		t.Fatalf("Marshal list: %v", err)
	}
	var want []Project
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatalf("Unmarshal list: %v", err)
	}
	testhelp.Equal(t, "List after reopen", r2.List(), want)

	got, err := r2.Get("zeta")
	testhelp.Equal(t, "Get zeta error", err, nil)
	testhelp.Equal(t, "Get zeta project", got, want[1])
}

func TestProjectRegistryExample5(t *testing.T) {
	r, base := rgSeed(t)

	type S struct {
		N int `json:"n"`
	}

	testhelp.Equal(t, "SaveState alpha error", r.SaveState("alpha", S{7}), nil)

	var out S
	testhelp.Equal(t, "LoadState alpha error", r.LoadState("alpha", &out), nil)
	testhelp.Equal(t, "loaded state", out, S{7})

	var out2 S
	testhelp.Equal(t, "LoadState zeta error", r.LoadState("zeta", &out2), ErrNoState)

	testhelp.Equal(t, "SaveState nope error", r.SaveState("nope", S{1}), ErrNotFound)

	var out3 S
	testhelp.Equal(t, "LoadState nope error", r.LoadState("nope", &out3), ErrNotFound)

	info, err := os.Stat(filepath.Join(base, "alpha", "state.json"))
	if err != nil {
		t.Fatalf("Stat state.json: %v", err)
	}
	testhelp.Equal(t, "state.json mode", info.Mode()&0o777, os.FileMode(0o600))

	dirInfo, err := os.Stat(filepath.Join(base, "alpha"))
	if err != nil {
		t.Fatalf("Stat alpha dir: %v", err)
	}
	testhelp.Equal(t, "alpha dir mode", dirInfo.Mode()&0o777, os.FileMode(0o700))
}

func TestProjectRegistryExample6(t *testing.T) {
	path := testhelp.WriteFile(t, "state/projects.json", "{not json")
	base := filepath.Dir(path)

	r, err := Open(base)
	testhelp.Equal(t, "Open registry is nil", r == nil, true)
	if err == nil {
		t.Fatal("Open error = nil, want prefix \"registry: read \"")
	}
	testhelp.Equal(t, "Open error prefix", strings.HasPrefix(err.Error(), "registry: read "), true)
}

func TestProjectRegistryExample7(t *testing.T) {
	r, _ := rgNew(t)
	rgAdd(t, r, Project{Name: "alpha", Language: "go", Dir: "/p/alpha"})

	p := r.List()
	p[0].Name = "changed"
	p = append(p, Project{Name: "extra"})
	testhelp.Equal(t, "returned list length", len(p), 2)

	g, err := r.Get("alpha")
	testhelp.Equal(t, "Get alpha error", err, nil)
	g.Name = "changed"

	testhelp.Equal(t, "List unchanged", r.List(), []Project{{Name: "alpha", Language: "go", Dir: "/p/alpha"}})
	got, err := r.Get("alpha")
	testhelp.Equal(t, "Get alpha error after mutation", err, nil)
	testhelp.Equal(t, "Get alpha project", got, Project{Name: "alpha", Language: "go", Dir: "/p/alpha"})
}

func rgOpen(t testing.TB, base string) *Registry {
	t.Helper()
	r, err := Open(base)
	testhelp.Equal(t, "Open error", err, nil)
	if r == nil {
		t.Fatal("Open returned nil registry")
	}
	return r
}

func rgNew(t testing.TB) (*Registry, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "state")
	return rgOpen(t, base), base
}

func rgAdd(t testing.TB, r *Registry, p Project) {
	t.Helper()
	testhelp.Equal(t, "Add error", r.Add(p), nil)
}

func rgSeed(t testing.TB) (*Registry, string) {
	t.Helper()
	r, base := rgNew(t)
	rgAdd(t, r, rgZeta())
	rgAdd(t, r, rgAlpha())
	return r, base
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
		Caps:      queue.Caps{ClaudeUSD: 10, Hours: 1, ExecutorUSD: 2},
		CreatedAt: time.Date(2026, 10, 8, 15, 1, 0, 0, time.UTC),
	}
}
