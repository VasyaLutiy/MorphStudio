package store

import (
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/project"
)

var _ Store = (*Mem)(nil)

func probeAt(h, m int) time.Time { return time.Date(2026, 10, 8, h, m, 0, 0, time.UTC) }

func probeIDs(t *testing.T, s Store) []string {
	t.Helper()
	l, err := s.List()
	testhelp.Equal(t, "List error", err, error(nil))
	ids := []string{}
	for _, p := range l {
		ids = append(ids, p.ID)
	}
	return ids
}

func probeMem2(t *testing.T) *Mem {
	t.Helper()
	m := NewMem()
	testhelp.Equal(t, "Create b", m.Create(project.Project{ID: "b", Slug: "beta", CreatedAt: probeAt(10, 0)}), error(nil))
	testhelp.Equal(t, "Create a", m.Create(project.Project{ID: "a", Slug: "alpha", CreatedAt: probeAt(9, 0)}), error(nil))
	testhelp.Equal(t, "Create c", m.Create(project.Project{ID: "c", Slug: "gamma", CreatedAt: probeAt(10, 0)}), error(nil))
	return m
}

func TestProbeMemoryStoreExample1(t *testing.T) {
	l, err := NewMem().List()
	testhelp.Equal(t, "NewMem().List()", l, []project.Project{})
	testhelp.Equal(t, "NewMem().List() error", err, error(nil))
}

func TestProbeMemoryStoreExample2(t *testing.T) {
	m := probeMem2(t)
	testhelp.Equal(t, "List IDs", probeIDs(t, m), []string{"a", "b", "c"})
	g, err := m.Get("b")
	testhelp.Equal(t, "Get(b)", g, project.Project{ID: "b", Slug: "beta", CreatedAt: probeAt(10, 0)})
	testhelp.Equal(t, "Get(b) error", err, error(nil))
}

func TestProbeMemoryStoreExample3(t *testing.T) {
	m := probeMem2(t)
	testhelp.Equal(t, "Create d with slug alpha", m.Create(project.Project{ID: "d", Slug: "alpha"}), ErrSlugTaken)
	g, err := m.Get("zz")
	testhelp.Equal(t, "Get(zz)", g, project.Project{})
	testhelp.Equal(t, "Get(zz) error", err, ErrNotFound)
	testhelp.Equal(t, "Update zz with slug alpha", m.Update(project.Project{ID: "zz", Slug: "alpha"}), ErrNotFound)
	testhelp.Equal(t, "Delete(zz)", m.Delete("zz"), ErrNotFound)
	testhelp.Equal(t, "List IDs after", probeIDs(t, m), []string{"a", "b", "c"})
}

func TestProbeMemoryStoreExample4(t *testing.T) {
	m := probeMem2(t)
	testhelp.Equal(t, "Update b own slug", m.Update(project.Project{ID: "b", Slug: "beta", Name: "B2", CreatedAt: probeAt(10, 0)}), error(nil))
	g, _ := m.Get("b")
	testhelp.Equal(t, "Get(b).Name", g.Name, "B2")
	testhelp.Equal(t, "Update b to alpha", m.Update(project.Project{ID: "b", Slug: "alpha", CreatedAt: probeAt(10, 0)}), ErrSlugTaken)
	g, _ = m.Get("b")
	testhelp.Equal(t, "Get(b).Slug", g.Slug, "beta")
}

func TestProbeMemoryStoreExample5(t *testing.T) {
	m := probeMem2(t)
	l, _ := m.List()
	if len(l) > 0 {
		l[0].Name = "changed"
	}
	g, _ := m.Get("a")
	g.Name = "changed"
	g2, _ := m.Get("a")
	testhelp.Equal(t, "Get(a) after changing copies", g2, project.Project{ID: "a", Slug: "alpha", CreatedAt: probeAt(9, 0)})
}

func TestProbeMemoryStoreExample6(t *testing.T) {
	m := probeMem2(t)
	testhelp.Equal(t, "Delete(a)", m.Delete("a"), error(nil))
	_, err := m.Get("a")
	testhelp.Equal(t, "Get(a) after Delete", err, ErrNotFound)
	testhelp.Equal(t, "Create e with alpha", m.Create(project.Project{ID: "e", Slug: "alpha", CreatedAt: probeAt(8, 0)}), error(nil))
	testhelp.Equal(t, "List IDs", probeIDs(t, m), []string{"e", "b", "c"})
}

func TestProbeMemoryStoreRows(t *testing.T) {
	testhelp.Equal(t, "ErrNotFound text", ErrNotFound.Error(), "not found")
	testhelp.Equal(t, "ErrSlugTaken text", ErrSlugTaken.Error(), "slug taken")
	m := NewMem()
	testhelp.Equal(t, "Create z", m.Create(project.Project{ID: "z", Slug: "s1", CreatedAt: probeAt(9, 0)}), error(nil))
	testhelp.Equal(t, "Create y", m.Create(project.Project{ID: "y", Slug: "s2", CreatedAt: probeAt(9, 0)}), error(nil))
	testhelp.Equal(t, "Create x", m.Create(project.Project{ID: "x", Slug: "s3", CreatedAt: probeAt(11, 0)}), error(nil))
	testhelp.Equal(t, "List IDs tie by ID", probeIDs(t, m), []string{"y", "z", "x"})
	testhelp.Equal(t, "Delete(y)", m.Delete("y"), error(nil))
	testhelp.Equal(t, "Delete(y) again", m.Delete("y"), ErrNotFound)
	testhelp.Equal(t, "List IDs after delete", probeIDs(t, m), []string{"z", "x"})
}
