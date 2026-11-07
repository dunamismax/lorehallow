#!/usr/bin/env bash
# Checks the Tailscale credentials the deploy-web workflow uses, before it
# tries to join the tailnet, and says plainly what's wrong. Never prints a
# secret: only its length, its prefix type and stray whitespace.
#
# Needs TS_OAUTH_CLIENT_ID and TS_OAUTH_SECRET. Prints the tags the client
# can create devices with as `tags=...` to $GITHUB_OUTPUT when it can.
set -euo pipefail

API=https://api.tailscale.com/api/v2
id="${TS_OAUTH_CLIENT_ID:-}"
secret="${TS_OAUTH_SECRET:-}"
out="${GITHUB_OUTPUT:-/dev/null}"

shape() { # name, value
  local v="$2" note=""
  [[ "$v" != "${v#"${v%%[![:space:]]*}"}" || "$v" != "${v%"${v##*[![:space:]]}"}" ]] && note=" (has leading or trailing whitespace)"
  case "$v" in
  tskey-client-*) echo "$1: ${#v} chars, an OAuth client secret (tskey-client-)$note" ;;
  tskey-api-*) echo "$1: ${#v} chars, an API ACCESS TOKEN (tskey-api-), not an OAuth secret$note" ;;
  tskey-auth-*) echo "$1: ${#v} chars, an AUTH KEY (tskey-auth-), not an OAuth secret$note" ;;
  tskey-*) echo "$1: ${#v} chars, a tskey of another kind$note" ;;
  *) echo "$1: ${#v} chars, no tskey- prefix$note" ;;
  esac
}
shape TS_OAUTH_CLIENT_ID "$id"
shape TS_OAUTH_SECRET "$secret"

# Trim whitespace, which a paste can add.
id="$(tr -d '[:space:]' <<<"$id")"
secret="$(tr -d '[:space:]' <<<"$secret")"

resp="$(curl -sS -w '\n%{http_code}' -d "client_id=$id" -d "client_secret=$secret" "$API/oauth/token")"
code="$(tail -n1 <<<"$resp")"
body="$(sed '$d' <<<"$resp")"
if [[ "$code" != 200 ]]; then
  echo "::error title=Tailscale rejected the OAuth credentials::HTTP $code: $(jq -r '.message // .error // .' <<<"$body" 2>/dev/null | head -c 300)"
  exit 1
fi
token="$(jq -r .access_token <<<"$body")"
echo "::add-mask::$token"
echo "token: accepted; scopes: $(jq -r '.scope // "(none listed)"' <<<"$body")"

# Which tags can this client put on a new device? Try an ephemeral,
# single-use, five-minute key with no tags, then with the tags the policy
# offers; delete any key that gets made.
try_key() { # tags JSON array
  local r c b
  r="$(curl -sS -w '\n%{http_code}' -H "Authorization: Bearer $token" -H 'Content-Type: application/json' \
    -d "{\"capabilities\":{\"devices\":{\"create\":{\"reusable\":false,\"ephemeral\":true,\"preauthorized\":true,\"tags\":$1}}},\"expirySeconds\":300,\"description\":\"lorehallow deploy credential check\"}" \
    "$API/tailnet/-/keys")"
  c="$(tail -n1 <<<"$r")"
  b="$(sed '$d' <<<"$r")"
  if [[ "$c" == 200 ]]; then
    local kid
    kid="$(jq -r .id <<<"$b")"
    echo "::add-mask::$(jq -r .key <<<"$b")"
    curl -sS -o /dev/null -X DELETE -H "Authorization: Bearer $token" "$API/tailnet/-/keys/$kid" || true
    echo "auth key with tags $1: allowed"
    return 0
  fi
  echo "auth key with tags $1: refused (HTTP $c: $(jq -r '.message // .' <<<"$b" 2>/dev/null | head -c 200))"
  return 1
}

for tags in '["tag:ci"]' '["tag:lorehallow-host"]' '[]'; do
  if try_key "$tags"; then
    echo "tags=$(jq -r 'join(",")' <<<"$tags")" >>"$out"
    exit 0
  fi
done
echo "::error title=The OAuth client can't create devices::Give it the Auth Keys write scope (and a tag, if Tailscale asks for one)."
exit 1
