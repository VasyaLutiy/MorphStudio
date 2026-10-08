package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/project"
)

var _ Store = (*File)(nil)

func probeT(h int) time.Time { return time.Date(2026, 10, 8, h, 0, 0, 0, time.UTC) }

func probeDir(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	testhelp.Equal(t, "ReadDir error", err, error(nil))
	names := []string{}
	for _, e := range es {
		names = append(names, e.Name())
	}
	return names
}

func probeFile2(t *testing.T) (*File, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "projects.json")
	f, err := OpenFile(path)
	if !testhelp.Equal(t, "OpenFile error", err, error(nil)) || f == nil {
		t.FailNow()
	}
	testhelp.Equal(t, "Create b", f.Create(project.Project{ID: "b", Name: "Beta", Slug: "beta", Status: "draft", CreatedAt: probeT(10), UpdatedAt: probeT(10)}), error(nil))
	testhelp.Equal(t, "Create a", f.Create(project.Project{ID: "a", Name: "Alpha", Slug: "alpha", Status: "active", CreatedAt: probeT(9), UpdatedAt: probeT(9)}), error(nil))
	return f, path
}

func probeList(t *testing.T, s Store) []project.Project {
	t.Helper()
	l, err := s.List()
	testhelp.Equal(t, "List error", err, error(nil))
	return l
}

func TestProbeFileStoreExample1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	f, err := OpenFile(path)
	testhelp.Equal(t, "OpenFile(missing) error", err, error(nil))
	testhelp.Equal(t, "OpenFile(missing) non-nil", f != nil, true)
	if f != nil {
		testhelp.Equal(t, "List", probeList(t, f), []project.Project{})
	}
	_, serr := os.Stat(path)
	testhelp.Equal(t, "file exists after OpenFile", os.IsNotExist(serr), true)
}

func TestProbeFileStoreExample2(t *testing.T) {
	f, path := probeFile2(t)
	b, _ := os.ReadFile(path)
	var decoded []project.Project
	testhelp.Equal(t, "file decodes", json.Unmarshal(b, &decoded), error(nil))
	want := []project.Project{
		{ID: "a", Name: "Alpha", Slug: "alpha", Status: "active", CreatedAt: probeT(9), UpdatedAt: probeT(9)},
		{ID: "b", Name: "Beta", Slug: "beta", Status: "draft", CreatedAt: probeT(10), UpdatedAt: probeT(10)},
	}
	testhelp.Equal(t, "file content", decoded, want)
	testhelp.Equal(t, "List", probeList(t, f), want)
	testhelp.Equal(t, "directory", probeDir(t, filepath.Dir(path)), []string{"projects.json"})
}

func TestProbeFileStoreExample3(t *testing.T) {
	f, path := probeFile2(t)
	g, err := OpenFile(path)
	if !testhelp.Equal(t, "reopen error", err, error(nil)) || g == nil {
		return
	}
	testhelp.Equal(t, "reopened List", probeList(t, g), probeList(t, f))
}

func TestProbeFileStoreExample4(t *testing.T) {
	f, path := probeFile2(t)
	before, _ := os.ReadFile(path)
	testhelp.Equal(t, "Create c alpha", f.Create(project.Project{ID: "c", Slug: "alpha"}), ErrSlugTaken)
	testhelp.Equal(t, "Update zz", f.Update(project.Project{ID: "zz", Slug: "zz"}), ErrNotFound)
	testhelp.Equal(t, "Delete zz", f.Delete("zz"), ErrNotFound)
	after, _ := os.ReadFile(path)
	testhelp.Equal(t, "file bytes", string(after), string(before))
	testhelp.Equal(t, "directory", probeDir(t, filepath.Dir(path)), []string{"projects.json"})
}

func TestProbeFileStoreExample5(t *testing.T) {
	f, path := probeFile2(t)
	testhelp.Equal(t, "Delete a", f.Delete("a"), error(nil))
	testhelp.Equal(t, "Delete b", f.Delete("b"), error(nil))
	b, _ := os.ReadFile(path)
	testhelp.Equal(t, "file content", strings.TrimSpace(string(b)), "[]")
	testhelp.Equal(t, "List", probeList(t, f), []project.Project{})
}

func TestProbeFileStoreExample6(t *testing.T) {
	path := testhelp.WriteFile(t, "projects.json", "{not json")
	f, err := OpenFile(path)
	testhelp.Equal(t, "OpenFile(bad) store", f, (*File)(nil))
	msg := "<nil>"
	if err != nil {
		msg = err.Error()
	}
	testhelp.Equal(t, "OpenFile(bad) error begins \"store: read \"", strings.HasPrefix(msg, "store: read "), true)
}

func TestProbeFileStoreExample7(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "projects.json")
	f, err := OpenFile(path)
	testhelp.Equal(t, "OpenFile error", err, error(nil))
	if f == nil {
		t.Fatal("OpenFile returned a nil store")
	}
	cerr := f.Create(project.Project{ID: "a", Slug: "alpha"})
	testhelp.Equal(t, "Create fails", cerr != nil && cerr != ErrNotFound && cerr != ErrSlugTaken, true)
	testhelp.Equal(t, "List after failed write", probeList(t, f), []project.Project{})
}

func TestProbeFileStoreRows(t *testing.T) {
	f, path := probeFile2(t)
	testhelp.Equal(t, "Update a", f.Update(project.Project{ID: "a", Name: "A2", Slug: "alpha", CreatedAt: probeT(9)}), error(nil))
	g, err := OpenFile(path)
	if !testhelp.Equal(t, "reopen error", err, error(nil)) || g == nil {
		return
	}
	p, _ := g.Get("a")
	testhelp.Equal(t, "reopened Get(a).Name", p.Name, "A2")
	testhelp.Equal(t, "reopened Get(zz)", func() error { _, e := g.Get("zz"); return e }(), ErrNotFound)
}
