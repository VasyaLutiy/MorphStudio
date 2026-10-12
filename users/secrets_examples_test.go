package users

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"morphstudio/internal/testhelp"
)

// usThree opens a fresh registry whose base/users.json is the bytes of
// tests/fixtures/users/users-3.json, and returns it with its base directory.
func usThree(t testing.TB) (*Registry, string) {
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
	if err != nil {
		t.Fatal(err)
	}
	return r, base
}

// usNames returns the sorted entry names of dir.
func usNames(t testing.TB, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range ents {
		names = append(names, e.Name())
	}
	return names
}

func TestUserSecretsExample1(t *testing.T) {
	r, base := usThree(t)

	err := r.PutSecret("ann", "anthropic-key", "sk-ant-api03-AAAA")
	testhelp.Equal(t, "PutSecret err", err, nil)
	testhelp.Equal(t, "HasSecret", r.HasSecret("ann", "anthropic-key"), true)

	v, err := r.ReadSecret("ann", "anthropic-key")
	testhelp.Equal(t, "ReadSecret err", err, nil)
	testhelp.Equal(t, "ReadSecret value", v, "sk-ant-api03-AAAA")

	p := r.SecretPath("ann", "anthropic-key")
	testhelp.Equal(t, "SecretPath", p, filepath.Join(base, "ann", "anthropic-key"))

	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "secret mode", fi.Mode()&0o777, os.FileMode(0o600))

	di, err := os.Stat(filepath.Join(base, "ann"))
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "ann dir mode", di.Mode()&0o777, os.FileMode(0o700))
	testhelp.Equal(t, "ann dir entries", usNames(t, filepath.Join(base, "ann")), []string{"anthropic-key"})
	testhelp.Equal(t, "base entries", usNames(t, base), []string{"ann", "users.json"})
}

func TestUserSecretsExample2(t *testing.T) {
	r, _ := usThree(t)

	err := r.PutSecret("bob", "github-token", "github_pat_11BBBB\n")
	testhelp.Equal(t, "PutSecret err", err, nil)

	v, err := r.ReadSecret("bob", "github-token")
	testhelp.Equal(t, "ReadSecret err", err, nil)
	testhelp.Equal(t, "ReadSecret value", v, "github_pat_11BBBB\n")
}

func TestUserSecretsExample3(t *testing.T) {
	r, base := usThree(t)

	err := r.PutSecret("zed", "github-token", "x")
	testhelp.Equal(t, "PutSecret zed is ErrNotFound", errors.Is(err, ErrNotFound), true)

	err = r.PutSecret("ann", "password", "sk-ant-api03-AAAA")
	if err == nil {
		t.Fatalf("PutSecret ann password: got nil error, want unknown kind error")
	}
	testhelp.Equal(t, "PutSecret unknown kind text", err.Error(), "users: unknown secret kind \"password\"")

	err = r.PutSecret("ann", "github-token", "")
	testhelp.Equal(t, "PutSecret empty is ErrEmptySecret", errors.Is(err, ErrEmptySecret), true)

	_, statErr := os.Stat(filepath.Join(base, "ann"))
	testhelp.Equal(t, "ann absent", os.IsNotExist(statErr), true)

	testhelp.Equal(t, "SecretKinds", SecretKinds, []string{"anthropic-key", "github-token"})
	testhelp.Equal(t, "ErrNoSecret text", ErrNoSecret.Error(), "users: no secret")
	testhelp.Equal(t, "ErrEmptySecret text", ErrEmptySecret.Error(), "users: empty secret")
}

func TestUserSecretsExample4(t *testing.T) {
	r, base := usThree(t)

	if err := r.PutSecret("ann", "anthropic-key", "sk-ant-api03-AAAA"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(r.SecretPath("ann", "anthropic-key"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := r.PutSecret("ann", "anthropic-key", "sk-ant-api03-CCCC")
	testhelp.Equal(t, "PutSecret err", err, nil)

	fi, err := os.Stat(r.SecretPath("ann", "anthropic-key"))
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "secret mode", fi.Mode()&0o777, os.FileMode(0o600))

	v, err := r.ReadSecret("ann", "anthropic-key")
	testhelp.Equal(t, "ReadSecret err", err, nil)
	testhelp.Equal(t, "ReadSecret value", v, "sk-ant-api03-CCCC")

	testhelp.Equal(t, "ann dir entries", usNames(t, filepath.Join(base, "ann")), []string{"anthropic-key"})
}

func TestUserSecretsExample5(t *testing.T) {
	r, _ := usThree(t)

	testhelp.Equal(t, "HasSecret", r.HasSecret("ann", "github-token"), false)

	v, err := r.ReadSecret("ann", "github-token")
	testhelp.Equal(t, "ReadSecret ann err", errors.Is(err, ErrNoSecret), true)
	testhelp.Equal(t, "ReadSecret ann value", v, "")

	v, err = r.ReadSecret("zed", "github-token")
	testhelp.Equal(t, "ReadSecret zed err", errors.Is(err, ErrNotFound), true)
	testhelp.Equal(t, "ReadSecret zed value", v, "")

	err = r.DeleteSecret("ann", "github-token")
	testhelp.Equal(t, "DeleteSecret ann err", err, nil)

	err = r.DeleteSecret("zed", "github-token")
	testhelp.Equal(t, "DeleteSecret zed err", errors.Is(err, ErrNotFound), true)

	err = r.PutSecret("ann", "github-token", "github_pat_11DDDD")
	testhelp.Equal(t, "PutSecret err", err, nil)

	err = r.DeleteSecret("ann", "github-token")
	testhelp.Equal(t, "DeleteSecret after put err", err, nil)

	testhelp.Equal(t, "HasSecret after delete", r.HasSecret("ann", "github-token"), false)
}

func TestUserSecretsExample6(t *testing.T) {
	r, base := usThree(t)

	if err := os.WriteFile(filepath.Join(base, "eve"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := r.PutSecret("eve", "anthropic-key", "sk-ant-api03-SECRET-EVE")
	if err == nil {
		t.Fatalf("PutSecret eve anthropic-key: got nil error, want non-nil")
	}
	testhelp.Equal(t, "anthropic error hides secret", strings.Contains(err.Error(), "sk-ant-api03-SECRET-EVE"), false)

	err = r.PutSecret("eve", "github-token", "github_pat_11SECRETEVE")
	if err == nil {
		t.Fatalf("PutSecret eve github-token: got nil error, want non-nil")
	}
	testhelp.Equal(t, "github error hides secret", strings.Contains(err.Error(), "github_pat_11SECRETEVE"), false)

	err = r.PutSecret("eve", "password", "sk-ant-api03-SECRET-EVE")
	if err == nil {
		t.Fatalf("PutSecret eve password: got nil error, want unknown kind error")
	}
	testhelp.Equal(t, "password error text", err.Error(), "users: unknown secret kind \"password\"")

	testhelp.Equal(t, "HasSecret eve", r.HasSecret("eve", "anthropic-key"), false)
}
