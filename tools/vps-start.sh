#!/usr/bin/env bash
# Start (or re-attach to) the autonomous session of MorphStudio inside tmux.
#   tools/vps-start.sh            start the session if it is not running
#   tools/vps-start.sh attach     attach to it (detach with Ctrl-b d)
#   tools/vps-start.sh log        follow the session log
# Runs as a non-root user (claude refuses --dangerously-skip-permissions under root).
# The session survives an SSH drop. Log: $HOME/morph-logs/MorphStudio-<date>.log
set -eu
REPO="$(cd "$(dirname "$0")/.." && pwd)"
SESSION="$(printf %s "MorphStudio" | tr ".:" "__")"
LOGDIR=$HOME/morph-logs; mkdir -p "$LOGDIR"
LOG="$LOGDIR/MorphStudio-$(date +%Y%m%d-%H%M).log"
export PATH="$HOME/.local/bin:$PATH" TERM="${TERM:-xterm-256color}"
case "${1:-start}" in
  attach) exec tmux attach -t "$SESSION" ;;
  log) exec tail -n 200 -f "$(ls -t "$LOGDIR"/MorphStudio-*.log | head -1)" ;;
  start)
    if tmux has-session -t "$SESSION" 2>/dev/null; then echo "session '$SESSION' already runs: tools/vps-start.sh attach"; exit 0; fi
    rm -f "$HOME/.morph-wait-operator" "$HOME/.morph-phase-done"
    # The session reads "State at handoff" from main: start only on the latest origin/main (operator 08.10).
    if ! git -C "$REPO" pull -q --ff-only; then
      "$REPO/tools/tg.sh" stop "Session not started: git pull --ff-only failed" "$REPO: fix main by hand, then start again"
      echo "git pull --ff-only failed in $REPO; not started" >&2; exit 1
    fi
    tmux new-session -d -s "$SESSION" -c "$REPO" "$REPO/tools/vps-session.sh"
    tmux pipe-pane -t "$SESSION" -o "cat >> '$LOG'"
    "$REPO/tools/tg.sh" start "MorphStudio session started on $(hostname)" "log $LOG"
    echo "started tmux session '$SESSION'; attach: tools/vps-start.sh attach; log: $LOG"
    ;;
  *) echo "usage: $0 [start|attach|log]"; exit 2 ;;
esac
