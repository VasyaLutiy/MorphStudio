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

// rgOpen opens the registry under base and fails the test on error.
func rgOpen(t *testing.T, base string) *Registry {
	t.Helper()
	r, err := Open(base)
	if err != nil {
		t.Fatalf("Open(%q): %v", base, err)
	}
	if r == nil {
		t.Fatalf("Open(%q) returned a nil registry", base)
	}
	return r
}

// rgAdd adds p and fails the test on error.
func rgAdd(t *testing.T, r *Registry, p Project) {
	t.Helper()
	if err := r.Add(p); err != nil {
		t.Fatalf("Add(%q): %v", p.Name, err)
	}
}

// rgNames lists the names of ps.
func rgNames(ps []Project) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Name
	}
	return out
}

// rgEqualProject compares got and want field by field, the creation time by
// its JSON round-trip form.
func rgEqualProject(t *testing.T, what string, got, want Project) {
	t.Helper()
	testhelp.Equal(t, what+" Name", got.Name, want.Name)
	testhelp.Equal(t, what+" Language", got.Language, want.Language)
	testhelp.Equal(t, what+" RepoURL", got.RepoURL, want.RepoURL)
	testhelp.Equal(t, what+" Dir", got.Dir, want.Dir)
	testhelp.Equal(t, what+" User", got.User, want.User)
	testhelp.Equal(t, what+" Caps", got.Caps, want.Caps)
	testhelp.Equal(t, what+" CreatedAt", got.CreatedAt.Format(time.RFC3339Nano), want.CreatedAt.Format(time.RFC3339Nano))
}

// rgZeta is the zeta project of the examples.
func rgZeta() Project {
	return Project{
		Name:      "zeta",
		Language:  "go",
		RepoURL:   "https://github.com/o/zeta",
		Dir:       "/p/zeta",
		User:      "mu-zeta",
		Caps:      queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5},
		CreatedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC),
	}
}

// rgAlpha is the alpha project of the examples.
func rgAlpha() Project {
	return Project{
		Name:      "alpha",
		Language:  "python",
		RepoURL:   "https://github.com/o/alpha",
		Dir:       "/p/alpha",
		User:      "mu-alpha",
		Caps:      queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5},
		CreatedAt: time.Date(2026, 10, 8, 15, 1, 0, 0, time.UTC),
	}
}

// rgSeed opens base and adds the zeta and alpha projects.
func rgSeed(t *testing.T, base string) *Registry {
	t.Helper()
	r := rgOpen(t, base)
	rgAdd(t, r, rgZeta())
	rgAdd(t, r, rgAlpha())
	return r
}

func TestProjectRegistryExample1(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")

	r, err := Open(base)
	testhelp.Equal(t, "Open error", err, error(nil))
	if r == nil {
		t.Fatal("Open returned a nil registry")
	}

	info, err := os.Stat(base)
	if err != nil {
		t.Fatalf("Stat(%q): %v", base, err)
	}
	if !info.IsDir() {
		t.Errorf("Stat(%q).IsDir(): got false, want true", base)
	}

	testhelp.Equal(t, "List", r.List(), []Project{})

	got, gotErr := r.Get("demo")
	testhelp.Equal(t, "Get project", got, Project{})
	testhelp.Equal(t, "Get error", gotErr, ErrNotFound)

	testhelp.Equal(t, "Base", r.Base(), base)
}

func TestProjectRegistryExample2(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	r := rgOpen(t, base)

	err := r.Add(rgZeta())
	testhelp.Equal(t, "Add zeta error", err, error(nil))
	err = r.Add(rgAlpha())
	testhelp.Equal(t, "Add alpha error", err, error(nil))

	got := r.List()
	testhelp.Equal(t, "List names", rgNames(got), []string{"alpha", "zeta"})

	data, err := os.ReadFile(filepath.Join(base, "projects.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var decoded []Project
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	testhelp.Equal(t, "decoded projects", decoded, got)

	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	testhelp.Equal(t, "base entries", names, []string{"projects.json"})
}

func TestProjectRegistryExample3(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	r := rgSeed(t, base)

	testhelp.Equal(t, "Add duplicate error", r.Add(Project{Name: "alpha"}), ErrExists)
	testhelp.Equal(t, "Add uppercase error", r.Add(Project{Name: "Alpha"}), ErrBadName)
	testhelp.Equal(t, "Add leading dash error", r.Add(Project{Name: "-x"}), ErrBadName)
	testhelp.Equal(t, "Add too long error", r.Add(Project{Name: strings.Repeat("a", 41)}), ErrBadName)

	testhelp.Equal(t, "List length", len(r.List()), 2)
}

func TestProjectRegistryExample4(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	first := rgSeed(t, base)

	again, err := Open(base)
	testhelp.Equal(t, "Open error", err, error(nil))
	if again == nil {
		t.Fatal("Open returned a nil registry")
	}

	got := again.List()
	want := first.List()
	testhelp.Equal(t, "List names", rgNames(got), rgNames(want))
	if len(got) != len(want) {
		t.Fatalf("List length: got %d, want %d", len(got), len(want))
	}
	for i := range got {
		rgEqualProject(t, "List project", got[i], want[i])
	}

	zeta, zetaErr := again.Get("zeta")
	testhelp.Equal(t, "Get zeta error", zetaErr, error(nil))
	rgEqualProject(t, "Get zeta", zeta, rgZeta())
}

func TestProjectRegistryExample5(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	r := rgSeed(t, base)

	type S struct {
		N int `json:"n"`
	}

	var out S
	testhelp.Equal(t, "SaveState alpha error", r.SaveState("alpha", S{7}), error(nil))
	testhelp.Equal(t, "LoadState alpha error", r.LoadState("alpha", &out), error(nil))
	testhelp.Equal(t, "loaded state", out, S{7})

	var out2 S
	testhelp.Equal(t, "LoadState zeta error", r.LoadState("zeta", &out2), ErrNoState)

	var out3 S
	testhelp.Equal(t, "SaveState nope error", r.SaveState("nope", S{1}), ErrNotFound)
	testhelp.Equal(t, "LoadState nope error", r.LoadState("nope", &out3), ErrNotFound)

	statePath := filepath.Join(base, "alpha", "state.json")
	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("Stat(%q): %v", statePath, err)
	}
	testhelp.Equal(t, "state.json mode", info.Mode()&0o777, os.FileMode(0o600))

	dirInfo, err := os.Stat(filepath.Join(base, "alpha"))
	if err != nil {
		t.Fatalf("Stat alpha dir: %v", err)
	}
	testhelp.Equal(t, "alpha dir mode", dirInfo.Mode()&0o777, os.FileMode(0o700))
}

func TestProjectRegistryExample6(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "projects.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := Open(base)
	testhelp.Equal(t, "registry", r, (*Registry)(nil))
	if err == nil {
		t.Fatal("Open error: got nil, want an error")
	}
	if !strings.HasPrefix(err.Error(), "registry: read ") {
		t.Errorf("Open error:\ngot:  %q\nwant prefix: %q", err.Error(), "registry: read ")
	}
}

func TestProjectRegistryExample7(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state")
	r := rgOpen(t, base)

	alpha := Project{Name: "alpha", Language: "go", Dir: "/p/alpha", User: "mu-alpha"}
	rgAdd(t, r, alpha)

	p := r.List()
	p[0].Name = "changed"
	p = append(p, Project{Name: "extra"})

	g, gErr := r.Get("alpha")
	if gErr != nil {
		t.Fatalf("Get alpha: %v", gErr)
	}
	g.Name = "changed"

	want := []Project{{Name: "alpha", Language: "go", Dir: "/p/alpha", User: "mu-alpha"}}
	testhelp.Equal(t, "List", r.List(), want)

	got, err := r.Get("alpha")
	testhelp.Equal(t, "Get alpha error", err, error(nil))
	testhelp.Equal(t, "Get alpha", got, alpha)
}
