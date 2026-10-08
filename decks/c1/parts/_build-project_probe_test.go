package project

import (
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

var probeT1 = time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)

func TestProbeBuildProjectExample1(t *testing.T) {
	got := New(Input{Name: "Alpha", Slug: "alpha", Description: "first", Status: "draft"}, "p1", probeT1)
	testhelp.Equal(t, "New example 1", got, Project{ID: "p1", Name: "Alpha", Slug: "alpha", Description: "first", Status: "draft", CreatedAt: probeT1, UpdatedAt: probeT1})
}

func TestProbeBuildProjectExample2(t *testing.T) {
	at := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	got := New(Input{Name: "Gamma", Slug: "gamma-2", Description: "", Status: "active"}, "7f3a", at)
	testhelp.Equal(t, "New example 2", got, Project{ID: "7f3a", Name: "Gamma", Slug: "gamma-2", Status: "active", CreatedAt: at, UpdatedAt: at})
}

func TestProbeBuildProjectExample3(t *testing.T) {
	got := New(Input{Name: " x ", Slug: "BAD", Description: "d", Status: ""}, "q", probeT1)
	testhelp.Equal(t, "New example 3", got, Project{ID: "q", Name: " x ", Slug: "BAD", Description: "d", Status: "", CreatedAt: probeT1, UpdatedAt: probeT1})
}

func TestProbeBuildProjectExample4(t *testing.T) {
	p := Project{ID: "p1", Name: "Alpha", Slug: "alpha", Description: "first", Status: "draft", CreatedAt: probeT1, UpdatedAt: probeT1}
	at := time.Date(2026, 10, 9, 10, 30, 0, 0, time.UTC)
	got := Apply(p, Input{Name: "Beta", Slug: "beta", Description: "", Status: "archived"}, at)
	testhelp.Equal(t, "Apply example 4", got, Project{ID: "p1", Name: "Beta", Slug: "beta", Description: "", Status: "archived", CreatedAt: probeT1, UpdatedAt: at})
}
