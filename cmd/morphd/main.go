// Package main is the morphd daemon and its configuration.
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"time"

	"morphstudio/api"
	"morphstudio/claude"
	"morphstudio/control"
	"morphstudio/daemon"
	"morphstudio/mcpserver"
	"morphstudio/registry"
	"morphstudio/runner"
	"morphstudio/supervisor"
	"morphstudio/telegram"
)

// newID returns a fresh version 4 UUID as a 36-character lower-case string.
// It panics when the system random source fails: the daemon cannot run
// without session ids, tokens or interrupt request ids.
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("morphd: cannot read random bytes: " + err.Error())
	}
	return UUIDv4(b)
}

// Build wires the control surface, the MCP mount and the bearer token into
// one HTTP handler.
func Build(d control.Control, known func(string) bool, session func(string) (string, bool), token string) http.Handler {
	mcp := mcpserver.Handler(d, known, mcpserver.Tokens{API: token, Session: session})
	return api.NewRouter(api.Handlers{Control: d}, token, mcp)
}

func main() {
	dotenv := map[string]string{}
	if data, err := os.ReadFile(".env"); err == nil {
		dotenv = ParseDotenv(string(data))
	}

	cfg, err := Load(os.Getenv, dotenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	reg, err := registry.Open(cfg.StateDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	ctx := context.Background()
	tg := telegram.Client{Token: cfg.TGToken, ChatID: cfg.TGChatID, Do: http.DefaultClient.Do}

	deps := daemon.Deps{
		Registry: reg,
		Runner:   runner.OS{},
		Spawn: func(ctx context.Context, l claude.Launch) (claude.Process, error) {
			return claude.Start(ctx, l.Bin, claude.Args(l), l.Dir, claude.Env(l))
		},
		Post:     tg.Post,
		GitHubDo: http.DefaultClient.Do,
		Now:      time.Now,
		NewID:    newID,
		Config: daemon.Config{
			ClaudeBin:   cfg.ClaudeBin,
			MorphBin:    cfg.MorphBin,
			ProjectsDir: cfg.ProjectsDir,
			MCPBaseURL:  fmt.Sprintf("http://127.0.0.1:%d", cfg.Port),
			Model:       cfg.Model,
			ExtraArgs:   cfg.ExtraArgs,
			Defaults:    cfg.Defaults,
			MaxParallel: cfg.MaxParallel,
			DoneWait:    time.Duration(cfg.DoneWait) * time.Second,
			Loop: supervisor.Config{
				MaxResumesPerHour: cfg.ResumesPerHour,
				StallMinutes:      cfg.StallMinutes,
				StretchUSD:        cfg.StretchUSD,
				UsageAlertPercent: cfg.UsageAlertPercent,
			},
		},
	}

	d, err := daemon.Open(ctx, deps)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			d.Tick(ctx)
		}
	}()

	known := func(p string) bool {
		_, err := reg.Get(p)
		return err == nil
	}
	session := func(p string) (string, bool) {
		token, err := reg.ReadSecret(p, "session-token")
		if err != nil {
			return "", false
		}
		return token, true
	}

	handler := Build(d, known, session, cfg.Token)
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
	fmt.Println("morphd listening on " + addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
