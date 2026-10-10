package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"strconv"
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

// newID mints a fresh version 4 UUID. The claude CLI refuses any other session
// id shape, and the daemon cannot run without ids, so a read failure panics.
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return UUIDv4(b)
}

// Build assembles the HTTP surface: the bearer-guarded API router plus the
// three MCP mounts.
func Build(d control.Control, known func(string) bool, session func(string) (string, bool), token string) http.Handler {
	return api.NewRouter(api.Handlers{Control: d}, token, mcpserver.Handler(d, known, mcpserver.Tokens{
		API:     token,
		Session: session,
	}))
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
	baseURL := "http://127.0.0.1:" + strconv.Itoa(cfg.Port)

	deps := daemon.Deps{
		Registry: reg,
		Runner:   runner.OS{},
		Spawn: func(ctx context.Context, l claude.Launch) (claude.Process, error) {
			return claude.Start(ctx, l.Bin, claude.Args(l), l.Dir, claude.Env(l))
		},
		Post: telegram.Client{
			Token:  cfg.TGToken,
			ChatID: cfg.TGChatID,
			Do:     http.DefaultClient.Do,
		}.Post,
		GitHubDo: http.DefaultClient.Do,
		Now:      time.Now,
		NewID:    newID,
		Config: daemon.Config{
			ClaudeBin:   cfg.ClaudeBin,
			MorphBin:    cfg.MorphBin,
			ProjectsDir: cfg.ProjectsDir,
			MCPBaseURL:  baseURL,
			Model:       cfg.Model,
			ExtraArgs:   cfg.ExtraArgs,
			Defaults:    cfg.Defaults,
			MaxParallel: cfg.MaxParallel,
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

	known := func(project string) bool {
		_, err := reg.Get(project)
		return err == nil
	}
	session := func(project string) (string, bool) {
		tok, err := reg.ReadSecret(project, "session-token")
		if err != nil {
			return "", false
		}
		return tok, true
	}

	handler := Build(d, known, session, cfg.Token)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			d.Tick(ctx)
		}
	}()

	addr := "127.0.0.1:" + strconv.Itoa(cfg.Port)
	fmt.Println("morphd listening on " + addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
