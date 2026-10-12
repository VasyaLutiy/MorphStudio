package users

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

// pUSOne replays the given and the when of example 1: one store, one user "ada",
// one secret anthropic-key = "sk-ant-xyz1" already written.
func pUSOne(t *testing.T, what string) (*Store, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "users")
	s, err := Open(base, "admin-0")
	if err != nil {
		t.Fatalf("%s: Open error %v, want nil", what, err)
	}
	if err := s.Add(User{Name: "ada", LinuxUser: "mu-ada", ChatID: 111, CreatedAt: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)}, "tok-ada"); err != nil {
		t.Fatalf("%s: Add(ada) error %v, want nil", what, err)
	}
	if err := s.PutSecret("ada", "anthropic-key", "sk-ant-xyz1"); err != nil {
		t.Fatalf("%s: PutSecret(anthropic-key) error %v, want nil", what, err)
	}
	return s, base
}

func pUSDir(t *testing.T, dir string) []string {
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

func pUSMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Errorf("stat %s: %v", path, err)
		return 0
	}
	return st.Mode() & 0o777
}

func TestProbeUserSecretsExample1(t *testing.T) {
	s, base := pUSOne(t, "example 1")
	testhelp.Equal(t, "example 1 HasSecret", s.HasSecret("ada", "anthropic-key"), true)
	v, err := s.ReadSecret("ada", "anthropic-key")
	testhelp.Equal(t, "example 1 ReadSecret value", v, "sk-ant-xyz1")
	testhelp.Equal(t, "example 1 ReadSecret error", err, error(nil))
	testhelp.Equal(t, "example 1 secret mode", pUSMode(t, s.SecretPath("ada", "anthropic-key")), os.FileMode(0o600))
	testhelp.Equal(t, "example 1 user dir mode", pUSMode(t, filepath.Join(base, "ada")), os.FileMode(0o700))
	testhelp.Equal(t, "example 1 SecretPath is under base/<name>", s.SecretPath("ada", "anthropic-key"), filepath.Join(base, "ada", "anthropic-key"))
	testhelp.Equal(t, "example 1 dir holds only anthropic-key", pUSDir(t, filepath.Join(base, "ada")), []string{"anthropic-key"})
}

func TestProbeUserSecretsExample2(t *testing.T) {
	s, base := pUSOne(t, "example 2")
	testhelp.Equal(t, "example 2 PutSecret(github-token)", s.PutSecret("ada", "github-token", "ghp_abc"), error(nil))
	v, err := s.ReadSecret("ada", "github-token")
	testhelp.Equal(t, "example 2 ReadSecret value", v, "ghp_abc")
	testhelp.Equal(t, "example 2 ReadSecret error", err, error(nil))
	testhelp.Equal(t, "example 2 github-token mode", pUSMode(t, s.SecretPath("ada", "github-token")), os.FileMode(0o600))
	testhelp.Equal(t, "example 2 dir holds both kinds", pUSDir(t, filepath.Join(base, "ada")), []string{"anthropic-key", "github-token"})
	testhelp.Equal(t, "example 2 SecretKinds", SecretKinds, []string{"anthropic-key", "github-token"})
}

func TestProbeUserSecretsExample3(t *testing.T) {
	s, base := pUSOne(t, "example 3")
	testhelp.Equal(t, "example 3 PutSecret(nope) is ErrNotFound", errors.Is(s.PutSecret("nope", "anthropic-key", "x-1"), ErrNotFound), true)
	err := s.PutSecret("ada", "password", "x-1")
	if err == nil {
		t.Fatalf("example 3: PutSecret(ada, password): nil error, want %q", `users: unknown secret kind "password"`)
	}
	testhelp.Equal(t, "example 3 unknown kind error text", err.Error(), `users: unknown secret kind "password"`)
	testhelp.Equal(t, "example 3 PutSecret empty is ErrEmptySecret", errors.Is(s.PutSecret("ada", "github-token", ""), ErrEmptySecret), true)
	testhelp.Equal(t, "example 3 dir still holds only anthropic-key", pUSDir(t, filepath.Join(base, "ada")), []string{"anthropic-key"})
}

func TestProbeUserSecretsExample4(t *testing.T) {
	s, _ := pUSOne(t, "example 4")
	for _, c := range []struct{ kind, value string }{
		{"anthropic-key", "sk-very-secret-2"},
		{"github-token", "ghp-very-secret-2"},
	} {
		err := s.PutSecret("nope", c.kind, c.value)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("example 4 PutSecret(nope, %s): error %v, want ErrNotFound", c.kind, err)
			continue
		}
		testhelp.Equal(t, "example 4 the "+c.kind+" value never appears in the error text", strings.Contains(err.Error(), c.value), false)
	}
}

func TestProbeUserSecretsExample5(t *testing.T) {
	s, _ := pUSOne(t, "example 5")
	if err := os.Chmod(s.SecretPath("ada", "anthropic-key"), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "example 5 PutSecret again", s.PutSecret("ada", "anthropic-key", "sk-ant-new"), error(nil))
	testhelp.Equal(t, "example 5 mode is 0600 again", pUSMode(t, s.SecretPath("ada", "anthropic-key")), os.FileMode(0o600))
	v, err := s.ReadSecret("ada", "anthropic-key")
	testhelp.Equal(t, "example 5 ReadSecret", v, "sk-ant-new")
	testhelp.Equal(t, "example 5 ReadSecret error", err, error(nil))
}

func TestProbeUserSecretsExample6(t *testing.T) {
	s, _ := pUSOne(t, "example 6")
	testhelp.Equal(t, "example 6 HasSecret(github-token)", s.HasSecret("ada", "github-token"), false)
	v, err := s.ReadSecret("ada", "github-token")
	testhelp.Equal(t, "example 6 ReadSecret(ada, github-token) value", v, "")
	testhelp.Equal(t, "example 6 ReadSecret(ada, github-token) is ErrNoSecret", errors.Is(err, ErrNoSecret), true)
	v2, err2 := s.ReadSecret("nope", "anthropic-key")
	testhelp.Equal(t, "example 6 ReadSecret(nope) value", v2, "")
	testhelp.Equal(t, "example 6 ReadSecret(nope) is ErrNotFound", errors.Is(err2, ErrNotFound), true)
	testhelp.Equal(t, "example 6 DeleteSecret(missing) is nil", s.DeleteSecret("ada", "github-token"), error(nil))
}

func TestProbeUserSecretsExample7(t *testing.T) {
	s, base := pUSOne(t, "example 7")
	if err := s.PutSecret("ada", "github-token", "tok-9"); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "example 7 DeleteSecret(github-token)", s.DeleteSecret("ada", "github-token"), error(nil))
	testhelp.Equal(t, "example 7 HasSecret after delete", s.HasSecret("ada", "github-token"), false)
	testhelp.Equal(t, "example 7 dir holds only anthropic-key", pUSDir(t, filepath.Join(base, "ada")), []string{"anthropic-key"})
}
