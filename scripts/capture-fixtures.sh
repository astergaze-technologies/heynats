#!/usr/bin/env bash
# Saves HeyNATS API responses as MSW fixtures. Needs the app on :5000 and a NATS
# server (e.g. the 3-node cluster) with some streams and KV buckets in it.
set -euo pipefail

APP=${APP:-http://localhost:5000}
NATS_HOST=${NATS_HOST:-localhost}
NATS_PORT=${NATS_PORT:-4222}
NATS_USER=${NATS_USER:-admin}
NATS_PASS=${NATS_PASS:?set NATS_PASS to the dev password}
OUT=client/src/mocks/fixtures
JAR=$(mktemp)
trap 'rm -f "$JAR"' EXIT

curl -sf -c "$JAR" -H "Origin: $APP" -H 'Content-Type: application/json' \
  -d "$(jq -n --arg h "$NATS_HOST" --arg p "$NATS_PORT" --arg u "$NATS_USER" --arg pw "$NATS_PASS" \
    '{host:$h,port:$p,username:$u,password:$pw}')" \
  "$APP/api/nats/connect" >/dev/null

get() { curl -sf -b "$JAR" "$APP/api/nats/$1" | jq . >"$OUT/$2.json"; echo "$OUT/$2.json"; }
get status status
get info info
get account account
get capabilities capabilities
get streams streams
get kv/buckets kv-buckets

curl -sf -b "$JAR" -H "Origin: $APP" -X POST "$APP/api/nats/disconnect" >/dev/null
