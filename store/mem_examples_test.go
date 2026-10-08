package store

import (
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/project"
)

func ids(ps []project.Project) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.ID
	}
	return out
}

func seedExample2(t *testing.T) *Mem {
	t.Helper()
	m := NewMem()
	b := project.Project{ID: "b", Slug: "beta", CreatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)}
	a := project.Project{ID: "a", Slug: "alpha", CreatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)}
	c := project.Project{ID: "c", Slug: "gamma", CreatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)}
	testhelp.Equal(t, "Create b", m.Create(b), nil)
	testhelp.Equal(t, "Create a", m.Create(a), nil)
	testhelp.Equal(t, "Create c", m.Create(c), nil)
	return m
}

func TestMemoryStoreExample1(t *testing.T) {
	var _ Store = (*Mem)(nil)

	m := NewMem()
	got, err := m.List()
	testhelp.Equal(t, "List error", err, nil)
	testhelp.Equal(t, "List", got, []project.Project{})
}

func TestMemoryStoreExample2(t *testing.T) {
	m := NewMem()

	b := project.Project{ID: "b", Slug: "beta", CreatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)}
	a := project.Project{ID: "a", Slug: "alpha", CreatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)}
	c := project.Project{ID: "c", Slug: "gamma", CreatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)}

	testhelp.Equal(t, "Create b", m.Create(b), nil)
	testhelp.Equal(t, "Create a", m.Create(a), nil)
	testhelp.Equal(t, "Create c", m.Create(c), nil)

	got, err := m.List()
	testhelp.Equal(t, "List error", err, nil)
	testhelp.Equal(t, "List IDs", ids(got), []string{"a", "b", "c"})

	g, err := m.Get("b")
	testhelp.Equal(t, "Get b error", err, nil)
	testhelp.Equal(t, "Get b", g, b)
}

func TestMemoryStoreExample3(t *testing.T) {
	m := seedExample2(t)

	testhelp.Equal(t, "Create d", m.Create(project.Project{ID: "d", Slug: "alpha"}), ErrSlugTaken)

	g, err := m.Get("zz")
	testhelp.Equal(t, "Get zz error", err, ErrNotFound)
	testhelp.Equal(t, "Get zz", g, project.Project{})

	testhelp.Equal(t, "Update zz", m.Update(project.Project{ID: "zz", Slug: "alpha"}), ErrNotFound)
	testhelp.Equal(t, "Delete zz", m.Delete("zz"), ErrNotFound)

	got, err := m.List()
	testhelp.Equal(t, "List error", err, nil)
	testhelp.Equal(t, "List IDs", ids(got), []string{"a", "b", "c"})
}

func TestMemoryStoreExample4(t *testing.T) {
	m := seedExample2(t)

	err := m.Update(project.Project{ID: "b", Slug: "beta", Name: "B2", CreatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)})
	testhelp.Equal(t, "Update b beta", err, nil)

	g, err := m.Get("b")
	testhelp.Equal(t, "Get b error", err, nil)
	testhelp.Equal(t, "Get b name", g.Name, "B2")

	err = m.Update(project.Project{ID: "b", Slug: "alpha", CreatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)})
	testhelp.Equal(t, "Update b alpha", err, ErrSlugTaken)

	g, err = m.Get("b")
	testhelp.Equal(t, "Get b error", err, nil)
	testhelp.Equal(t, "Get b slug", g.Slug, "beta")
}

func TestMemoryStoreExample5(t *testing.T) {
	m := seedExample2(t)

	l, err := m.List()
	testhelp.Equal(t, "List error", err, nil)
	if len(l) > 0 {
		l[0].Name = "changed"
	}

	g, err := m.Get("a")
	testhelp.Equal(t, "Get a error", err, nil)
	g.Name = "changed"

	got, err := m.Get("a")
	testhelp.Equal(t, "Get a error", err, nil)
	testhelp.Equal(t, "Get a name", got.Name, "")
}

func TestMemoryStoreExample6(t *testing.T) {
	m := seedExample2(t)

	testhelp.Equal(t, "Delete a", m.Delete("a"), nil)

	_, err := m.Get("a")
	testhelp.Equal(t, "Get a error", err, ErrNotFound)

	testhelp.Equal(t, "Create e", m.Create(project.Project{ID: "e", Slug: "alpha", CreatedAt: time.Date(2026, 10, 8, 8, 0, 0, 0, time.UTC)}), nil)

	got, err := m.List()
	testhelp.Equal(t, "List error", err, nil)
	testhelp.Equal(t, "List IDs", ids(got), []string{"e", "b", "c"})
}
