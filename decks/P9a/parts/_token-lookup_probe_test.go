package users

import (
	"os"
	"path/filepath"
	"testing"

	"morphstudio/internal/testhelp"
)

// pTLThree opens a registry whose base/users.json holds the bytes of the users-3 fixture.
func pTLThree(t *testing.T, what string) *Registry {
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
	return r
}

type pTLGot struct {
	C  Caller
	OK bool
}

func pTLLookup(r *Registry, admin, bearer string) pTLGot {
	c, ok := r.Lookup(admin, bearer)
	return pTLGot{c, ok}
}

func TestProbeTokenLookupExample1(t *testing.T) {
	r := pTLThree(t, "example 1")
	testhelp.Equal(t, "example 1 ann's token", pTLLookup(r, "mcp-admin-9", "mcp-ann-0001"), pTLGot{Caller{User: "ann"}, true})
	testhelp.Equal(t, "example 1 bob's token", pTLLookup(r, "mcp-admin-9", "mcp-bob-0002"), pTLGot{Caller{User: "bob"}, true})
	testhelp.Equal(t, "example 1 the admin token", pTLLookup(r, "mcp-admin-9", "mcp-admin-9"), pTLGot{Caller{Admin: true}, true})
}

func TestProbeTokenLookupExample2(t *testing.T) {
	r := pTLThree(t, "example 2")
	for _, b := range []string{"", "mcp-ann-000", "MCP-ANN-0001", " mcp-ann-0001", "44d5cbb64a5151b2608de24987b45d9344b8e60386c1a2d4f6ce8904f7e56d2b"} {
		testhelp.Equal(t, "example 2 Lookup("+b+")", pTLLookup(r, "mcp-admin-9", b), pTLGot{})
	}
}

func TestProbeTokenLookupExample3(t *testing.T) {
	r := pTLThree(t, "example 3")
	testhelp.Equal(t, "example 3 empty both", pTLLookup(r, "", ""), pTLGot{})
	testhelp.Equal(t, "example 3 ann without an admin token", pTLLookup(r, "", "mcp-ann-0001"), pTLGot{Caller{User: "ann"}, true})
	testhelp.Equal(t, "example 3 empty admin matches nothing", pTLLookup(r, "", "mcp-admin-9"), pTLGot{})
}

func TestProbeTokenLookupExample4(t *testing.T) {
	r := pTLThree(t, "example 4")
	testhelp.Equal(t, "example 4 SetToken(eve)", r.SetToken("eve", "mcp-new-0003"), error(nil))
	testhelp.Equal(t, "example 4 eve's new token", pTLLookup(r, "mcp-admin-9", "mcp-new-0003"), pTLGot{Caller{User: "eve"}, true})
	testhelp.Equal(t, "example 4 SetToken(ann) rotates", r.SetToken("ann", "mcp-ann-0002"), error(nil))
	testhelp.Equal(t, "example 4 ann's old token", pTLLookup(r, "mcp-admin-9", "mcp-ann-0001"), pTLGot{})
	testhelp.Equal(t, "example 4 ann's new token", pTLLookup(r, "mcp-admin-9", "mcp-ann-0002"), pTLGot{Caller{User: "ann"}, true})
}

func TestProbeTokenLookupExample5(t *testing.T) {
	admin, ann, none := Caller{Admin: true}, Caller{User: "ann"}, Caller{}
	testhelp.Equal(t, "example 5 admin Sees(ann), Sees(\"\")", []bool{admin.Sees("ann"), admin.Sees("")}, []bool{true, true})
	testhelp.Equal(t, "example 5 ann Sees(ann), Sees(\"\")", []bool{ann.Sees("ann"), ann.Sees("")}, []bool{true, false})
	testhelp.Equal(t, "example 5 none Sees(ann), Sees(\"\")", []bool{none.Sees("ann"), none.Sees("")}, []bool{false, false})
	testhelp.Equal(t, "example 5 ann Sees(bob)", ann.Sees("bob"), false)
}
