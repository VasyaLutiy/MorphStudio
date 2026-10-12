package users

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"morphstudio/internal/testhelp"
)

// pUSThree opens a registry whose base/users.json holds the bytes of the users-3 fixture.
func pUSThree(t *testing.T, what string) (*Registry, string) {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/users/users-3.json")
	if err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "users.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Open(base)
	if err != nil || r == nil {
		t.Fatalf("%s: Open(%s) = %v, %v; want a registry, nil", what, base, r, err)
	}
	return r, base
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

func TestProbeUserSecretsExample1(t *testing.T) {
	r, base := pUSThree(t, "example 1")
	testhelp.Equal(t, "example 1 PutSecret(ann, anthropic-key)", r.PutSecret("ann", "anthropic-key", "sk-ant-api03-AAAA"), error(nil))
	testhelp.Equal(t, "example 1 HasSecret", r.HasSecret("ann", "anthropic-key"), true)
	v, err := r.ReadSecret("ann", "anthropic-key")
	testhelp.Equal(t, "example 1 ReadSecret", v, "sk-ant-api03-AAAA")
	testhelp.Equal(t, "example 1 ReadSecret error", err, error(nil))
	testhelp.Equal(t, "example 1 SecretPath", r.SecretPath("ann", "anthropic-key"), filepath.Join(base, "ann", "anthropic-key"))
	testhelp.Equal(t, "example 1 file mode", pUSMode(t, filepath.Join(base, "ann", "anthropic-key")), os.FileMode(0o600))
	testhelp.Equal(t, "example 1 dir mode", pUSMode(t, filepath.Join(base, "ann")), os.FileMode(0o700))
	testhelp.Equal(t, "example 1 base/ann holds only anthropic-key", pUSDir(t, filepath.Join(base, "ann")), []string{"anthropic-key"})
	testhelp.Equal(t, "example 1 base holds ann and users.json", pUSDir(t, base), []string{"ann", "users.json"})
}

func TestProbeUserSecretsExample2(t *testing.T) {
	r, _ := pUSThree(t, "example 2")
	testhelp.Equal(t, "example 2 PutSecret(bob, github-token)", r.PutSecret("bob", "github-token", "github_pat_11BBBB\n"), error(nil))
	v, err := r.ReadSecret("bob", "github-token")
	testhelp.Equal(t, "example 2 ReadSecret keeps the newline", v, "github_pat_11BBBB\n")
	testhelp.Equal(t, "example 2 ReadSecret error", err, error(nil))
}

func TestProbeUserSecretsExample3(t *testing.T) {
	r, base := pUSThree(t, "example 3")
	testhelp.Equal(t, "example 3 PutSecret(zed) is ErrNotFound", errors.Is(r.PutSecret("zed", "github-token", "x"), ErrNotFound), true)
	err := r.PutSecret("ann", "password", "sk-ant-api03-AAAA")
	text := "<nil>"
	if err != nil {
		text = err.Error()
	}
	testhelp.Equal(t, "example 3 PutSecret(ann, password) text", text, `users: unknown secret kind "password"`)
	testhelp.Equal(t, "example 3 PutSecret(ann, github-token, \"\") is ErrEmptySecret", errors.Is(r.PutSecret("ann", "github-token", ""), ErrEmptySecret), true)
	_, statErr := os.Stat(filepath.Join(base, "ann"))
	testhelp.Equal(t, "example 3 base/ann does not exist", os.IsNotExist(statErr), true)
	testhelp.Equal(t, "example 3 SecretKinds", SecretKinds, []string{"anthropic-key", "github-token"})
	testhelp.Equal(t, "example 3 sentinel texts", []string{ErrNoSecret.Error(), ErrEmptySecret.Error()}, []string{"users: no secret", "users: empty secret"})
}

func TestProbeUserSecretsExample4(t *testing.T) {
	r, base := pUSThree(t, "example 4")
	testhelp.Equal(t, "example 4 first PutSecret", r.PutSecret("ann", "anthropic-key", "sk-ant-api03-AAAA"), error(nil))
	if err := os.Chmod(r.SecretPath("ann", "anthropic-key"), 0o644); err != nil {
		t.Errorf("example 4 chmod: %v", err)
	}
	testhelp.Equal(t, "example 4 second PutSecret", r.PutSecret("ann", "anthropic-key", "sk-ant-api03-CCCC"), error(nil))
	testhelp.Equal(t, "example 4 mode back to 0600", pUSMode(t, filepath.Join(base, "ann", "anthropic-key")), os.FileMode(0o600))
	v, err := r.ReadSecret("ann", "anthropic-key")
	testhelp.Equal(t, "example 4 ReadSecret", v, "sk-ant-api03-CCCC")
	testhelp.Equal(t, "example 4 ReadSecret error", err, error(nil))
	testhelp.Equal(t, "example 4 base/ann holds only anthropic-key", pUSDir(t, filepath.Join(base, "ann")), []string{"anthropic-key"})
}

func TestProbeUserSecretsExample5(t *testing.T) {
	r, _ := pUSThree(t, "example 5")
	testhelp.Equal(t, "example 5 HasSecret before", r.HasSecret("ann", "github-token"), false)
	v, err := r.ReadSecret("ann", "github-token")
	testhelp.Equal(t, "example 5 ReadSecret missing value", v, "")
	testhelp.Equal(t, "example 5 ReadSecret missing is ErrNoSecret", errors.Is(err, ErrNoSecret), true)
	v, err = r.ReadSecret("zed", "github-token")
	testhelp.Equal(t, "example 5 ReadSecret(zed) value", v, "")
	testhelp.Equal(t, "example 5 ReadSecret(zed) is ErrNotFound", errors.Is(err, ErrNotFound), true)
	testhelp.Equal(t, "example 5 DeleteSecret missing", r.DeleteSecret("ann", "github-token"), error(nil))
	testhelp.Equal(t, "example 5 DeleteSecret(zed) is ErrNotFound", errors.Is(r.DeleteSecret("zed", "github-token"), ErrNotFound), true)
	testhelp.Equal(t, "example 5 PutSecret", r.PutSecret("ann", "github-token", "github_pat_11DDDD"), error(nil))
	testhelp.Equal(t, "example 5 DeleteSecret", r.DeleteSecret("ann", "github-token"), error(nil))
	testhelp.Equal(t, "example 5 HasSecret after delete", r.HasSecret("ann", "github-token"), false)
}

func TestProbeUserSecretsExample6(t *testing.T) {
	r, base := pUSThree(t, "example 6")
	if err := os.WriteFile(filepath.Join(base, "eve"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := r.PutSecret("eve", "anthropic-key", "sk-ant-api03-SECRET-EVE")
	testhelp.Equal(t, "example 6 anthropic-key: an error", err != nil, true)
	if err != nil {
		testhelp.Equal(t, "example 6 anthropic-key: the error text holds no key", strings.Contains(err.Error(), "sk-ant-api03-SECRET-EVE"), false)
	}
	err = r.PutSecret("eve", "github-token", "github_pat_11SECRETEVE")
	testhelp.Equal(t, "example 6 github-token: an error", err != nil, true)
	if err != nil {
		testhelp.Equal(t, "example 6 github-token: the error text holds no PAT", strings.Contains(err.Error(), "github_pat_11SECRETEVE"), false)
	}
	err = r.PutSecret("eve", "password", "sk-ant-api03-SECRET-EVE")
	text := "<nil>"
	if err != nil {
		text = err.Error()
	}
	testhelp.Equal(t, "example 6 unknown kind text", text, `users: unknown secret kind "password"`)
	testhelp.Equal(t, "example 6 HasSecret(eve, anthropic-key)", r.HasSecret("eve", "anthropic-key"), false)
}
