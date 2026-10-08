package project

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

func probeValidate(t *testing.T, in, wantIn Input, wantErr map[string]string) {
	t.Helper()
	got, errs := Validate(in)
	testhelp.Equal(t, fmt.Sprintf("Validate(%#v) input", in), got, wantIn)
	testhelp.Equal(t, fmt.Sprintf("Validate(%#v) errors", in), errs, wantErr)
}

func TestProbeValidateInputExample1(t *testing.T) {
	probeValidate(t,
		Input{Name: "  Morph Studio ", Slug: " morph-studio ", Description: "\tThe backend\n", Status: ""},
		Input{Name: "Morph Studio", Slug: "morph-studio", Description: "The backend", Status: "draft"}, nil)
}

func TestProbeValidateInputExample2(t *testing.T) {
	probeValidate(t, Input{}, Input{Status: "draft"}, map[string]string{"name": "required", "slug": "required"})
}

func TestProbeValidateInputExample3(t *testing.T) {
	name, slug, desc := strings.Repeat("é", 64), strings.Repeat("a", 40), strings.Repeat("ж", 500)
	probeValidate(t,
		Input{Name: " " + name + " ", Slug: slug, Description: desc, Status: "archived"},
		Input{Name: name, Slug: slug, Description: desc, Status: "archived"}, nil)
}

func TestProbeValidateInputExample4(t *testing.T) {
	in := Input{Name: strings.Repeat("é", 65), Slug: strings.Repeat("a", 41), Description: strings.Repeat("ж", 501), Status: "Active"}
	probeValidate(t, in, in, map[string]string{"name": "too long", "slug": "too long", "description": "too long", "status": "invalid"})
}

func TestProbeValidateInputExample5(t *testing.T) {
	slugs := []string{"a1-b2-c3", "0", "Morph", "morph--studio", "-morph", "morph-", "morph_studio", "morph studio", strings.Repeat("A", 41)}
	wants := []string{"", "", "invalid", "invalid", "invalid", "invalid", "invalid", "invalid", "too long"}
	for i, s := range slugs {
		in := Input{Name: "x", Slug: s, Status: "active"}
		var want map[string]string
		if wants[i] != "" {
			want = map[string]string{"slug": wants[i]}
		}
		probeValidate(t, in, in, want)
	}
}

func TestProbeValidateInputExample6(t *testing.T) {
	probeValidate(t,
		Input{Name: "   ", Slug: "ok", Description: "  ", Status: " active "},
		Input{Name: "", Slug: "ok", Description: "", Status: " active "},
		map[string]string{"name": "required", "status": "invalid"})
}

func TestProbeValidateInputRows(t *testing.T) {
	probeValidate(t, Input{Name: "n", Slug: "s", Status: "draft"}, Input{Name: "n", Slug: "s", Status: "draft"}, nil)
	probeValidate(t, Input{Name: "n", Slug: "s", Status: "active"}, Input{Name: "n", Slug: "s", Status: "active"}, nil)
	probeValidate(t, Input{Name: "n", Slug: "s", Status: "deleted"}, Input{Name: "n", Slug: "s", Status: "deleted"}, map[string]string{"status": "invalid"})
	probeValidate(t, Input{Name: "n", Slug: "", Status: "draft"}, Input{Name: "n", Status: "draft"}, map[string]string{"slug": "required"})
	probeValidate(t, Input{Name: "n", Slug: "a-", Status: "draft"}, Input{Name: "n", Slug: "a-", Status: "draft"}, map[string]string{"slug": "invalid"})
	b, _ := json.Marshal(Input{Name: "n"})
	testhelp.Equal(t, "json of Input", string(b), `{"name":"n","slug":"","description":"","status":""}`)
	b, _ = json.Marshal(Project{ID: "i", Status: "draft", CreatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)})
	testhelp.Equal(t, "json of Project", string(b), `{"id":"i","name":"","slug":"","description":"","status":"draft","createdAt":"2026-10-08T09:00:00Z","updatedAt":"0001-01-01T00:00:00Z"}`)
}
