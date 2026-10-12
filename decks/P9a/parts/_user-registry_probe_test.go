package users

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

var adaH = "92ba63901405cdae3c83bde1abe474f1d6d4124de3c42b6d25090ab18eaab9cd"
var zoeH = "501ce40fcde8d3659605c9f745cc9dd6de0a503f13c47579da118d9c71da5ea7"

var urAda = User{Name: "ada", LinuxUser: "mu-ada", ChatID: 111, TokenHash: adaH, CreatedAt: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)}
var urZoe = User{Name: "zoe", LinuxUser: "mu-zoe", ChatID: 999, TokenHash: zoeH, CreatedAt: time.Date(2026, 10, 11, 9, 1, 0, 0, time.UTC)}

func urOpen(t *testing.T, what, base string) *Store {
	t.Helper()
	s, err := Open(base, "")
	if err != nil {
		t.Fatalf("%s: Open(%s) error %v, want nil", what, base, err)
	}
	if s == nil {
		t.Fatalf("%s: Open(%s) returned a nil *Store", what, base)
	}
	return s
}

func urNames(us []User) []string {
	out := []string{}
	for _, u := range us {
		out = append(out, u.Name)
	}
	return out
}

func urDir(t *testing.T, dir string) []string {
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

// urTwo is the store of example 2: opened empty, then ada and zoe added with their tokens.
func urTwo(t *testing.T, what string) (*Store, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "users")
	s := urOpen(t, what, base)
	if err := s.Add(User{Name: "ada", LinuxUser: "mu-ada", ChatID: 111, TokenHash: "given", CreatedAt: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)}, "tok-ada"); err != nil {
		t.Fatalf("%s: Add(ada) error %v, want nil", what, err)
	}
	if err := s.Add(User{Name: "zoe", LinuxUser: "mu-zoe", ChatID: 999, CreatedAt: time.Date(2026, 10, 11, 9, 1, 0, 0, time.UTC)}, "tok-zoe"); err != nil {
		t.Fatalf("%s: Add(zoe) error %v, want nil", what, err)
	}
	return s, base
}

func TestProbeUserRegistryExample1(t *testing.T) {
	base := filepath.Join(t.TempDir(), "users")
	s := urOpen(t, "example 1", base)
	st, err := os.Stat(base)
	testhelp.Equal(t, "example 1 base directory exists", err == nil && st.IsDir(), true)
	testhelp.Equal(t, "example 1 base directory mode", st.Mode()&0o777, os.FileMode(0o700))
	l := s.List()
	testhelp.Equal(t, "example 1 List is non-nil", l != nil, true)
	testhelp.Equal(t, "example 1 List", l, []User{})
	u, err := s.Get("ada")
	testhelp.Equal(t, "example 1 Get(ada) user", u, User{})
	testhelp.Equal(t, "example 1 Get(ada) is ErrNotFound", errors.Is(err, ErrNotFound), true)
	testhelp.Equal(t, "example 1 Base", s.Base(), base)
	// variant: another base name, nested two levels deep
	base2 := filepath.Join(t.TempDir(), "x", "users-7")
	s2 := urOpen(t, "example 1 variant", base2)
	testhelp.Equal(t, "example 1 variant Base", s2.Base(), base2)
	testhelp.Equal(t, "example 1 sentinel texts", []string{ErrNotFound.Error(), ErrExists.Error(), ErrBadName.Error(), ErrEmptyToken.Error()}, []string{"user not found", "user exists", "bad user name", "empty token"})
}

func TestProbeUserRegistryExample2(t *testing.T) {
	s, base := urTwo(t, "example 2")
	l := s.List()
	testhelp.Equal(t, "example 2 List names", urNames(l), []string{"ada", "zoe"})
	testhelp.Equal(t, "example 2 List", l, []User{urAda, urZoe})
	b, err := os.ReadFile(filepath.Join(base, "users.json"))
	if err != nil {
		t.Fatalf("example 2: read users.json: %v", err)
	}
	var file []User
	testhelp.Equal(t, "example 2 users.json decodes", json.Unmarshal(b, &file), error(nil))
	testhelp.Equal(t, "example 2 users.json equals List", file, l)
	want, _ := json.MarshalIndent(l, "", "  ")
	testhelp.Equal(t, "example 2 users.json is MarshalIndent with two spaces", strings.TrimRight(string(b), "\n"), string(want))
	testhelp.Equal(t, "example 2 base holds only users.json", urDir(t, base), []string{"users.json"})
	// variant: a third user sorts between them; the file follows
	m := User{Name: "mid", LinuxUser: "mu-mid", ChatID: 555, CreatedAt: time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)}
	testhelp.Equal(t, "example 2 variant Add(mid)", s.Add(m, "tok-mid"), error(nil))
	testhelp.Equal(t, "example 2 variant List", s.List(), []User{urAda, User{Name: "mid", LinuxUser: "mu-mid", ChatID: 555, TokenHash: "73e7b18a59cc7b6e1ac782816c55896f5b6660cda36d65b5911111c0791b690f", CreatedAt: time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)}, urZoe})
	b, _ = os.ReadFile(filepath.Join(base, "users.json"))
	file = nil
	_ = json.Unmarshal(b, &file)
	testhelp.Equal(t, "example 2 variant users.json", file, s.List())
}

func TestProbeUserRegistryExample3(t *testing.T) {
	s, _ := urTwo(t, "example 3")
	testhelp.Equal(t, "example 3 Add(ada) is ErrExists", errors.Is(s.Add(User{Name: "ada"}, "t"), ErrExists), true)
	for _, n := range []string{"Ada", "-x", strings.Repeat("a", 41)} {
		testhelp.Equal(t, "example 3 Add("+n+") is ErrBadName", errors.Is(s.Add(User{Name: n}, "t"), ErrBadName), true)
	}
	testhelp.Equal(t, "example 3 Add with empty token is ErrEmptyToken", errors.Is(s.Add(User{Name: "ok-user"}, ""), ErrEmptyToken), true)
	testhelp.Equal(t, "example 3 List still 2", urNames(s.List()), []string{"ada", "zoe"})
	// variant: the name rule's edges on a fresh store
	s2 := urOpen(t, "example 3 variant", filepath.Join(t.TempDir(), "users"))
	for _, n := range []string{"", "a_b", "a.b", "-", "ab ", "маша"} {
		testhelp.Equal(t, "example 3 variant Add("+n+") is ErrBadName", errors.Is(s2.Add(User{Name: n}, "t"), ErrBadName), true)
	}
	for _, n := range []string{strings.Repeat("b", 40), "9-ok", "x"} {
		testhelp.Equal(t, "example 3 variant Add("+n+")", s2.Add(User{Name: n}, "t"), error(nil))
	}
	testhelp.Equal(t, "example 3 variant List", urNames(s2.List()), []string{"9-ok", strings.Repeat("b", 40), "x"})
}

func TestProbeUserRegistryExample4(t *testing.T) {
	s, base := urTwo(t, "example 4")
	s2 := urOpen(t, "example 4 reopen", base)
	testhelp.Equal(t, "example 4 reopened List", s2.List(), s.List())
	g, err := s2.Get("zoe")
	testhelp.Equal(t, "example 4 Get(zoe)", g, urZoe)
	testhelp.Equal(t, "example 4 Get(zoe) error", err, error(nil))
	// variant: an Add after the reopen keeps the old entries in the file
	testhelp.Equal(t, "example 4 variant Add(mid)", s2.Add(User{Name: "mid", LinuxUser: "mu-mid", ChatID: 555}, "tok-mid"), error(nil))
	s3 := urOpen(t, "example 4 second reopen", base)
	testhelp.Equal(t, "example 4 variant names after reopen", urNames(s3.List()), []string{"ada", "mid", "zoe"})
}

func TestProbeUserRegistryExample5(t *testing.T) {
	base := filepath.Join(t.TempDir(), "users")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	fb, err := os.ReadFile("../tests/fixtures/users/users-3.json")
	if err != nil {
		t.Fatalf("example 5: read fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "users.json"), fb, 0o600); err != nil {
		t.Fatal(err)
	}
	var fixture []User
	if err := json.Unmarshal(fb, &fixture); err != nil {
		t.Fatalf("example 5: fixture decode: %v", err)
	}
	s := urOpen(t, "example 5", base)
	testhelp.Equal(t, "example 5 List equals the fixture", s.List(), fixture)
	testhelp.Equal(t, "example 5 List names", urNames(s.List()), []string{"ada", "mid", "zoe"})
	mid, err := s.Get("mid")
	testhelp.Equal(t, "example 5 Get(mid)", mid, User{Name: "mid", LinuxUser: "mu-mid", ChatID: 555, TokenHash: "73e7b18a59cc7b6e1ac782816c55896f5b6660cda36d65b5911111c0791b690f", CreatedAt: time.Date(2026, 10, 11, 9, 2, 0, 0, time.UTC)})
	testhelp.Equal(t, "example 5 Get(mid) error", err, error(nil))
}

func TestProbeUserRegistryExample6(t *testing.T) {
	for _, text := range []string{"{not json", `{"name":"x"}`} {
		base := t.TempDir()
		if err := os.WriteFile(filepath.Join(base, "users.json"), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		s, err := Open(base, "")
		testhelp.Equal(t, "example 6 Open("+text+") store is nil", s == nil, true)
		if err == nil {
			t.Errorf("example 6 Open(%s) error: nil, want one beginning %q", text, "users: read ")
		} else if !strings.HasPrefix(err.Error(), "users: read ") {
			t.Errorf("example 6 Open(%s) error: %q does not begin %q", text, err.Error(), "users: read ")
		}
	}
}

func TestProbeUserRegistryExample7(t *testing.T) {
	s := urOpen(t, "example 7", filepath.Join(t.TempDir(), "users"))
	want := User{Name: "ada", LinuxUser: "mu-ada", ChatID: 111, CreatedAt: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)}
	testhelp.Equal(t, "example 7 Add(ada)", s.Add(want, "tok-ada"), error(nil))
	want.TokenHash = adaH
	u := s.List()
	if len(u) > 0 {
		u[0].Name = "changed"
	}
	u = append(u, User{Name: "extra"})
	_ = u
	g, _ := s.Get("ada")
	g.Name = "changed"
	testhelp.Equal(t, "example 7 List after writes to returned values", s.List(), []User{want})
	g2, err := s.Get("ada")
	testhelp.Equal(t, "example 7 Get(ada) after writes", g2, want)
	testhelp.Equal(t, "example 7 Get(ada) error", err, error(nil))
	// variant: the slice handed back by one List is not the next one's
	l1 := s.List()
	l2 := s.List()
	if len(l1) > 0 && len(l2) > 0 {
		l1[0].LinuxUser = "/elsewhere"
		testhelp.Equal(t, "example 7 variant two Lists are separate slices", l2[0].LinuxUser, "mu-ada")
	}
}
