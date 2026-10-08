#!/usr/bin/env bash
# Post one message to the operator's Telegram channel. Reads TG_BOT_TOKEN and TG_CHAT_ID
# from $MORPH_TG_ENV (default $HOME/.config/morph/tg.env, mode 600, written by the operator).
# Usage:
#   tools/tg.sh <kind> "<headline>" ["<numbers>"]   a milestone: icon, bold headline, numbers below
#   tools/tg.sh "text"   or   some-command | tools/tg.sh   plain text (first word not a kind)
# Kinds: start 🚀  gate 🚦  run 🏁  fail ❌  merge 🔀  stop 🛑  debt 💸  smoke 🧪  end 🎉
#        watchdog 🐕  idle 💤  ask 🔔  info 💬
# Never prints the token; silent no-op (exit 0) when the env file is absent, so a missing
# channel never breaks a phase.
set -u
ENV_FILE="${MORPH_TG_ENV:-$HOME/.config/morph/tg.env}"
[ -r "$ENV_FILE" ] || exit 0
# shellcheck disable=SC1090
. "$ENV_FILE"
[ -n "${TG_BOT_TOKEN:-}" ] && [ -n "${TG_CHAT_ID:-}" ] || exit 0
esc() { printf '%s' "$1" | sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g'; }
case "${1:-}" in
  start) ICON="🚀" ;; gate) ICON="🚦" ;; run) ICON="🏁" ;; fail) ICON="❌" ;;
  merge) ICON="🔀" ;; stop) ICON="🛑" ;; debt) ICON="💸" ;; smoke) ICON="🧪" ;;
  end) ICON="🎉" ;; watchdog) ICON="🐕" ;; idle) ICON="💤" ;; ask) ICON="🔔" ;; info) ICON="💬" ;;
  *) ICON="" ;;
esac
if [ -n "$ICON" ]; then
  HEAD="${2:-}"; NUMS="${3:-}"
  TEXT="${ICON} <b>$(esc "$HEAD")</b>"
  [ -n "$NUMS" ] && TEXT="${TEXT}"$'\n'"$(esc "${NUMS:0:3500}")"
else
  if [ $# -gt 0 ]; then RAW="$*"; else RAW="$(cat)"; fi
  TEXT="$(esc "${RAW:0:3800}")"
fi
curl -sS -m 20 -o /dev/null -X POST "https://api.telegram.org/bot${TG_BOT_TOKEN}/sendMessage" \
  --data-urlencode "chat_id=${TG_CHAT_ID}" --data-urlencode "text=${TEXT}" \
  --data-urlencode "parse_mode=HTML" --data-urlencode "disable_web_page_preview=true" || true
exit 0
