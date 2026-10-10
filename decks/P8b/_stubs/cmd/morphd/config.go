package main

import (
	"errors"

	"morphstudio/queue"
)

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

func ParseDotenv(text string) map[string]string { return nil }

func Load(getenv func(string) string, dotenv map[string]string) (Config, error) {
	return Config{}, errors.New("stub Load")
}

func UUIDv4(b [16]byte) string { return "stub UUIDv4" }
