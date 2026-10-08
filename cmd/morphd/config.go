package main

import (
	"fmt"
	"strconv"
	"strings"

	"morphstudio/queue"
)

// Config holds every setting of the morphd daemon.
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
}

// ParseDotenv parses a .env file into a key-to-value map. It trims each line,
// skips blank lines and lines beginning with "#", removes an "export "
// prefix, splits on the first "=", trims both sides and strips a pair of
// matching single or double quotes around the value.
func ParseDotenv(text string) map[string]string {
	out := map[string]string{}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		i := strings.Index(line, "=")
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		if key == "" {
			continue
		}
		val := strings.TrimSpace(line[i+1:])
		if len(val) >= 2 {
			first, last := val[0], val[len(val)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		out[key] = val
	}
	return out
}

// Load reads a Config from getenv with dotenv as fallback and the documented
// defaults. A numeric setting that does not parse is an error; the message
// names the key and the offending value, without echoing any secret.
func Load(getenv func(string) string, dotenv map[string]string) (Config, error) {
	lookup := func(key string) string {
		if v := getenv(key); v != "" {
			return v
		}
		if v := dotenv[key]; v != "" {
			return v
		}
		return ""
	}

	intVal := func(key string, def int) (int, error) {
		raw := lookup(key)
		if raw == "" {
			return def, nil
		}
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, fmt.Errorf("config: %s: not a number: %s", key, raw)
		}
		return n, nil
	}

	floatVal := func(key string, def float64) (float64, error) {
		raw := lookup(key)
		if raw == "" {
			return def, nil
		}
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, fmt.Errorf("config: %s: not a number: %s", key, raw)
		}
		return n, nil
	}

	token := lookup("MORPHD_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("config: MORPHD_TOKEN is required")
	}

	port, err := intVal("MORPHD_PORT", 7080)
	if err != nil {
		return Config{}, err
	}
	claudeUSD, err := floatVal("MORPHD_CLAUDE_USD", 30)
	if err != nil {
		return Config{}, err
	}
	hours, err := floatVal("MORPHD_HOURS", 3)
	if err != nil {
		return Config{}, err
	}
	executorUSD, err := floatVal("MORPHD_EXECUTOR_USD", 5)
	if err != nil {
		return Config{}, err
	}
	stretchUSD, err := floatVal("MORPHD_STRETCH_USD", 30)
	if err != nil {
		return Config{}, err
	}
	maxParallel, err := intVal("MORPHD_MAX_PARALLEL", 1)
	if err != nil {
		return Config{}, err
	}
	resumesPerHour, err := intVal("MORPHD_RESUMES_PER_HOUR", 3)
	if err != nil {
		return Config{}, err
	}
	stallMinutes, err := intVal("MORPHD_STALL_MINUTES", 30)
	if err != nil {
		return Config{}, err
	}
	usageAlert, err := intVal("MORPHD_USAGE_ALERT", 50)
	if err != nil {
		return Config{}, err
	}

	home := getenv("HOME")

	stateDir := lookup("MORPHD_STATE_DIR")
	if stateDir == "" {
		stateDir = home + "/.local/state/morphd"
	}
	projectsDir := lookup("MORPHD_PROJECTS_DIR")
	if projectsDir == "" {
		projectsDir = home + "/projects"
	}
	claudeBin := lookup("MORPHD_CLAUDE_BIN")
	if claudeBin == "" {
		claudeBin = "claude"
	}
	morphBin := lookup("MORPHD_MORPH_BIN")
	if morphBin == "" {
		morphBin = "morph"
	}

	var extraArgs []string
	if raw := lookup("MORPHD_CLAUDE_EXTRA_ARGS"); raw != "" {
		extraArgs = strings.Fields(raw)
		if len(extraArgs) == 0 {
			extraArgs = nil
		}
	}

	return Config{
		Token:             token,
		Port:              port,
		StateDir:          stateDir,
		ProjectsDir:       projectsDir,
		ClaudeBin:         claudeBin,
		MorphBin:          morphBin,
		TGToken:           lookup("TG_BOT_TOKEN"),
		TGChatID:          lookup("TG_CHAT_ID"),
		Model:             lookup("MORPHD_MODEL"),
		ExtraArgs:         extraArgs,
		Defaults:          queue.Caps{ClaudeUSD: claudeUSD, Hours: hours, ExecutorUSD: executorUSD},
		StretchUSD:        stretchUSD,
		MaxParallel:       maxParallel,
		ResumesPerHour:    resumesPerHour,
		StallMinutes:      stallMinutes,
		UsageAlertPercent: usageAlert,
	}, nil
}
