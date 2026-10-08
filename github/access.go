package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Access describes the result of checking a GitHub repository.
type Access struct {
	Checked   bool      `json:"checked"`
	Reachable bool      `json:"reachable"`
	Push      bool      `json:"push"`
	Reason    string    `json:"reason"`
	CheckedAt time.Time `json:"checked_at"`
}

// Parse parses a GitHub repository URL.
func Parse(repoURL string) (owner, repo string, err error) {
	const httpsPrefix = "https://github.com/"
	const gitPrefix = "git@github.com:"

	var rest string
	switch {
	case strings.HasPrefix(repoURL, httpsPrefix):
		rest = strings.TrimPrefix(repoURL, httpsPrefix)
		rest = strings.TrimSuffix(rest, "/")
		rest = strings.TrimSuffix(rest, ".git")
	case strings.HasPrefix(repoURL, gitPrefix):
		rest = strings.TrimPrefix(repoURL, gitPrefix)
		if !strings.HasSuffix(rest, ".git") {
			return "", "", errors.New("github: not a github repository url: " + repoURL)
		}
		rest = strings.TrimSuffix(rest, ".git")
	default:
		return "", "", errors.New("github: not a github repository url: " + repoURL)
	}

	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		return "", "", errors.New("github: not a github repository url: " + repoURL)
	}
	owner, repo = parts[0], parts[1]
	if owner == "" || repo == "" || strings.Contains(owner, "/") || strings.Contains(repo, "/") {
		return "", "", errors.New("github: not a github repository url: " + repoURL)
	}
	return owner, repo, nil
}

// CredentialLine returns the git credential-store line for a GitHub token.
func CredentialLine(token string) string {
	return "https://x-access-token:" + token + "@github.com\n"
}

// Check reports whether a token can reach the repository with push access.
func Check(ctx context.Context, do func(*http.Request) (*http.Response, error), token, repoURL string, now time.Time) Access {
	if token == "" {
		return Access{Checked: true, Reason: "no token", CheckedAt: now}
	}

	owner, repo, err := Parse(repoURL)
	if err != nil {
		return Access{Checked: true, Reason: err.Error(), CheckedAt: now}
	}

	if do == nil {
		return Access{Checked: true, Reason: "network: no Do", CheckedAt: now}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	url := "https://api.github.com/repos/" + owner + "/" + repo
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		reason := "network: " + err.Error()
		reason = strings.ReplaceAll(reason, token, "***")
		return Access{Checked: true, Reason: reason, CheckedAt: now}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "morphd")

	resp, err := do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		reason := "network: " + err.Error()
		reason = strings.ReplaceAll(reason, token, "***")
		return Access{Checked: true, Reason: reason, CheckedAt: now}
	}
	if resp == nil {
		return Access{Checked: true, Reason: "network: nil response", CheckedAt: now}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		reason := "network: " + err.Error()
		reason = strings.ReplaceAll(reason, token, "***")
		return Access{Checked: true, Reason: reason, CheckedAt: now}
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var payload struct {
			Permissions struct {
				Push bool `json:"push"`
			} `json:"permissions"`
		}
		_ = json.Unmarshal(body, &payload)
		return Access{Checked: true, Reachable: true, Push: payload.Permissions.Push, Reason: "", CheckedAt: now}
	case http.StatusUnauthorized:
		return Access{Checked: true, Reason: "401 bad credentials", CheckedAt: now}
	case http.StatusForbidden:
		return Access{Checked: true, Reason: "403 forbidden", CheckedAt: now}
	case http.StatusNotFound:
		return Access{Checked: true, Reason: "404 not found or no access", CheckedAt: now}
	default:
		return Access{Checked: true, Reason: fmt.Sprintf("%d unexpected", resp.StatusCode), CheckedAt: now}
	}
}
