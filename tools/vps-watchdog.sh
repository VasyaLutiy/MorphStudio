#!/usr/bin/env bash
# Watchdog for the autonomous tmux session of MorphStudio (cron, every 10 minutes, as the session's user):
#   */10 * * * * <repo>/tools/vps-watchdog.sh
# If the pane shows a usage/rate limit message and the screen has not changed since the last check, nudge the session to
# continue and tell the operator. If the session is gone, tell the operator once per hour. A phase ended with
# ~/.morph-phase-done gets a fresh session. Never starts a paid run itself.
set -u
export PATH="$HOME/.local/bin:$PATH" TERM=xterm-256color
HERE="$(cd "$(dirname "$0")" && pwd)"
SESSION="$(printf %s "MorphStudio" | tr ".:" "__")"
TG="$HERE/tg.sh"
STATE="$HOME/.morph-watchdog/$SESSION"; mkdir -p "$STATE"
if ! tmux has-session -t "$SESSION" 2>/dev/null; then
  if [ ! -f "$STATE/dead" ] || [ $(( $(date +%s) - $(stat -c %Y "$STATE/dead") )) -gt 3600 ]; then
    "$TG" watchdog "tmux session '$SESSION' is not running" "start it: tools/vps-start.sh"; touch "$STATE/dead"; fi
  exit 0
fi
rm -f "$STATE/dead"
# One phase, one session: the session ends a phase by touching ~/.morph-phase-done → restart it fresh.
if [ -f "$HOME/.morph-phase-done" ]; then
  rm -f "$HOME/.morph-phase-done" "$STATE/sum" "$STATE/same" "$STATE/stalled"
  tmux kill-session -t "$SESSION"; sleep 2
  "$HERE/vps-start.sh" start >/dev/null
  "$TG" start "Fresh session for the next phase" "one phase, one session"
  exit 0
fi
# The session waits for the operator (~/.morph-wait-operator): never nudge it.
[ -f "$HOME/.morph-wait-operator" ] && exit 0
PANE="$(tmux capture-pane -pt "$SESSION" -S -40 2>/dev/null)"
SUM="$(printf '%s' "$PANE" | md5sum | cut -c1-12)"
PREV="$(cat "$STATE/sum" 2>/dev/null || true)"; printf '%s' "$SUM" > "$STATE/sum"
if printf '%s' "$PANE" | grep -qiE "limit reached|usage limit|rate limit|out of extra usage|resets at|try again"; then
  if [ "$SUM" = "$PREV" ]; then
    N=$(( $(cat "$STATE/nudges" 2>/dev/null || echo 0) + 1 )); printf '%s' "$N" > "$STATE/nudges"
    if [ "$N" -le 12 ]; then
      tmux send-keys -t "$SESSION" C-u
      tmux send-keys -t "$SESSION" -l "Continue by docs/AUTONOMY.md from where you stopped; the usage window has reset. Post the current state to tools/tg.sh first."
      sleep 2; tmux send-keys -t "$SESSION" C-m
      "$TG" watchdog "Limit on screen, nudged the session" "nudge $N of 12"
    fi
  fi
else
  rm -f "$STATE/nudges"
  # A stall: the screen unchanged for three checks (~30 min), no morph run in flight and no background agent on screen.
  S=0
  if [ "$SUM" = "$PREV" ]; then
    S=$(( $(cat "$STATE/same" 2>/dev/null || echo 0) + 1 )); printf '%s' "$S" > "$STATE/same"
  else
    printf '0' > "$STATE/same"; rm -f "$STATE/stalled"
  fi
  # A run in flight = the node process of the binary copy itself; match the interpreter, not a shell loop that names it.
  if [ "$S" -ge 3 ] && ! pgrep -f '^[^ ]*node [^ ]*dist/cli\.js run' >/dev/null && [ ! -f "$STATE/stalled" ] &&
     { [ "$S" -ge 6 ] || ! printf '%s' "$PANE" | grep -qiE "waiting for .*agent|background agent|tokens$"; }; then
    tmux send-keys -t "$SESSION" C-u
    tmux send-keys -t "$SESSION" -l "Nothing has happened on this screen for 30 minutes and no run is in flight. Check the state of your agents and the run branch, then continue by docs/AUTONOMY.md from where you are; post the current state to tools/tg.sh first."
    sleep 2; tmux send-keys -t "$SESSION" C-m
    touch "$STATE/stalled"
    "$TG" watchdog "Stall, nudged the session" "screen unchanged 30 min · no run in flight"
  fi
fi
exit 0
