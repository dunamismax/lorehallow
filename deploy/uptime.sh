#!/usr/bin/env bash
# Uptime check for the Uptime workflow (.github/workflows/uptime.yml, ADR
# 0013). Checks every endpoint, retrying once a minute for five
# minutes before calling it down, and remembers the last state in
# .uptime-state so it alerts once when something goes down and once when it
# comes back, not every five minutes.
#
#   deploy/uptime.sh                 check, alert on a change of state
#   deploy/uptime.sh --test-alert    send one test notification and exit
#
# Alerts go to ntfy (https://ntfy.sh/$NTFY_TOPIC) when NTFY_TOPIC is set.
# The run that first finds something down exits 1, so the workflow fails
# and GitHub emails the owner too (once per outage, like the ntfy alert).
#
# Locally: UPTIME_RETRY_SECONDS=0 deploy/uptime.sh
set -euo pipefail

STATE_FILE="${UPTIME_STATE_FILE:-.uptime-state}"
RETRIES="${UPTIME_RETRIES:-6}"
RETRY_SECONDS="${UPTIME_RETRY_SECONDS:-60}"
NTFY_SERVER="${NTFY_SERVER:-https://ntfy.sh}"

# name|url|text the response must contain
ENDPOINTS=(
  "lorehallow.com|https://lorehallow.com/|lorehallow-web"
  "progress|https://lorehallow.com/progress.json|\"phases\""
  "play.lorehallow.com|https://play.lorehallow.com/healthz|"
)
if [[ -n "${UPTIME_ENDPOINTS:-}" ]]; then # for testing: "name|url|text;name|url|text"
  IFS=';' read -r -a ENDPOINTS <<<"$UPTIME_ENDPOINTS"
fi

notify() { # title, message, priority, tags
  if [[ -z "${NTFY_TOPIC:-}" ]]; then
    echo "uptime: no NTFY_TOPIC set; not sending: $1: $2"
    return 0
  fi
  curl -fsS --max-time 15 -H "Title: $1" -H "Priority: $3" -H "Tags: $4" \
    -H "Click: https://github.com/dunamismax/lorehallow/actions/workflows/uptime.yml" \
    -d "$2" "$NTFY_SERVER/$NTFY_TOPIC" >/dev/null && echo "uptime: sent: $1"
}

if [[ "${1:-}" == "--test-alert" ]]; then
  notify "Lorehallow uptime: test" "Alerts from the Uptime workflow reach this device." default white_check_mark
  exit 0
fi

check_one() { # url, text: 0 when it answers 2xx within 15 s and contains text
  local body
  body="$(curl -fsS --max-time 15 -A "lorehallow-uptime" "$1" 2>/dev/null)" || return 1
  [[ -z "$2" ]] || grep -qF -- "$2" <<<"$body"
}

down=()
for attempt in $(seq 1 "$RETRIES"); do
  down=()
  for e in "${ENDPOINTS[@]}"; do
    IFS='|' read -r name url text <<<"$e"
    check_one "$url" "$text" || down+=("$name")
  done
  if [[ ${#down[@]} -eq 0 ]]; then
    break
  fi
  echo "uptime: attempt $attempt/$RETRIES: down: ${down[*]}"
  if [[ "$attempt" -lt "$RETRIES" ]]; then
    sleep "$RETRY_SECONDS"
  fi
done

previous="$(cat "$STATE_FILE" 2>/dev/null || echo up)"
if [[ ${#down[@]} -eq 0 ]]; then
  echo up >"$STATE_FILE"
  echo "uptime: all up"
  if [[ "$previous" != up ]]; then
    notify "Lorehallow is back up" "Everything answers again (was down: ${previous#down: })." default white_check_mark
  fi
  exit 0
fi

now="down: ${down[*]}"
echo "$now" >"$STATE_FILE"
echo "uptime: $now"
if [[ "$previous" == "$now" ]]; then
  # Already reported; stay quiet until it changes.
  echo "::warning title=Still down::${down[*]}"
  exit 0
fi
minutes=$(((RETRIES - 1) * RETRY_SECONDS / 60))
notify "Lorehallow is down" "Not answering for at least $minutes minutes: ${down[*]}." high rotating_light
exit 1
