package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"morphstudio/project"
)

// File is a Store that keeps the whole set of projects in one JSON file,
// rewritten atomically after every successful change. It wraps a *Mem for
// in-memory state and the shared semantics.
type File struct {
	path string
	mem  *Mem
}

// OpenFile reads path and returns a File store. A missing file is an empty
// store and is not created. A file that is not a JSON array of
// project.Project yields (nil, error) with text beginning "store: read ".
func OpenFile(path string) (*File, error) {
	f := &File{
		path: path,
		mem:  NewMem(),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return f, nil
		}
		return nil, fmt.Errorf("store: read %s: %w", path, err)
	}

	var projects []project.Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, fmt.Errorf("store: read %s: %w", path, err)
	}

	for _, p := range projects {
		f.mem.projects[p.ID] = p
	}

	return f, nil
}

func (f *File) List() ([]project.Project, error) {
	return f.mem.List()
}

func (f *File) Get(id string) (project.Project, error) {
	return f.mem.Get(id)
}

func (f *File) Create(p project.Project) error {
	f.mem.mu.Lock()
	defer f.mem.mu.Unlock()

	for _, existing := range f.mem.projects {
		if existing.Slug == p.Slug {
			return ErrSlugTaken
		}
	}

	f.mem.projects[p.ID] = p
	if err := f.persistLocked(); err != nil {
		delete(f.mem.projects, p.ID)
		return err
	}
	return nil
}

func (f *File) Update(p project.Project) error {
	f.mem.mu.Lock()
	defer f.mem.mu.Unlock()

	old, ok := f.mem.projects[p.ID]
	if !ok {
		return ErrNotFound
	}

	for id, existing := range f.mem.projects {
		if id != p.ID && existing.Slug == p.Slug {
			return ErrSlugTaken
		}
	}

	f.mem.projects[p.ID] = p
	if err := f.persistLocked(); err != nil {
		f.mem.projects[p.ID] = old
		return err
	}
	return nil
}

func (f *File) Delete(id string) error {
	f.mem.mu.Lock()
	defer f.mem.mu.Unlock()

	old, ok := f.mem.projects[id]
	if !ok {
		return ErrNotFound
	}

	delete(f.mem.projects, id)
	if err := f.persistLocked(); err != nil {
		f.mem.projects[id] = old
		return err
	}
	return nil
}

// persistLocked writes the whole store as one JSON array to a temporary file
// in the same directory and renames it onto path. The caller must hold
// f.mem.mu.
func (f *File) persistLocked() error {
	projects := make([]project.Project, 0, len(f.mem.projects))
	for _, p := range f.mem.projects {
		projects = append(projects, p)
	}

	sort.Slice(projects, func(i, j int) bool {
		if projects[i].CreatedAt.Equal(projects[j].CreatedAt) {
			return projects[i].ID < projects[j].ID
		}
		return projects[i].CreatedAt.Before(projects[j].CreatedAt)
	})

	data, err := json.Marshal(projects)
	if err != nil {
		return err
	}

	dir := filepath.Dir(f.path)
	tmp, err := os.CreateTemp(dir, "projects-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, f.path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
