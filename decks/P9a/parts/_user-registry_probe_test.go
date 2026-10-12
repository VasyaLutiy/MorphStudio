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

const pURFixture = "../tests/fixtures/users/users-3.json"

var (
	pURAnn = User{Name: "ann", LinuxUser: "mu-ann", ChatID: 111111111, TokenHash: "44d5cbb64a5151b2608de24987b45d9344b8e60386c1a2d4f6ce8904f7e56d2b", CreatedAt: time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)}
	pURBob = User{Name: "bob", LinuxUser: "mu-bob", ChatID: 0, TokenHash: "67cfcbd6b49e25979dbe31281596a0765b85aa0b22850cfbfc515247220055ff", CreatedAt: time.Date(2026, 10, 12, 9, 5, 0, 0, time.UTC)}
	pUREve = User{Name: "eve", LinuxUser: "mu-eve", ChatID: 333333333, TokenHash: "", CreatedAt: time.Date(2026, 10, 12, 9, 10, 0, 0, time.UTC)}
)

func pURRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func pURMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Errorf("stat %s: %v", path, err)
		return 0
	}
	return st.Mode() & 0o777
}

func pURDir(t *testing.T, dir string) []string {
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

func pUROpen(t *testing.T, what, base string) *Registry {
	t.Helper()
	r, err := Open(base)
	if err != nil {
		t.Fatalf("%s: Open(%s) error %v, want nil", what, base, err)
	}
	if r == nil {
		t.Fatalf("%s: Open(%s) returned a nil *Registry", what, base)
	}
	return r
}

// pURThree is the registry of example 3: base/users.json holds the fixture's bytes.
func pURThree(t *testing.T, what string) (*Registry, string) {
	t.Helper()
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "users.json"), []byte(pURRead(t, pURFixture)), 0o600); err != nil {
		t.Fatal(err)
	}
	return pUROpen(t, what, base), base
}

func TestProbeUserRegistryExample1(t *testing.T) {
	base := filepath.Join(t.TempDir(), "users")
	r := pUROpen(t, "example 1", base)
	testhelp.Equal(t, "example 1 base mode", pURMode(t, base), os.FileMode(0o700))
	l := r.List()
	testhelp.Equal(t, "example 1 List is non-nil", l != nil, true)
	testhelp.Equal(t, "example 1 List", l, []User{})
	u, err := r.Get("ann")
	testhelp.Equal(t, "example 1 Get(ann)", u, User{})
	testhelp.Equal(t, "example 1 Get(ann) is ErrNotFound", errors.Is(err, ErrNotFound), true)
	u, err = r.ByChat(0)
	testhelp.Equal(t, "example 1 ByChat(0)", u, User{})
	testhelp.Equal(t, "example 1 ByChat(0) is ErrNotFound", errors.Is(err, ErrNotFound), true)
	testhelp.Equal(t, "example 1 Base", r.Base(), base)
	testhelp.Equal(t, "example 1 sentinel texts",
		[]string{ErrNotFound.Error(), ErrExists.Error(), ErrBadName.Error(), ErrChatTaken.Error(), ErrEmptyToken.Error(), ErrTokenTaken.Error()},
		[]string{"users: user not found", "users: user exists", "users: bad user name", "users: chat bound to another user", "users: empty token", "users: token taken"})
}

func TestProbeUserRegistryExample2(t *testing.T) {
	base := filepath.Join(t.TempDir(), "users")
	r := pUROpen(t, "example 2", base)
	u, err := r.Add("eve", time.Date(2026, 10, 12, 9, 10, 0, 0, time.UTC))
	testhelp.Equal(t, "example 2 Add(eve)", u, User{Name: "eve", LinuxUser: "mu-eve", CreatedAt: time.Date(2026, 10, 12, 9, 10, 0, 0, time.UTC)})
	testhelp.Equal(t, "example 2 Add(eve) error", err, error(nil))
	_, err = r.Add("ann", time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC))
	testhelp.Equal(t, "example 2 Add(ann) error", err, error(nil))
	_, err = r.Add("bob", time.Date(2026, 10, 12, 9, 5, 0, 0, time.UTC))
	testhelp.Equal(t, "example 2 Add(bob) error", err, error(nil))
	testhelp.Equal(t, "example 2 SetChat(ann)", r.SetChat("ann", 111111111), error(nil))
	testhelp.Equal(t, "example 2 SetChat(eve)", r.SetChat("eve", 333333333), error(nil))
	testhelp.Equal(t, "example 2 SetToken(ann)", r.SetToken("ann", "mcp-ann-0001"), error(nil))
	testhelp.Equal(t, "example 2 SetToken(bob)", r.SetToken("bob", "mcp-bob-0002"), error(nil))
	testhelp.Equal(t, "example 2 users.json equals the fixture", pURRead(t, filepath.Join(base, "users.json")), pURRead(t, pURFixture))
	testhelp.Equal(t, "example 2 users.json mode", pURMode(t, filepath.Join(base, "users.json")), os.FileMode(0o600))
	testhelp.Equal(t, "example 2 base holds only users.json", pURDir(t, base), []string{"users.json"})
}

func TestProbeUserRegistryExample3(t *testing.T) {
	r, _ := pURThree(t, "example 3")
	testhelp.Equal(t, "example 3 List", r.List(), []User{pURAnn, pURBob, pUREve})
	u, err := r.Get("bob")
	testhelp.Equal(t, "example 3 Get(bob)", u, pURBob)
	testhelp.Equal(t, "example 3 Get(bob) error", err, error(nil))
	u, err = r.ByChat(333333333)
	testhelp.Equal(t, "example 3 ByChat(333333333)", u, pUREve)
	testhelp.Equal(t, "example 3 ByChat(333333333) error", err, error(nil))
	u, err = r.ByChat(222222222)
	testhelp.Equal(t, "example 3 ByChat(222222222)", u, User{})
	testhelp.Equal(t, "example 3 ByChat(222222222) is ErrNotFound", errors.Is(err, ErrNotFound), true)
}

func TestProbeUserRegistryExample4(t *testing.T) {
	r, base := pURThree(t, "example 4")
	t0 := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	u, err := r.Add("ann", t0)
	testhelp.Equal(t, "example 4 Add(ann)", u, User{})
	testhelp.Equal(t, "example 4 Add(ann) is ErrExists", errors.Is(err, ErrExists), true)
	for _, n := range []string{"Ann", "mu-x", "1a", "", strings.Repeat("a", 17)} {
		u, err := r.Add(n, t0)
		testhelp.Equal(t, "example 4 Add("+n+")", u, User{})
		testhelp.Equal(t, "example 4 Add("+n+") is ErrBadName", errors.Is(err, ErrBadName), true)
	}
	var texts []string
	err = r.SetChat("bob", 111111111)
	testhelp.Equal(t, "example 4 SetChat(bob, 111111111) is ErrChatTaken", errors.Is(err, ErrChatTaken), true)
	testhelp.Equal(t, "example 4 SetChat(zed) is ErrNotFound", errors.Is(r.SetChat("zed", 5), ErrNotFound), true)
	testhelp.Equal(t, "example 4 SetToken(eve, \"\") is ErrEmptyToken", errors.Is(r.SetToken("eve", ""), ErrEmptyToken), true)
	err = r.SetToken("eve", "mcp-ann-0001")
	testhelp.Equal(t, "example 4 SetToken(eve, ann's token) is ErrTokenTaken", errors.Is(err, ErrTokenTaken), true)
	if err != nil {
		texts = append(texts, err.Error())
	}
	err = r.SetToken("zed", "mcp-zed-1")
	testhelp.Equal(t, "example 4 SetToken(zed) is ErrNotFound", errors.Is(err, ErrNotFound), true)
	if err != nil {
		texts = append(texts, err.Error())
	}
	testhelp.Equal(t, "example 4 users.json unchanged", pURRead(t, filepath.Join(base, "users.json")), pURRead(t, pURFixture))
	for _, s := range texts {
		testhelp.Equal(t, "example 4 error text "+s+" holds no token", strings.Contains(s, "mcp-ann-0001") || strings.Contains(s, "mcp-zed-1"), false)
	}
}

func TestProbeUserRegistryExample5(t *testing.T) {
	r, base := pURThree(t, "example 5")
	const h = "866e38f9ff3bcf4eae882fcd6e81a2637709e9d6a7b5e6ec9fd7ec48ced9c413"
	testhelp.Equal(t, "example 5 SetToken(eve)", r.SetToken("eve", "mcp-new-0003"), error(nil))
	testhelp.Equal(t, "example 5 HashToken", HashToken("mcp-new-0003"), h)
	u, _ := r.Get("eve")
	testhelp.Equal(t, "example 5 eve TokenHash", u.TokenHash, h)
	text := pURRead(t, filepath.Join(base, "users.json"))
	testhelp.Equal(t, "example 5 users.json holds the hash", strings.Contains(text, h), true)
	testhelp.Equal(t, "example 5 users.json holds no token", strings.Contains(text, "mcp-new-0003"), false)
}

func TestProbeUserRegistryExample6(t *testing.T) {
	r, base := pURThree(t, "example 6")
	testhelp.Equal(t, "example 6 SetChat(ann, 0)", r.SetChat("ann", 0), error(nil))
	_, err := r.ByChat(111111111)
	testhelp.Equal(t, "example 6 ByChat(111111111) after unbind is ErrNotFound", errors.Is(err, ErrNotFound), true)
	testhelp.Equal(t, "example 6 SetChat(bob, 111111111)", r.SetChat("bob", 111111111), error(nil))
	u, err := r.ByChat(111111111)
	want := pURBob
	want.ChatID = 111111111
	testhelp.Equal(t, "example 6 ByChat(111111111)", u, want)
	testhelp.Equal(t, "example 6 ByChat(111111111) error", err, error(nil))
	testhelp.Equal(t, "example 6 SetChat(bob, 111111111) again", r.SetChat("bob", 111111111), error(nil))
	r2 := pUROpen(t, "example 6 reopen", base)
	u, _ = r2.Get("bob")
	testhelp.Equal(t, "example 6 reopened bob", u, want)
	u, _ = r2.Get("ann")
	testhelp.Equal(t, "example 6 reopened ann ChatID", u.ChatID, int64(0))
}

func TestProbeUserRegistryExample7(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "users.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Open(base)
	testhelp.Equal(t, "example 7 Open registry is nil", r == nil, true)
	if err == nil {
		t.Errorf("example 7 Open error: nil, want one beginning %q", "users: read ")
	} else if !strings.HasPrefix(err.Error(), "users: read ") {
		t.Errorf("example 7 Open error: %q does not begin %q", err.Error(), "users: read ")
	}
}

func TestProbeUserRegistryExample8(t *testing.T) {
	r, _ := pURThree(t, "example 8")
	l := r.List()
	if len(l) > 0 {
		l[0].Name = "changed"
		l[0].TokenHash = ""
	}
	g, _ := r.Get("bob")
	g.ChatID = 7
	testhelp.Equal(t, "example 8 List after writes to returned values", r.List(), []User{pURAnn, pURBob, pUREve})
	g2, err := r.Get("bob")
	testhelp.Equal(t, "example 8 Get(bob) after writes", g2, pURBob)
	testhelp.Equal(t, "example 8 Get(bob) error", err, error(nil))
}
