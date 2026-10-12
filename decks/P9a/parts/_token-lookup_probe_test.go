package users

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"morphstudio/internal/testhelp"
)

// ptFix: a base holding the three-user fixture as users.json, opened with the admin token.
func ptFix(t *testing.T) (*Users, []User, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "ustate")
	b, err := os.ReadFile("../tests/fixtures/users/users-3.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "users.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	var want []User
	if err := json.Unmarshal(b, &want); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	u, err := Open(base, "admin-op-9")
	if err != nil {
		t.Fatalf("Open error %v, want nil", err)
	}
	return u, want, base
}

func TestProbeTokenLookupExample1(t *testing.T) {
	u, want, _ := ptFix(t)
	toks := []string{"tok-bob-2", "tok-alice-1", "tok-carol-3"}
	names := []string{"bob", "alice", "carol"}
	for i, tok := range toks {
		g, ok := u.LookupToken(tok)
		testhelp.Equal(t, "example 1 LookupToken("+names[i]+") ok", ok, true)
		var w User
		for _, c := range want {
			if c.Name == names[i] {
				w = c
			}
		}
		testhelp.Equal(t, "example 1 LookupToken("+names[i]+") user", g, w)
	}
}

func TestProbeTokenLookupExample2(t *testing.T) {
	u, want, _ := ptFix(t)
	g, ok := u.LookupToken("wrong-token")
	testhelp.Equal(t, "example 2 LookupToken(wrong) ok", ok, false)
	testhelp.Equal(t, "example 2 LookupToken(wrong) user", g, User{})
	g, ok = u.LookupToken("")
	testhelp.Equal(t, "example 2 LookupToken(empty) ok", ok, false)
	testhelp.Equal(t, "example 2 LookupToken(empty) user", g, User{})
	testhelp.Equal(t, "example 2 List unchanged", u.List(), want)
}

func TestProbeTokenLookupExample3(t *testing.T) {
	u, _, _ := ptFix(t)
	testhelp.Equal(t, "example 3 IsAdmin(admin token)", u.IsAdmin("admin-op-9"), true)
	testhelp.Equal(t, "example 3 IsAdmin(bob token)", u.IsAdmin("tok-bob-2"), false)
	testhelp.Equal(t, "example 3 IsAdmin(empty)", u.IsAdmin(""), false)
}

func TestProbeTokenLookupExample4(t *testing.T) {
	b, err := os.ReadFile("../tests/fixtures/users/users-3.json")
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(t.TempDir(), "ustate")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "users.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	u, err := Open(base, "")
	if err != nil {
		t.Fatalf("Open with empty admin token error %v, want nil", err)
	}
	testhelp.Equal(t, "example 4 IsAdmin(admin token)", u.IsAdmin("admin-op-9"), false)
	testhelp.Equal(t, "example 4 IsAdmin(empty)", u.IsAdmin(""), false)
}
