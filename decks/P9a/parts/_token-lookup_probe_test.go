package users

import (
	"path/filepath"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

var pTLAdaHash = "92ba63901405cdae3c83bde1abe474f1d6d4124de3c42b6d25090ab18eaab9cd"
var pTLZoeHash = "501ce40fcde8d3659605c9f745cc9dd6de0a503f13c47579da118d9c71da5ea7"

var pTLAda = User{Name: "ada", LinuxUser: "mu-ada", ChatID: 111, TokenHash: pTLAdaHash, CreatedAt: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)}
var pTLZoe = User{Name: "zoe", LinuxUser: "mu-zoe", ChatID: 999, TokenHash: pTLZoeHash, CreatedAt: time.Date(2026, 10, 11, 9, 1, 0, 0, time.UTC)}

// pTLTwo is the store of example 1: ada and zoe, opened with the admin token.
func pTLTwo(t *testing.T, what string) *Store {
	t.Helper()
	base := filepath.Join(t.TempDir(), "users")
	s, err := Open(base, "admin-0")
	if err != nil {
		t.Fatalf("%s: Open error %v, want nil", what, err)
	}
	if err := s.Add(User{Name: "ada", LinuxUser: "mu-ada", ChatID: 111, CreatedAt: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)}, "tok-ada"); err != nil {
		t.Fatalf("%s: Add(ada) error %v, want nil", what, err)
	}
	if err := s.Add(User{Name: "zoe", LinuxUser: "mu-zoe", ChatID: 999, CreatedAt: time.Date(2026, 10, 11, 9, 1, 0, 0, time.UTC)}, "tok-zoe"); err != nil {
		t.Fatalf("%s: Add(zoe) error %v, want nil", what, err)
	}
	return s
}

func TestProbeTokenLookupExample1(t *testing.T) {
	s := pTLTwo(t, "example 1")
	u, ok := s.Lookup("tok-ada")
	testhelp.Equal(t, "example 1 Lookup(tok-ada)", u, pTLAda)
	testhelp.Equal(t, "example 1 Lookup(tok-ada) ok", ok, true)
	u, ok = s.Lookup("tok-zoe")
	testhelp.Equal(t, "example 1 Lookup(tok-zoe)", u, pTLZoe)
	testhelp.Equal(t, "example 1 Lookup(tok-zoe) ok", ok, true)
	u, ok = s.Lookup("tok-mid")
	testhelp.Equal(t, "example 1 Lookup(tok-mid) user", u, User{})
	testhelp.Equal(t, "example 1 Lookup(tok-mid) ok", ok, false)
}

func TestProbeTokenLookupExample2(t *testing.T) {
	s := pTLTwo(t, "example 2")
	testhelp.Equal(t, "example 2 IsAdminToken(admin-0)", s.IsAdminToken("admin-0"), true)
	testhelp.Equal(t, "example 2 IsAdminToken(tok-ada)", s.IsAdminToken("tok-ada"), false)
	testhelp.Equal(t, "example 2 IsAdminToken(\"\")", s.IsAdminToken(""), false)
	u, ok := s.Lookup("admin-0")
	testhelp.Equal(t, "example 2 Lookup(admin-0) user", u, User{})
	testhelp.Equal(t, "example 2 Lookup(admin-0) ok", ok, false)
}

func TestProbeTokenLookupExample3(t *testing.T) {
	base := filepath.Join(t.TempDir(), "users")
	s, err := Open(base, "")
	if err != nil {
		t.Fatalf("example 3: Open error %v, want nil", err)
	}
	if err := s.Add(User{Name: "ada", LinuxUser: "mu-ada", ChatID: 111}, "tok-ada"); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(User{Name: "zoe", LinuxUser: "mu-zoe", ChatID: 999}, "tok-zoe"); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "example 3 IsAdminToken(admin-0)", s.IsAdminToken("admin-0"), false)
	testhelp.Equal(t, "example 3 IsAdminToken(\"\")", s.IsAdminToken(""), false)
}

func TestProbeTokenLookupExample4(t *testing.T) {
	s := pTLTwo(t, "example 4")
	u, _ := s.Lookup("tok-ada")
	u.Name = "changed"
	again, ok := s.Lookup("tok-ada")
	testhelp.Equal(t, "example 4 Lookup again", again, pTLAda)
	testhelp.Equal(t, "example 4 Lookup again ok", ok, true)
}
