package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// Build assembles the daemon's HTTP surface: the control API behind a bearer
// token plus the MCP mount that speaks for the daemon, the projects and their
// sessions.
func Build(d control.Control, known func(string) bool, session func(string) (string, bool), token string) http.Handler {
	return api.NewRouter(
		api.Handlers{Control: d},
		token,
		mcpserver.Handler(d, known, mcpserver.Tokens{API: token, Session: session}),
	)
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

	newID := func() string {
		var b [16]byte
		_, _ = rand.Read(b[:])
		return hex.EncodeToString(b[:])
	}

	addr := "127.0.0.1:" + strconv.Itoa(cfg.Port)
	baseURL := "http://" + addr
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

	ctx := context.Background()
	dm, err := daemon.Open(ctx, deps)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	known := func(p string) bool {
		_, err := reg.Get(p)
		return err == nil
	}
	session := func(p string) (string, bool) {
		tok, err := reg.ReadSecret(p, "session-token")
		if err != nil {
			return "", false
		}
		return tok, true
	}

	handler := Build(dm, known, session, cfg.Token)

	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			dm.Tick(ctx)
		}
	}()

	fmt.Println("morphd listening on " + addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
