package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/project"
)

func TestFileStoreExample1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")

	f, err := OpenFile(path)
	if !testhelp.Equal(t, "OpenFile error", err, nil) {
		return
	}
	testhelp.Equal(t, "OpenFile store is nil", f == nil, false)
	if f == nil {
		return
	}

	list, err := f.List()
	testhelp.Equal(t, "List error", err, nil)
	testhelp.Equal(t, "List", list, []project.Project{})

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file should still not exist, stat err = %v", err)
	}
}

func TestFileStoreExample2(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")

	f, err := OpenFile(path)
	if err != nil || f == nil {
		t.Fatalf("OpenFile: store=%v err=%v", f, err)
	}

	b := project.Project{
		ID:          "b",
		Name:        "Beta",
		Slug:        "beta",
		Description: "",
		Status:      "draft",
		CreatedAt:   time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC),
	}
	a := project.Project{
		ID:          "a",
		Name:        "Alpha",
		Slug:        "alpha",
		Description: "",
		Status:      "active",
		CreatedAt:   time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
	}

	if err := f.Create(b); !testhelp.Equal(t, "Create b error", err, nil) {
		return
	}
	if err := f.Create(a); !testhelp.Equal(t, "Create a error", err, nil) {
		return
	}

	list, err := f.List()
	testhelp.Equal(t, "List error", err, nil)
	if !testhelp.Equal(t, "List length", len(list), 2) {
		return
	}
	testhelp.Equal(t, "List[0].ID", list[0].ID, "a")
	testhelp.Equal(t, "List[1].ID", list[1].ID, "b")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []project.Project
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "decoded file", decoded, list)

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	testhelp.Equal(t, "directory entries", names, []string{"projects.json"})
}

func TestFileStoreExample3(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")

	f1, err := OpenFile(path)
	if err != nil || f1 == nil {
		t.Fatalf("OpenFile first: store=%v err=%v", f1, err)
	}
	b := project.Project{
		ID:        "b",
		Name:      "Beta",
		Slug:      "beta",
		Status:    "draft",
		CreatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC),
	}
	a := project.Project{
		ID:        "a",
		Name:      "Alpha",
		Slug:      "alpha",
		Status:    "active",
		CreatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
	}
	if err := f1.Create(b); err != nil {
		t.Fatal(err)
	}
	if err := f1.Create(a); err != nil {
		t.Fatal(err)
	}

	first, err := f1.List()
	if err != nil {
		t.Fatal(err)
	}

	f2, err := OpenFile(path)
	if err != nil || f2 == nil {
		t.Fatalf("OpenFile second: store=%v err=%v", f2, err)
	}
	second, err := f2.List()
	testhelp.Equal(t, "reopened List error", err, nil)
	testhelp.Equal(t, "reopened List", second, first)
}

func TestFileStoreExample4(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")

	f, err := OpenFile(path)
	if err != nil || f == nil {
		t.Fatalf("OpenFile: store=%v err=%v", f, err)
	}
	b := project.Project{
		ID:        "b",
		Name:      "Beta",
		Slug:      "beta",
		Status:    "draft",
		CreatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC),
	}
	a := project.Project{
		ID:        "a",
		Name:      "Alpha",
		Slug:      "alpha",
		Status:    "active",
		CreatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
	}
	if err := f.Create(b); err != nil {
		t.Fatal(err)
	}
	if err := f.Create(a); err != nil {
		t.Fatal(err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	errCreate := f.Create(project.Project{ID: "c", Slug: "alpha"})
	errUpdate := f.Update(project.Project{ID: "zz", Slug: "zz"})
	errDelete := f.Delete("zz")

	testhelp.Equal(t, "Create slug taken", errCreate, ErrSlugTaken)
	testhelp.Equal(t, "Update not found", errUpdate, ErrNotFound)
	testhelp.Equal(t, "Delete not found", errDelete, ErrNotFound)

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "file bytes unchanged", after, before)

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	testhelp.Equal(t, "directory entries", names, []string{"projects.json"})
}

func TestFileStoreExample5(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")

	f, err := OpenFile(path)
	if err != nil || f == nil {
		t.Fatalf("OpenFile: store=%v err=%v", f, err)
	}
	b := project.Project{
		ID:        "b",
		Name:      "Beta",
		Slug:      "beta",
		Status:    "draft",
		CreatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC),
	}
	a := project.Project{
		ID:        "a",
		Name:      "Alpha",
		Slug:      "alpha",
		Status:    "active",
		CreatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
	}
	if err := f.Create(b); err != nil {
		t.Fatal(err)
	}
	if err := f.Create(a); err != nil {
		t.Fatal(err)
	}

	if err := f.Delete("a"); !testhelp.Equal(t, "Delete a error", err, nil) {
		return
	}
	if err := f.Delete("b"); !testhelp.Equal(t, "Delete b error", err, nil) {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "file content", strings.TrimSpace(string(data)), "[]")

	list, err := f.List()
	testhelp.Equal(t, "List error", err, nil)
	testhelp.Equal(t, "List", list, []project.Project{})
}

func TestFileStoreExample6(t *testing.T) {
	path := testhelp.WriteFile(t, "projects.json", "{not json")

	f, err := OpenFile(path)
	testhelp.Equal(t, "OpenFile store", f, (*File)(nil))
	if f != nil {
		return
	}
	if !testhelp.Equal(t, "OpenFile error nil?", err == nil, false) {
		return
	}
	if !strings.HasPrefix(err.Error(), "store: read ") {
		t.Errorf("error text = %q, want prefix %q", err.Error(), "store: read ")
	}
}

func TestFileStoreExample7(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "projects.json")

	f, err := OpenFile(path)
	testhelp.Equal(t, "OpenFile error", err, nil)
	if f == nil {
		t.Fatal("OpenFile returned nil store")
	}

	list, err := f.List()
	testhelp.Equal(t, "List error", err, nil)
	testhelp.Equal(t, "List", list, []project.Project{})

	createErr := f.Create(project.Project{ID: "a", Slug: "alpha"})
	if createErr == nil {
		t.Fatal("Create should have failed")
	}
	testhelp.Equal(t, "Create error is ErrNotFound", errors.Is(createErr, ErrNotFound), false)
	testhelp.Equal(t, "Create error is ErrSlugTaken", errors.Is(createErr, ErrSlugTaken), false)

	list, err = f.List()
	testhelp.Equal(t, "List error after failed Create", err, nil)
	testhelp.Equal(t, "List after failed Create", list, []project.Project{})
}
