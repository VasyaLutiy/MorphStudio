#!/usr/bin/env bash
# The command tmux runs for the autonomous session: claude with the prompt of
# tools/vps-prompt.txt, then keep the pane alive so the log and the last screen stay readable.
set -u
REPO="$(cd "$(dirname "$0")/.." && pwd)"
export PATH="$HOME/.local/bin:$PATH" TERM="${TERM:-xterm-256color}"
# Claude auto-memory off in every phase session (operator 08.10, "100%"): the repo is the only memory of a phase.
export CLAUDE_CODE_DISABLE_AUTO_MEMORY=1
cd "$REPO"
claude --dangerously-skip-permissions "$(cat "$REPO/tools/vps-prompt.txt")"
echo; echo "[claude exited with $?]"; sleep 86400
