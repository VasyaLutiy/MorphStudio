package users

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"morphstudio/internal/testhelp"
)

// psReg: the registry of example 1 of User Registry with user "alice".
func psReg(t *testing.T, what string) (*Users, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "ustate")
	u, err := Open(base, "")
	if err != nil {
		t.Fatalf("%s: Open error %v, want nil", what, err)
	}
	if err := u.Add(User{Name: "alice"}); err != nil {
		t.Fatalf("%s: Add(alice) error %v, want nil", what, err)
	}
	return u, base
}

func psMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Errorf("stat %s: %v", path, err)
		return 0
	}
	return st.Mode() & 0o777
}

func psDir(t *testing.T, dir string) []string {
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
	return out
}

func TestProbeUserSecretsExample1(t *testing.T) {
	u, base := psReg(t, "example 1")
	testhelp.Equal(t, "example 1 PutSecret", u.PutSecret("alice", "anthropic-key", "sk-ant-abc"), error(nil))
	testhelp.Equal(t, "example 1 HasSecret", u.HasSecret("alice", "anthropic-key"), true)
	v, err := u.ReadSecret("alice", "anthropic-key")
	testhelp.Equal(t, "example 1 ReadSecret error", err, error(nil))
	testhelp.Equal(t, "example 1 ReadSecret value", v, "sk-ant-abc")
	testhelp.Equal(t, "example 1 secret mode", psMode(t, filepath.Join(base, "alice", "anthropic-key")), os.FileMode(0o600))
	testhelp.Equal(t, "example 1 dir mode", psMode(t, filepath.Join(base, "alice")), os.FileMode(0o700))
	testhelp.Equal(t, "example 1 dir holds only anthropic-key", psDir(t, filepath.Join(base, "alice")), []string{"anthropic-key"})
	testhelp.Equal(t, "example 1 SecretPath", u.SecretPath("alice", "anthropic-key"), filepath.Join(base, "alice", "anthropic-key"))
	testhelp.Equal(t, "example 1 SecretKinds", SecretKinds, []string{"anthropic-key", "github-token"})
	testhelp.Equal(t, "example 1 sentinel texts", []string{ErrNoSecret.Error(), ErrEmptySecret.Error()}, []string{"no secret", "empty secret"})
}

func TestProbeUserSecretsExample2(t *testing.T) {
	u, base := psReg(t, "example 2")
	testhelp.Equal(t, "example 2 PutSecret(github-token)", u.PutSecret("alice", "github-token", "ghp_xyz"), error(nil))
	v, err := u.ReadSecret("alice", "github-token")
	testhelp.Equal(t, "example 2 ReadSecret error", err, error(nil))
	testhelp.Equal(t, "example 2 ReadSecret value", v, "ghp_xyz")
	testhelp.Equal(t, "example 2 dir holds the two kinds", psDir(t, filepath.Join(base, "alice")), []string{"anthropic-key", "github-token"})
}

func TestProbeUserSecretsExample3(t *testing.T) {
	u, base := psReg(t, "example 3")
	testhelp.Equal(t, "example 3 PutSecret(nope) is ErrNotFound", errors.Is(u.PutSecret("nope", "anthropic-key", "sk-1"), ErrNotFound), true)
	err := u.PutSecret("alice", "password", "p")
	testhelp.Equal(t, "example 3 unknown kind error text", err.Error(), `users: unknown secret kind "password"`)
	testhelp.Equal(t, "example 3 PutSecret(empty) is ErrEmptySecret", errors.Is(u.PutSecret("alice", "github-token", ""), ErrEmptySecret), true)
	testhelp.Equal(t, "example 3 dir still holds only anthropic-key", psDir(t, filepath.Join(base, "alice")), []string{"anthropic-key"})
}

func TestProbeUserSecretsExample4(t *testing.T) {
	u, base := psReg(t, "example 4")
	p := filepath.Join(base, "alice", "anthropic-key")
	if err := u.PutSecret("alice", "anthropic-key", "sk-ant-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "example 4 PutSecret again", u.PutSecret("alice", "anthropic-key", "sk-ant-new"), error(nil))
	testhelp.Equal(t, "example 4 mode 0o600 again", psMode(t, p), os.FileMode(0o600))
	v, err := u.ReadSecret("alice", "anthropic-key")
	testhelp.Equal(t, "example 4 ReadSecret error", err, error(nil))
	testhelp.Equal(t, "example 4 ReadSecret value", v, "sk-ant-new")
}

func TestProbeUserSecretsExample5(t *testing.T) {
	u, _ := psReg(t, "example 5")
	testhelp.Equal(t, "example 5 HasSecret(github-token)", u.HasSecret("alice", "github-token"), false)
	v, err := u.ReadSecret("alice", "github-token")
	testhelp.Equal(t, "example 5 ReadSecret missing value", v, "")
	testhelp.Equal(t, "example 5 ReadSecret missing is ErrNoSecret", errors.Is(err, ErrNoSecret), true)
	_, err = u.ReadSecret("nope", "anthropic-key")
	testhelp.Equal(t, "example 5 ReadSecret(nope) is ErrNotFound", errors.Is(err, ErrNotFound), true)
	testhelp.Equal(t, "example 5 DeleteSecret missing file", u.DeleteSecret("alice", "github-token"), error(nil))
}

func TestProbeUserSecretsExample6(t *testing.T) {
	u, base := psReg(t, "example 6")
	if err := u.PutSecret("alice", "github-token", "ghp_new"); err != nil {
		t.Fatal(err)
	}
	// replace the directory base/alice with a read-only file where the directory was
	if err := os.RemoveAll(filepath.Join(base, "alice")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "alice"), []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	err := u.PutSecret("alice", "github-token", "ghp_new")
	if err == nil || errors.Is(err, ErrNotFound) || errors.Is(err, ErrEmptySecret) {
		t.Errorf("example 6 PutSecret error %v, want a file system error", err)
	}
	if err != nil && strings.Contains(err.Error(), "ghp_new") {
		t.Errorf("example 6 PutSecret error text contains the secret: %q", err.Error())
	}
	_, err = u.ReadSecret("alice", "github-token")
	if err != nil && strings.Contains(err.Error(), "ghp_new") {
		t.Errorf("example 6 ReadSecret error text contains the secret: %q", err.Error())
	}
}
