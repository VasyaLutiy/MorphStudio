// Package main is the morphd daemon and its configuration.
package main

import (
	"fmt"
	"strconv"
	"strings"

	"morphstudio/queue"
)

// Config is the daemon's whole configuration, as loaded from the environment
// and a .env file.
type Config struct {
	Token             string
	Port              int
	StateDir          string
	ProjectsDir       string
	ClaudeBin         string
	MorphBin          string
	TGToken           string
	TGChatID          string
	Model             string
	ExtraArgs         []string
	Defaults          queue.Caps
	StretchUSD        float64
	MaxParallel       int
	ResumesPerHour    int
	StallMinutes      int
	UsageAlertPercent int
	DoneWait          int
}

// ParseDotenv parses a .env body into a map. Blank lines and lines whose
// first non-space character is "#" are skipped, as are lines without an "=".
// A leading "export " is removed. The first "=" splits key from value; both
// sides are trimmed and a value wrapped in matching single or double quotes
// is unquoted.
func ParseDotenv(text string) map[string]string {
	out := map[string]string{}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "export "); ok {
			line = strings.TrimSpace(rest)
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" {
			continue
		}
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 {
			first := val[0]
			last := val[len(val)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		out[key] = val
	}
	return out
}

// Load reads every configuration key from getenv, falling back to dotenv and
// then to the built-in default. A missing MORPHD_TOKEN is an error, as is any
// numeric value that does not parse.
func Load(getenv func(string) string, dotenv map[string]string) (Config, error) {
	get := func(key string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return dotenv[key]
	}
	getOr := func(key, def string) string {
		if v := get(key); v != "" {
			return v
		}
		return def
	}
	parseInt := func(key, def string) (int, error) {
		raw := getOr(key, def)
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, fmt.Errorf("config: %s: not a number: %s", key, raw)
		}
		return n, nil
	}
	parseFloat := func(key, def string) (float64, error) {
		raw := getOr(key, def)
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, fmt.Errorf("config: %s: not a number: %s", key, raw)
		}
		return f, nil
	}

	token := get("MORPHD_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("config: MORPHD_TOKEN is required")
	}

	var c Config
	c.Token = token

	var err error
	if c.Port, err = parseInt("MORPHD_PORT", "7080"); err != nil {
		return Config{}, err
	}
	home := getenv("HOME")
	c.StateDir = getOr("MORPHD_STATE_DIR", home+"/.local/state/morphd")
	c.ProjectsDir = getOr("MORPHD_PROJECTS_DIR", home+"/projects")
	c.ClaudeBin = getOr("MORPHD_CLAUDE_BIN", "claude")
	c.MorphBin = getOr("MORPHD_MORPH_BIN", "morph")
	c.TGToken = getOr("TG_BOT_TOKEN", "")
	c.TGChatID = getOr("TG_CHAT_ID", "")
	c.Model = getOr("MORPHD_MODEL", "")

	if raw := get("MORPHD_CLAUDE_EXTRA_ARGS"); raw != "" {
		if fields := strings.Fields(raw); len(fields) > 0 {
			c.ExtraArgs = fields
		}
	}

	if c.Defaults.ClaudeUSD, err = parseFloat("MORPHD_CLAUDE_USD", "30"); err != nil {
		return Config{}, err
	}
	if c.Defaults.Hours, err = parseFloat("MORPHD_HOURS", "3"); err != nil {
		return Config{}, err
	}
	if c.Defaults.ExecutorUSD, err = parseFloat("MORPHD_EXECUTOR_USD", "5"); err != nil {
		return Config{}, err
	}
	if c.StretchUSD, err = parseFloat("MORPHD_STRETCH_USD", "30"); err != nil {
		return Config{}, err
	}
	if c.MaxParallel, err = parseInt("MORPHD_MAX_PARALLEL", "1"); err != nil {
		return Config{}, err
	}
	if c.ResumesPerHour, err = parseInt("MORPHD_RESUMES_PER_HOUR", "3"); err != nil {
		return Config{}, err
	}
	if c.StallMinutes, err = parseInt("MORPHD_STALL_MINUTES", "30"); err != nil {
		return Config{}, err
	}
	if c.UsageAlertPercent, err = parseInt("MORPHD_USAGE_ALERT", "50"); err != nil {
		return Config{}, err
	}
	if c.DoneWait, err = parseInt("MORPHD_DONE_WAIT", "30"); err != nil {
		return Config{}, err
	}
	return c, nil
}

// UUIDv4 renders b as an RFC 4122 version 4 UUID: byte 6 carries the version
// nibble 0x4, byte 8 carries the variant bits 0b10, and the 16 bytes appear
// as lower-case hex in the groups 8-4-4-4-12 joined by "-".
func UUIDv4(b [16]byte) string {
	b[6] = 0x40 | (b[6] & 0x0f)
	b[8] = 0x80 | (b[8] & 0x3f)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
