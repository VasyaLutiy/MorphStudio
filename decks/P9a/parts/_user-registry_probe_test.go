package users

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

var puBob = User{Name: "bob", LinuxUser: "bob-ux", ChatID: 42, TokenHash: "a0a996e6da7d3784ee7348bee3e43d60f64b72a3ea18196d9f0fbc99a40a8dbd", CreatedAt: time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)}
var puAlice = User{Name: "alice", LinuxUser: "alice-ux", ChatID: 7, TokenHash: "61fdf299956e0522e0a49b4ae572f446b7f811dd73234bc6ddc67aac81d9dcf2", CreatedAt: time.Date(2026, 10, 12, 10, 1, 0, 0, time.UTC)}

func puOpen(t *testing.T, what, base string) *Users {
	t.Helper()
	u, err := Open(base, "")
	if err != nil {
		t.Fatalf("%s: Open(%s) error %v, want nil", what, base, err)
	}
	if u == nil {
		t.Fatalf("%s: Open(%s) returned a nil *Users", what, base)
	}
	return u
}

func puNames(us []User) []string {
	out := []string{}
	for _, v := range us {
		out = append(out, v.Name)
	}
	return out
}

func puDir(t *testing.T, dir string) []string {
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

func puMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Errorf("stat %s: %v", path, err)
		return 0
	}
	return st.Mode() & 0o777
}

// puTwo is the registry after example 2: bob then alice added.
func puTwo(t *testing.T, what string) (*Users, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "ustate")
	u := puOpen(t, what, base)
	if err := u.Add(puBob); err != nil {
		t.Fatalf("%s: Add(bob) error %v, want nil", what, err)
	}
	if err := u.Add(puAlice); err != nil {
		t.Fatalf("%s: Add(alice) error %v, want nil", what, err)
	}
	return u, base
}

func TestProbeUserRegistryExample1(t *testing.T) {
	base := filepath.Join(t.TempDir(), "ustate")
	u := puOpen(t, "example 1", base)
	testhelp.Equal(t, "example 1 base directory mode", puMode(t, base), os.FileMode(0o700))
	l := u.List()
	testhelp.Equal(t, "example 1 List is non-nil", l != nil, true)
	testhelp.Equal(t, "example 1 List", l, []User{})
	g, err := u.Get("alice")
	testhelp.Equal(t, "example 1 Get(alice) user", g, User{})
	testhelp.Equal(t, "example 1 Get(alice) is ErrNotFound", errors.Is(err, ErrNotFound), true)
	testhelp.Equal(t, "example 1 Base", u.Base(), base)
	testhelp.Equal(t, "example 1 sentinel texts", []string{ErrNotFound.Error(), ErrExists.Error(), ErrBadName.Error(), ErrChatTaken.Error()}, []string{"user not found", "user exists", "bad user name", "chat already has a user"})
}

func TestProbeUserRegistryExample2(t *testing.T) {
	u, base := puTwo(t, "example 2")
	l := u.List()
	testhelp.Equal(t, "example 2 List names", puNames(l), []string{"alice", "bob"})
	testhelp.Equal(t, "example 2 List", l, []User{puAlice, puBob})
	b, err := os.ReadFile(filepath.Join(base, "users.json"))
	if err != nil {
		t.Fatalf("example 2: read users.json: %v", err)
	}
	var file []User
	testhelp.Equal(t, "example 2 users.json decodes", json.Unmarshal(b, &file), error(nil))
	testhelp.Equal(t, "example 2 users.json equals List", file, l)
	testhelp.Equal(t, "example 2 base holds only users.json", puDir(t, base), []string{"users.json"})
}

func TestProbeUserRegistryExample3(t *testing.T) {
	u, _ := puTwo(t, "example 3")
	testhelp.Equal(t, "example 3 Add(alice) is ErrExists", errors.Is(u.Add(User{Name: "alice", LinuxUser: "x", ChatID: 9, TokenHash: "h2"}), ErrExists), true)
	testhelp.Equal(t, "example 3 Add(carol with bob's chat) is ErrChatTaken", errors.Is(u.Add(User{Name: "carol", LinuxUser: "c", ChatID: 42, TokenHash: "h3"}), ErrChatTaken), true)
	testhelp.Equal(t, "example 3 Add(Carol) is ErrBadName", errors.Is(u.Add(User{Name: "Carol"}), ErrBadName), true)
	testhelp.Equal(t, "example 3 Add(41 x a) is ErrBadName", errors.Is(u.Add(User{Name: strings.Repeat("a", 41)}), ErrBadName), true)
	testhelp.Equal(t, "example 3 List still 2", puNames(u.List()), []string{"alice", "bob"})
}

func TestProbeUserRegistryExample4(t *testing.T) {
	u, base := puTwo(t, "example 4")
	u2, err := Open(base, "admin-op-9")
	if err != nil {
		t.Fatalf("example 4: reopen error %v, want nil", err)
	}
	testhelp.Equal(t, "example 4 reopened List", u2.List(), u.List())
	g, err := u2.Get("bob")
	testhelp.Equal(t, "example 4 Get(bob)", g, puBob)
	testhelp.Equal(t, "example 4 Get(bob) error", err, error(nil))
}

func TestProbeUserRegistryExample5(t *testing.T) {
	u, base := puTwo(t, "example 5")
	if err := os.MkdirAll(filepath.Join(base, "bob"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "bob", "anthropic-key"), []byte("sk-ant-bob"), 0o600); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "example 5 Delete(bob)", u.Delete("bob"), error(nil))
	_, err := u.Get("bob")
	testhelp.Equal(t, "example 5 Get(bob) is ErrNotFound", errors.Is(err, ErrNotFound), true)
	if _, err := os.Stat(filepath.Join(base, "bob")); !os.IsNotExist(err) {
		t.Errorf("example 5: base/bob still exists, stat err %v", err)
	}
	testhelp.Equal(t, "example 5 List holds only alice", puNames(u.List()), []string{"alice"})
	testhelp.Equal(t, "example 5 Delete(nope) is ErrNotFound", errors.Is(u.Delete("nope"), ErrNotFound), true)
}

func TestProbeUserRegistryExample6(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "users.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	u, err := Open(base, "x")
	testhelp.Equal(t, "example 6 Open({not json) users is nil", u == nil, true)
	if err == nil {
		t.Errorf("example 6 Open({not json) error: nil, want one beginning %q", "users: read ")
	} else if !strings.HasPrefix(err.Error(), "users: read ") {
		t.Errorf("example 6 error %q does not begin %q", err.Error(), "users: read ")
	}
}
