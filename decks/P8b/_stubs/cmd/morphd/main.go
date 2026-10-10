package main

import (
	"net/http"

	"morphstudio/control"
)

func newID() string { return "stub newID" }

func Build(d control.Control, known func(string) bool, session func(string) (string, bool), token string) http.Handler {
	return http.NotFoundHandler()
}

func main() {}
