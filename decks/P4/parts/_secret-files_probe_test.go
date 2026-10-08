package registry

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"morphstudio/internal/testhelp"
)

// pSFDemo is a registry with project "demo" (Add), and its base.
func pSFDemo(t *testing.T, what string) (*Registry, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "state")
	r, err := Open(base)
	if err != nil || r == nil {
		t.Fatalf("%s: Open(%s) = %v, %v; want a registry", what, base, r, err)
	}
	if err := r.Add(Project{Name: "demo", Language: "go"}); err != nil {
		t.Fatalf("%s: Add(demo): %v", what, err)
	}
	return r, base
}

func pSFDir(t *testing.T, dir string) []string {
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

func pSFMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Errorf("stat %s: %v", path, err)
		return 0
	}
	return st.Mode() & 0o777
}

func pSFRead(t *testing.T, what string, r *Registry, name, kind, want string) {
	t.Helper()
	got, err := r.ReadSecret(name, kind)
	testhelp.Equal(t, what+" ReadSecret("+name+", "+kind+")", got, want)
	testhelp.Equal(t, what+" ReadSecret("+name+", "+kind+") error", err, error(nil))
}

func TestProbeSecretFilesExample1(t *testing.T) {
	r, base := pSFDemo(t, "example 1")
	testhelp.Equal(t, "example 1 SecretKinds", SecretKinds, []string{"github-token", "git-credentials", "session-token"})
	testhelp.Equal(t, "example 1 SecretPath", r.SecretPath("demo", "github-token"), filepath.Join(base, "demo", "github-token"))
	testhelp.Equal(t, "example 1 PutSecret", r.PutSecret("demo", "github-token", "ghp_abc"), error(nil))
	testhelp.Equal(t, "example 1 HasSecret", r.HasSecret("demo", "github-token"), true)
	pSFRead(t, "example 1", r, "demo", "github-token", "ghp_abc")
	testhelp.Equal(t, "example 1 secret file mode", pSFMode(t, filepath.Join(base, "demo", "github-token")), os.FileMode(0o600))
	testhelp.Equal(t, "example 1 project dir mode", pSFMode(t, filepath.Join(base, "demo")), os.FileMode(0o700))
	testhelp.Equal(t, "example 1 project dir holds only github-token", pSFDir(t, filepath.Join(base, "demo")), []string{"github-token"})
	// variant: another project and value; the registry's own file is untouched
	if err := r.Add(Project{Name: "web-2"}); err != nil {
		t.Fatalf("example 1 variant Add(web-2): %v", err)
	}
	testhelp.Equal(t, "example 1 variant PutSecret(web-2)", r.PutSecret("web-2", "session-token", "tok-77"), error(nil))
	pSFRead(t, "example 1 variant", r, "web-2", "session-token", "tok-77")
	testhelp.Equal(t, "example 1 variant SecretPath(web-2)", r.SecretPath("web-2", "session-token"), filepath.Join(base, "web-2", "session-token"))
	testhelp.Equal(t, "example 1 variant base", pSFDir(t, base), []string{"demo", "projects.json", "web-2"})
	testhelp.Equal(t, "example 1 variant demo HasSecret(session-token)", r.HasSecret("demo", "session-token"), false)
	testhelp.Equal(t, "example 1 sentinel texts", []string{ErrNoSecret.Error(), ErrEmptySecret.Error()}, []string{"no secret", "empty secret"})
}

func TestProbeSecretFilesExample2(t *testing.T) {
	r, _ := pSFDemo(t, "example 2")
	_ = r.PutSecret("demo", "github-token", "ghp_abc")
	line := "https://x-access-token:ghp_abc@github.com\n"
	testhelp.Equal(t, "example 2 PutSecret(git-credentials)", r.PutSecret("demo", "git-credentials", line), error(nil))
	pSFRead(t, "example 2", r, "demo", "git-credentials", line)
	// variant: white space at both ends is kept verbatim
	testhelp.Equal(t, "example 2 variant PutSecret(session-token)", r.PutSecret("demo", "session-token", "  t-1 \n\n"), error(nil))
	pSFRead(t, "example 2 variant", r, "demo", "session-token", "  t-1 \n\n")
}

func TestProbeSecretFilesExample3(t *testing.T) {
	r, base := pSFDemo(t, "example 3")
	_ = r.PutSecret("demo", "github-token", "ghp_abc")
	testhelp.Equal(t, "example 3 PutSecret(nope) is ErrNotFound", errors.Is(r.PutSecret("nope", "github-token", "x"), ErrNotFound), true)
	err := r.PutSecret("demo", "password", "x")
	if err == nil {
		t.Errorf("example 3 PutSecret(demo, password): nil, want %q", `registry: unknown secret kind "password"`)
	} else {
		testhelp.Equal(t, "example 3 PutSecret(demo, password) error", err.Error(), `registry: unknown secret kind "password"`)
	}
	testhelp.Equal(t, "example 3 PutSecret(demo, session-token, \"\") is ErrEmptySecret", errors.Is(r.PutSecret("demo", "session-token", ""), ErrEmptySecret), true)
	testhelp.Equal(t, "example 3 project dir still holds only github-token", pSFDir(t, filepath.Join(base, "demo")), []string{"github-token"})
	// variant: another unknown kind; the unknown project creates no directory
	err = r.PutSecret("demo", "state.json", "x")
	if err == nil {
		t.Errorf("example 3 variant PutSecret(demo, state.json): nil, want %q", `registry: unknown secret kind "state.json"`)
	} else {
		testhelp.Equal(t, "example 3 variant PutSecret(demo, state.json) error", err.Error(), `registry: unknown secret kind "state.json"`)
	}
	testhelp.Equal(t, "example 3 variant base", pSFDir(t, base), []string{"demo", "projects.json"})
}

func TestProbeSecretFilesExample4(t *testing.T) {
	r, base := pSFDemo(t, "example 4")
	_ = r.PutSecret("demo", "github-token", "ghp_abc")
	path := filepath.Join(base, "demo", "github-token")
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("example 4: chmod: %v", err)
	}
	testhelp.Equal(t, "example 4 PutSecret(ghp_new)", r.PutSecret("demo", "github-token", "ghp_new"), error(nil))
	testhelp.Equal(t, "example 4 mode after rewrite", pSFMode(t, path), os.FileMode(0o600))
	pSFRead(t, "example 4", r, "demo", "github-token", "ghp_new")
	// variant: a file written by someone else with 0o640 and a longer content
	path2 := filepath.Join(base, "demo", "git-credentials")
	if err := os.WriteFile(path2, []byte("https://x-access-token:old-and-much-longer-token@github.com\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "example 4 variant PutSecret(git-credentials)", r.PutSecret("demo", "git-credentials", "short"), error(nil))
	testhelp.Equal(t, "example 4 variant mode", pSFMode(t, path2), os.FileMode(0o600))
	pSFRead(t, "example 4 variant", r, "demo", "git-credentials", "short")
	testhelp.Equal(t, "example 4 variant dir", pSFDir(t, filepath.Join(base, "demo")), []string{"git-credentials", "github-token"})
}

func TestProbeSecretFilesExample5(t *testing.T) {
	r, _ := pSFDemo(t, "example 5")
	_ = r.PutSecret("demo", "github-token", "ghp_abc")
	testhelp.Equal(t, "example 5 HasSecret(session-token)", r.HasSecret("demo", "session-token"), false)
	got, err := r.ReadSecret("demo", "session-token")
	testhelp.Equal(t, "example 5 ReadSecret(demo, session-token) value", got, "")
	testhelp.Equal(t, "example 5 ReadSecret(demo, session-token) is ErrNoSecret", errors.Is(err, ErrNoSecret), true)
	got, err = r.ReadSecret("nope", "github-token")
	testhelp.Equal(t, "example 5 ReadSecret(nope, github-token) value", got, "")
	testhelp.Equal(t, "example 5 ReadSecret(nope, github-token) is ErrNotFound", errors.Is(err, ErrNotFound), true)
	testhelp.Equal(t, "example 5 DeleteSecret(missing)", r.DeleteSecret("demo", "session-token"), error(nil))
	// variant: a project registered with no directory at all
	if err := r.Add(Project{Name: "bare"}); err != nil {
		t.Fatalf("example 5 variant Add(bare): %v", err)
	}
	got, err = r.ReadSecret("bare", "github-token")
	testhelp.Equal(t, "example 5 variant ReadSecret(bare) value", got, "")
	testhelp.Equal(t, "example 5 variant ReadSecret(bare) is ErrNoSecret", errors.Is(err, ErrNoSecret), true)
	testhelp.Equal(t, "example 5 variant HasSecret(bare)", r.HasSecret("bare", "github-token"), false)
}

func TestProbeSecretFilesExample6(t *testing.T) {
	r, base := pSFDemo(t, "example 6")
	_ = r.PutSecret("demo", "github-token", "ghp_abc")
	testhelp.Equal(t, "example 6 PutSecret(session-token)", r.PutSecret("demo", "session-token", "tok-9"), error(nil))
	testhelp.Equal(t, "example 6 HasSecret before delete", r.HasSecret("demo", "session-token"), true)
	testhelp.Equal(t, "example 6 DeleteSecret", r.DeleteSecret("demo", "session-token"), error(nil))
	testhelp.Equal(t, "example 6 HasSecret after delete", r.HasSecret("demo", "session-token"), false)
	testhelp.Equal(t, "example 6 the other secret stays", pSFDir(t, filepath.Join(base, "demo")), []string{"github-token"})
	pSFRead(t, "example 6", r, "demo", "github-token", "ghp_abc")
}
