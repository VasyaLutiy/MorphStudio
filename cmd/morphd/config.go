package main

import (
	"fmt"
	"strconv"
	"strings"

	"morphstudio/queue"
)

// Config holds the resolved daemon configuration.
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

// ParseDotenv reads a .env document into a key-value map. Lines are trimmed;
// empty lines and lines beginning with "#" are skipped; an "export " prefix is
// removed; the first "=" splits key and value; the value is trimmed and, when
// wrapped in matching single or double quotes, unquoted.
func ParseDotenv(text string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
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

// Load resolves the configuration. Each key is read from getenv, falling back
// to dotenv, then to its default.
func Load(getenv func(string) string, dotenv map[string]string) (Config, error) {
	get := func(key string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return dotenv[key]
	}
	home := getenv("HOME")

	token := get("MORPHD_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("config: MORPHD_TOKEN is required")
	}

	port, err := getInt(get, "MORPHD_PORT", 7080)
	if err != nil {
		return Config{}, err
	}
	claudeUSD, err := getFloat(get, "MORPHD_CLAUDE_USD", 30)
	if err != nil {
		return Config{}, err
	}
	hours, err := getFloat(get, "MORPHD_HOURS", 3)
	if err != nil {
		return Config{}, err
	}
	executorUSD, err := getFloat(get, "MORPHD_EXECUTOR_USD", 5)
	if err != nil {
		return Config{}, err
	}
	stretch, err := getFloat(get, "MORPHD_STRETCH_USD", 30)
	if err != nil {
		return Config{}, err
	}
	maxParallel, err := getInt(get, "MORPHD_MAX_PARALLEL", 1)
	if err != nil {
		return Config{}, err
	}
	resumesPerHour, err := getInt(get, "MORPHD_RESUMES_PER_HOUR", 3)
	if err != nil {
		return Config{}, err
	}
	stallMinutes, err := getInt(get, "MORPHD_STALL_MINUTES", 30)
	if err != nil {
		return Config{}, err
	}
	usageAlert, err := getInt(get, "MORPHD_USAGE_ALERT", 50)
	if err != nil {
		return Config{}, err
	}

	var extraArgs []string
	if raw := get("MORPHD_CLAUDE_EXTRA_ARGS"); raw != "" {
		extraArgs = strings.Fields(raw)
	}

	cfg := Config{
		Token:             token,
		Port:              port,
		StateDir:          getStr(get, "MORPHD_STATE_DIR", home+"/.local/state/morphd"),
		ProjectsDir:       getStr(get, "MORPHD_PROJECTS_DIR", home+"/projects"),
		ClaudeBin:         getStr(get, "MORPHD_CLAUDE_BIN", "claude"),
		MorphBin:          getStr(get, "MORPHD_MORPH_BIN", "morph"),
		TGToken:           getStr(get, "TG_BOT_TOKEN", ""),
		TGChatID:          getStr(get, "TG_CHAT_ID", ""),
		Model:             getStr(get, "MORPHD_MODEL", ""),
		ExtraArgs:         extraArgs,
		StretchUSD:        stretch,
		MaxParallel:       maxParallel,
		ResumesPerHour:    resumesPerHour,
		StallMinutes:      stallMinutes,
		UsageAlertPercent: usageAlert,
		Defaults: queue.Caps{
			ClaudeUSD:   claudeUSD,
			Hours:       hours,
			ExecutorUSD: executorUSD,
		},
	}
	return cfg, nil
}

func getStr(get func(string) string, key, def string) string {
	if v := get(key); v != "" {
		return v
	}
	return def
}

func getInt(get func(string) string, key string, def int) (int, error) {
	v := get(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s: not a number: %s", key, v)
	}
	return n, nil
}

func getFloat(get func(string) string, key string, def float64) (float64, error) {
	v := get(key)
	if v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("config: %s: not a number: %s", key, v)
	}
	return f, nil
}

// UUIDv4 renders a version 4 UUID from the given 16 bytes: the version nibble
// is forced in byte 6, the variant bits in byte 8, everything else is
// preserved. The result is lower-case hex in the groups 8-4-4-4-12.
func UUIDv4(b [16]byte) string {
	b[6] = 0x40 | (b[6] & 0x0f)
	b[8] = 0x80 | (b[8] & 0x3f)

	const hex = "0123456789abcdef"
	out := make([]byte, 0, 36)
	for i := 0; i < 16; i++ {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out = append(out, '-')
		}
		out = append(out, hex[b[i]>>4], hex[b[i]&0x0f])
	}
	return string(out)
}
