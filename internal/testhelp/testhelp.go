// Package testhelp is the module's one helpers package: every test compares with Equal, so a failure prints
// the two lines the acceptance's first-difference locator reads ("got:" then "want:").
package testhelp

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Equal reports what differs, got first, want second, each as a Go literal (%#v), and returns whether they are equal.
func Equal(t testing.TB, what string, got, want any) bool {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return true
	}
	t.Errorf("%s:\ngot:  %#v\nwant: %#v", what, got, want)
	return false
}

// WriteFile writes text to name under a fresh temporary directory of the test and returns the file's path.
func WriteFile(t testing.TB, name, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
