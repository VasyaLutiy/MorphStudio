package daemon

import (
	"context"

	"morphstudio/claude"
	"morphstudio/eventlog"
)

func init() {
	startPump = func(d *Daemon, ctx context.Context, project string, p claude.Process, l *eventlog.Log) {}
}

func (d *Daemon) pump(ctx context.Context, project string, p claude.Process, l *eventlog.Log) {}

func (d *Daemon) Wait(project string) {}
