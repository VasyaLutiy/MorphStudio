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

// urTime builds the CreatedAt times used across the examples.
func urTime(h, m int) time.Time {
	return time.Date(2026, 10, 12, h, m, 0, 0, time.UTC)
}

// urFixture returns the bytes of the three-user fixture.
func urFixture(t testing.TB) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/users/users-3.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// urThree returns the three users of the fixture in file order.
func urThree() []User {
	return []User{
		{
			Name:      "ann",
			LinuxUser: "mu-ann",
			ChatID:    111111111,
			TokenHash: "44d5cbb64a5151b2608de24987b45d9344b8e60386c1a2d4f6ce8904f7e56d2b",
			CreatedAt: time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC),
		},
		{
			Name:      "bob",
			LinuxUser: "mu-bob",
			ChatID:    0,
			TokenHash: "67cfcbd6b49e25979dbe31281596a0765b85aa0b22850cfbfc515247220055ff",
			CreatedAt: time.Date(2026, 10, 12, 9, 5, 0, 0, time.UTC),
		},
		{
			Name:      "eve",
			LinuxUser: "mu-eve",
			ChatID:    333333333,
			TokenHash: "",
			CreatedAt: time.Date(2026, 10, 12, 9, 10, 0, 0, time.UTC),
		},
	}
}

// urOpenFixture writes the fixture into a fresh base/users.json and opens it.
func urOpenFixture(t testing.TB) (*Registry, string) {
	t.Helper()
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "users.json"), urFixture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	return r, base
}

// TestUserRegistryExample1 covers the empty registry created by Open.
func TestUserRegistryExample1(t *testing.T) {
	base := filepath.Join(t.TempDir(), "users")
	r, err := Open(base)
	testhelp.Equal(t, "Open err", err, nil)
	if r == nil {
		t.Fatal("registry is nil")
	}

	testhelp.Equal(t, "List", r.List(), []User{})
	testhelp.Equal(t, "Base", r.Base(), base)

	if _, err := r.Get("ann"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(ann) = %v, want ErrNotFound", err)
	}
	if _, err := r.ByChat(0); !errors.Is(err, ErrNotFound) {
		t.Errorf("ByChat(0) = %v, want ErrNotFound", err)
	}

	fi, err := os.Stat(base)
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "base mode", fi.Mode()&0o777, os.FileMode(0o700))

	testhelp.Equal(t, "ErrNotFound text", ErrNotFound.Error(), "users: user not found")
	testhelp.Equal(t, "ErrExists text", ErrExists.Error(), "users: user exists")
	testhelp.Equal(t, "ErrBadName text", ErrBadName.Error(), "users: bad user name")
	testhelp.Equal(t, "ErrChatTaken text", ErrChatTaken.Error(), "users: chat bound to another user")
	testhelp.Equal(t, "ErrEmptyToken text", ErrEmptyToken.Error(), "users: empty token")
	testhelp.Equal(t, "ErrTokenTaken text", ErrTokenTaken.Error(), "users: token taken")
}

// TestUserRegistryExample2 covers building the fixture from scratch.
func TestUserRegistryExample2(t *testing.T) {
	base := filepath.Join(t.TempDir(), "users")
	r, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}

	eve, err := r.Add("eve", urTime(9, 10))
	testhelp.Equal(t, "Add eve err", err, nil)
	testhelp.Equal(t, "Add eve", eve, User{
		Name:      "eve",
		LinuxUser: "mu-eve",
		ChatID:    0,
		TokenHash: "",
		CreatedAt: urTime(9, 10),
	})

	_, err = r.Add("ann", urTime(9, 0))
	testhelp.Equal(t, "Add ann err", err, nil)
	_, err = r.Add("bob", urTime(9, 5))
	testhelp.Equal(t, "Add bob err", err, nil)
	testhelp.Equal(t, "SetChat ann", r.SetChat("ann", 111111111), nil)
	testhelp.Equal(t, "SetChat eve", r.SetChat("eve", 333333333), nil)
	testhelp.Equal(t, "SetToken ann", r.SetToken("ann", "mcp-ann-0001"), nil)
	testhelp.Equal(t, "SetToken bob", r.SetToken("bob", "mcp-bob-0002"), nil)

	got, err := os.ReadFile(filepath.Join(base, "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "users.json bytes", string(got), string(urFixture(t)))

	fi, err := os.Stat(filepath.Join(base, "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "users.json mode", fi.Mode()&0o777, os.FileMode(0o600))

	ents, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range ents {
		names = append(names, e.Name())
	}
	testhelp.Equal(t, "base contents", names, []string{"users.json"})
}

// TestUserRegistryExample3 covers reading the fixture back.
func TestUserRegistryExample3(t *testing.T) {
	r, _ := urOpenFixture(t)

	testhelp.Equal(t, "List", r.List(), urThree())

	bob, err := r.Get("bob")
	testhelp.Equal(t, "Get bob err", err, nil)
	testhelp.Equal(t, "Get bob", bob, urThree()[1])

	eve, err := r.ByChat(333333333)
	testhelp.Equal(t, "ByChat 333333333 err", err, nil)
	testhelp.Equal(t, "ByChat 333333333", eve, urThree()[2])

	miss, err := r.ByChat(222222222)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("ByChat(222222222) = %v, want ErrNotFound", err)
	}
	testhelp.Equal(t, "ByChat miss", miss, User{})
}

// TestUserRegistryExample4 covers the rejected mutations.
func TestUserRegistryExample4(t *testing.T) {
	r, base := urOpenFixture(t)
	t0 := urTime(10, 0)

	if _, err := r.Add("ann", t0); !errors.Is(err, ErrExists) {
		t.Errorf("Add(ann) = %v, want ErrExists", err)
	}
	for _, name := range []string{"Ann", "mu-x", "1a", "", strings.Repeat("a", 17)} {
		if _, err := r.Add(name, t0); !errors.Is(err, ErrBadName) {
			t.Errorf("Add(%q) = %v, want ErrBadName", name, err)
		}
	}
	if err := r.SetChat("bob", 111111111); !errors.Is(err, ErrChatTaken) {
		t.Errorf("SetChat(bob) = %v, want ErrChatTaken", err)
	}
	if err := r.SetChat("zed", 5); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetChat(zed) = %v, want ErrNotFound", err)
	}
	if err := r.SetToken("eve", ""); !errors.Is(err, ErrEmptyToken) {
		t.Errorf("SetToken(eve, empty) = %v, want ErrEmptyToken", err)
	}
	if err := r.SetToken("eve", "mcp-ann-0001"); !errors.Is(err, ErrTokenTaken) {
		t.Errorf("SetToken(eve, dup) = %v, want ErrTokenTaken", err)
	} else if strings.Contains(err.Error(), "mcp-ann-0001") {
		t.Errorf("error text leaks token: %v", err)
	}
	if err := r.SetToken("zed", "mcp-zed-1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetToken(zed) = %v, want ErrNotFound", err)
	} else if strings.Contains(err.Error(), "mcp-zed-1") {
		t.Errorf("error text leaks token: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(base, "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "users.json unchanged", string(got), string(urFixture(t)))
}

// TestUserRegistryExample5 covers storing only a token's hash.
func TestUserRegistryExample5(t *testing.T) {
	r, base := urOpenFixture(t)

	testhelp.Equal(t, "SetToken eve", r.SetToken("eve", "mcp-new-0003"), nil)

	h := HashToken("mcp-new-0003")
	testhelp.Equal(t, "HashToken", h, "866e38f9ff3bcf4eae882fcd6e81a2637709e9d6a7b5e6ec9fd7ec48ced9c413")

	eve, err := r.Get("eve")
	testhelp.Equal(t, "Get eve err", err, nil)
	testhelp.Equal(t, "Get eve", eve, User{
		Name:      "eve",
		LinuxUser: "mu-eve",
		ChatID:    333333333,
		TokenHash: h,
		CreatedAt: urTime(9, 10),
	})

	b, err := os.ReadFile(filepath.Join(base, "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "file has hash", strings.Contains(string(b), h), true)
	testhelp.Equal(t, "file has no token", strings.Contains(string(b), "mcp-new-0003"), false)
}

// TestUserRegistryExample6 covers unbinding and rebinding a chat.
func TestUserRegistryExample6(t *testing.T) {
	r, base := urOpenFixture(t)

	testhelp.Equal(t, "SetChat ann 0", r.SetChat("ann", 0), nil)
	if _, err := r.ByChat(111111111); !errors.Is(err, ErrNotFound) {
		t.Errorf("ByChat(111111111) after unbind = %v, want ErrNotFound", err)
	}

	testhelp.Equal(t, "SetChat bob", r.SetChat("bob", 111111111), nil)
	bob, err := r.ByChat(111111111)
	testhelp.Equal(t, "ByChat bob err", err, nil)
	testhelp.Equal(t, "ByChat bob", bob, User{
		Name:      "bob",
		LinuxUser: "mu-bob",
		ChatID:    111111111,
		TokenHash: "67cfcbd6b49e25979dbe31281596a0765b85aa0b22850cfbfc515247220055ff",
		CreatedAt: urTime(9, 5),
	})

	testhelp.Equal(t, "SetChat bob again", r.SetChat("bob", 111111111), nil)

	r2, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	gotBob, err := r2.Get("bob")
	testhelp.Equal(t, "reopened Get bob err", err, nil)
	testhelp.Equal(t, "reopened Get bob", gotBob, User{
		Name:      "bob",
		LinuxUser: "mu-bob",
		ChatID:    111111111,
		TokenHash: "67cfcbd6b49e25979dbe31281596a0765b85aa0b22850cfbfc515247220055ff",
		CreatedAt: urTime(9, 5),
	})
	gotAnn, err := r2.Get("ann")
	testhelp.Equal(t, "reopened Get ann err", err, nil)
	testhelp.Equal(t, "reopened ann chat", gotAnn.ChatID, int64(0))
}

// TestUserRegistryExample7 covers a users.json that is not a JSON array.
func TestUserRegistryExample7(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "users.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := Open(base)
	testhelp.Equal(t, "Open registry", r, (*Registry)(nil))
	if err == nil {
		t.Fatal("Open err = nil, want non-nil")
	}
	if !strings.HasPrefix(err.Error(), "users: read ") {
		t.Errorf("Open err = %q, want prefix %q", err.Error(), "users: read ")
	}
}

// TestUserRegistryExample8 covers that returned values are copies.
func TestUserRegistryExample8(t *testing.T) {
	r, _ := urOpenFixture(t)

	l := r.List()
	l[0].Name = "changed"
	l[0].TokenHash = ""

	g, err := r.Get("bob")
	if err != nil {
		t.Fatal(err)
	}
	g.ChatID = 7

	testhelp.Equal(t, "List after writes", r.List(), urThree())
	bob, err := r.Get("bob")
	testhelp.Equal(t, "Get bob err", err, nil)
	testhelp.Equal(t, "Get bob", bob, urThree()[1])
}
