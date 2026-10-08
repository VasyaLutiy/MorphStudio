#!/usr/bin/env bash
# Claude Code hook → Telegram. Receives the hook's JSON on stdin; posts a one-line event.
# Wired in .claude/settings.json for Stop (the session stopped: finished or waiting for
# input) and Notification (permission prompt or idle notice). Needs tools/tg.sh.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
SESSION="$(printf %s "MorphStudio" | tr ".:" "__")"
EVENT="${1:-event}"
INPUT="$(cat)"
LAST="$(printf '%s' "$INPUT" | python3 -c '
import json,sys
try:
    d=json.load(sys.stdin)
except Exception:
    print(""); sys.exit()
msg=d.get("message") or d.get("title") or ""
tp=d.get("transcript_path")
if not msg and tp:
    try:
        last=""
        for line in open(tp, encoding="utf-8"):
            try: o=json.loads(line)
            except Exception: continue
            m=o.get("message") or {}
            if o.get("type")=="assistant" and isinstance(m.get("content"),list):
                t=" ".join(b.get("text","") for b in m["content"] if b.get("type")=="text").strip()
                if t: last=t
        msg=last
    except Exception:
        pass
print(msg[:1500])
' 2>/dev/null)"
# A Stop while a background agent still works is not a stop: the main turn ended by handing
# the work to an agent. Skip those; everything else is posted.
if [ "$EVENT" = "stop" ] && printf '%s' "$LAST" | grep -qiE "background agent|waiting for .*agent|agent .*is now working"; then exit 0; fi
# ... and when the tmux screen shows an agent still running (its token counter line), the main
# turn ended only to wait: nothing to post.
if [ "$EVENT" = "stop" ] && tmux capture-pane -pt "$SESSION" 2>/dev/null | grep -qE "tokens\s*$|Waiting for [0-9]+ background agent"; then exit 0; fi
# "Claude is waiting for your input" is the idle notice of an empty queue, not an event.
if [ "$EVENT" = "notification" ] && printf '%s' "$LAST" | grep -qi "waiting for your input"; then exit 0; fi
if [ "$EVENT" = "stop" ]; then "$HERE/tg.sh" idle "Session stopped" "${LAST:-(no text)}"
else "$HERE/tg.sh" ask "Session needs attention" "${LAST:-(no text)}"; fi
exit 0
