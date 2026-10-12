package users

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

// usAda returns the user "ada" as the examples construct it.
func usAda() User {
	return User{
		Name:      "ada",
		LinuxUser: "mu-ada",
		ChatID:    111,
		CreatedAt: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC),
	}
}

// usNewStore returns a store with user "ada" added and no secrets.
func usNewStore(t testing.TB) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "users"), "admin-0")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Add(usAda(), "tok-ada"); err != nil {
		t.Fatal(err)
	}
	return s
}

// usStoreEx1 returns the store of example 1: ada holding anthropic-key "sk-ant-xyz1".
func usStoreEx1(t testing.TB) *Store {
	t.Helper()
	s := usNewStore(t)
	if err := s.PutSecret("ada", "anthropic-key", "sk-ant-xyz1"); err != nil {
		t.Fatal(err)
	}
	return s
}

// usNames returns the sorted names of the entries of dir.
func usNames(t testing.TB, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		names = append(names, e.Name())
	}
	return names
}

// usFileMode returns the permission bits of the file at path.
func usFileMode(t testing.TB, path string) os.FileMode {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode() & 0o777
}

func TestUserSecretsExample1(t *testing.T) {
	s := usNewStore(t)
	testhelp.Equal(t, "PutSecret error", s.PutSecret("ada", "anthropic-key", "sk-ant-xyz1"), nil)
	testhelp.Equal(t, "HasSecret", s.HasSecret("ada", "anthropic-key"), true)
	got, err := s.ReadSecret("ada", "anthropic-key")
	testhelp.Equal(t, "ReadSecret value", got, "sk-ant-xyz1")
	testhelp.Equal(t, "ReadSecret error", err, nil)
	testhelp.Equal(t, "secret mode", usFileMode(t, s.SecretPath("ada", "anthropic-key")), os.FileMode(0o600))
	testhelp.Equal(t, "user dir mode", usFileMode(t, filepath.Join(s.Base(), "ada")), os.FileMode(0o700))
	testhelp.Equal(t, "dir entries", usNames(t, filepath.Join(s.Base(), "ada")), []string{"anthropic-key"})
}

func TestUserSecretsExample2(t *testing.T) {
	s := usStoreEx1(t)
	testhelp.Equal(t, "PutSecret error", s.PutSecret("ada", "github-token", "ghp_abc"), nil)
	got, err := s.ReadSecret("ada", "github-token")
	testhelp.Equal(t, "ReadSecret value", got, "ghp_abc")
	testhelp.Equal(t, "ReadSecret error", err, nil)
	testhelp.Equal(t, "secret mode", usFileMode(t, s.SecretPath("ada", "github-token")), os.FileMode(0o600))
	testhelp.Equal(t, "dir entries", usNames(t, filepath.Join(s.Base(), "ada")), []string{"anthropic-key", "github-token"})
}

func TestUserSecretsExample3(t *testing.T) {
	s := usStoreEx1(t)
	testhelp.Equal(t, "PutSecret unknown user", s.PutSecret("nope", "anthropic-key", "x-1"), ErrNotFound)
	got := s.PutSecret("ada", "password", "x-1")
	testhelp.Equal(t, "PutSecret unknown kind text", got.Error(), `users: unknown secret kind "password"`)
	testhelp.Equal(t, "PutSecret empty secret", s.PutSecret("ada", "github-token", ""), ErrEmptySecret)
	testhelp.Equal(t, "dir entries", usNames(t, filepath.Join(s.Base(), "ada")), []string{"anthropic-key"})
}

func TestUserSecretsExample4(t *testing.T) {
	s := usStoreEx1(t)
	errA := s.PutSecret("nope", "anthropic-key", "sk-very-secret-2")
	testhelp.Equal(t, "PutSecret anthropic-key", errors.Is(errA, ErrNotFound), true)
	testhelp.Equal(t, "anthropic-key error hides value", strings.Contains(errA.Error(), "sk-very-secret-2"), false)
	errG := s.PutSecret("nope", "github-token", "ghp-very-secret-2")
	testhelp.Equal(t, "PutSecret github-token", errors.Is(errG, ErrNotFound), true)
	testhelp.Equal(t, "github-token error hides value", strings.Contains(errG.Error(), "ghp-very-secret-2"), false)
}

func TestUserSecretsExample5(t *testing.T) {
	s := usStoreEx1(t)
	if err := os.Chmod(s.SecretPath("ada", "anthropic-key"), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "PutSecret error", s.PutSecret("ada", "anthropic-key", "sk-ant-new"), nil)
	testhelp.Equal(t, "secret mode", usFileMode(t, s.SecretPath("ada", "anthropic-key")), os.FileMode(0o600))
	got, err := s.ReadSecret("ada", "anthropic-key")
	testhelp.Equal(t, "ReadSecret value", got, "sk-ant-new")
	testhelp.Equal(t, "ReadSecret error", err, nil)
}

func TestUserSecretsExample6(t *testing.T) {
	s := usStoreEx1(t)
	testhelp.Equal(t, "HasSecret github-token", s.HasSecret("ada", "github-token"), false)
	got, err := s.ReadSecret("ada", "github-token")
	testhelp.Equal(t, "ReadSecret ada value", got, "")
	testhelp.Equal(t, "ReadSecret ada error", err, ErrNoSecret)
	got, err = s.ReadSecret("nope", "anthropic-key")
	testhelp.Equal(t, "ReadSecret nope value", got, "")
	testhelp.Equal(t, "ReadSecret nope error", err, ErrNotFound)
	testhelp.Equal(t, "DeleteSecret error", s.DeleteSecret("ada", "github-token"), nil)
}

func TestUserSecretsExample7(t *testing.T) {
	s := usStoreEx1(t)
	if err := s.PutSecret("ada", "github-token", "tok-9"); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "DeleteSecret error", s.DeleteSecret("ada", "github-token"), nil)
	testhelp.Equal(t, "HasSecret github-token", s.HasSecret("ada", "github-token"), false)
	testhelp.Equal(t, "dir entries", usNames(t, filepath.Join(s.Base(), "ada")), []string{"anthropic-key"})
}
