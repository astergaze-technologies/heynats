# Phase 0: Foundation, step by step

The first phase of [ROADMAP.md](ROADMAP.md), broken into small steps that ship one at a time.
Each step is **one branch, one PR, tested, checked, merged and deployed** before the next one starts. Steps don't need to happen on the same day.

## How every step works
1. Branch from `develop`: `fix/...` or `feat/...`.
2. Change **one** thing. Leave anything else you notice for its own step.
3. Write a test that **fails before** the change and **passes after**.
4. Run the checks:
   - Go: `go vet ./... && go test -race ./...`
   - Client (if changed): `cd client && pnpm lint && pnpm build`
5. Manual check in the running app: `docker compose up -d nats`, then `make dev-full`.
6. Open PR → review → merge to `develop` → deploy → watch logs → tick the step below.

**One-time setup:** `cd client && pnpm install` (local `node_modules` is stale) and `brew install golangci-lint` (needed by the pre-push hook).

## Progress

| # | Step | Branch | Size | Status |
|---|---|---|---|---|
| 1 | Stop live subscriptions crashing the server | `fix/sse-subscription-crash` | S | ☐ |
| 2 | CI workflow | `ci/checks` | S | ☐ |
| 3 | Close NATS connections properly | `fix/connection-leaks` | S | ☐ |
| 4 | No network calls under the global lock | `fix/connection-store-lock` | S | ☐ |
| 5 | `/connect` with new credentials replaces the connection | `fix/connect-new-credentials` | S | ☐ |
| 6 | Small backend bug pack | `fix/backend-bugs` | S | ☐ |
| 7 | KV fixes (UI + API) | `fix/kv` | S | ☐ |
| 8 | Stream fixes | `fix/streams` | S | ☐ |
| 9 | Config, structured logs, graceful shutdown | `feat/config-logging` | M | ☐ |
| 10 | Security basics | `fix/security-basics` | M | ☐ |
| 11 | Dev NATS config with a system account | `chore/dev-nats-config` | S | ☐ |

Order = severity first (crash), then the safety net (CI), then leaks and slowness, then the bugs users see, then hardening.

---

## Step 1: Stop live subscriptions crashing the server
**Commit:** `fix(api): stop live subscriptions from crashing the server`

**Problem.** Any user can take down the whole server:
- `internal/api/subscribe.go`: when `max_messages` is reached, the NATS callback calls `close(done)`, then calls it again on every later message, and the deferred cleanup calls it once more. The result is a "close of closed channel" panic.
- The cleanup also closes `msgChan` right after `Unsubscribe()`, but a callback that is still running can send into it. The result is a "send on closed channel" panic.
- Both panics happen on the NATS callback goroutine, which gin's recovery doesn't cover, so the **process exits**.
- `messageCount` is read and written from two goroutines without synchronisation (a data race).
- `internal/api/stream.go` (stream live tail) has the same close-after-unsubscribe pattern and uses the deprecated `CloseNotify`.

**Fix** (no change to the response format; the frontend is untouched):
- New `internal/api/sse.go` with one helper used by both endpoints:
  - Subscribe with `ChanSubscribe` / `ChanQueueSubscribe` into a buffered channel (256). nats.go fills it; when it's full, nats.go drops the message and reports a slow consumer. **The channel is never closed.**
  - A single loop in the request handler reads messages, writes the SSE event, counts, and returns after `max_messages`, sending the same `completed` event as before.
  - `defer sub.Unsubscribe()`. Stops when the client disconnects (`Request.Context().Done()`).
- `subscribe.go` and `stream.go` keep their parameter parsing and call the helper. Remove the callback, `done`, `msgChan` and the counter.

**Not in this step:** keepalive pings, removing the CORS headers (step 10), base64 payloads, multi-value headers.

**Tests:**
- New `internal/testutil/natstest`: starts a real NATS server inside the test (`github.com/nats-io/nats-server/v2`, imported only from test code so it never enters the binary).
- New `internal/api/subscribe_test.go`:
  1. Subscribe with `max_messages=3`, publish 10 → exactly 3 messages, then `completed`, then the stream ends; the server still answers.
  2. Client disconnects while messages are flowing, repeated 50 times → no panic.
  3. Same as 2 for the stream live tail.
- Run `go test -race -count=20 ./internal/api/ -run Subscribe`. First show test 1 panics on the old code.

**Manual check:** subscribe to `logs.test` with max messages 3 and publish 10. You should see 3 messages and the subscription completes. Close the tab mid-stream. Do the same on the stream live tail. The server keeps running.

**Done when:** tests pass under `-race -count=20`, the manual check passes, and the diff touches only `sse.go`, `subscribe.go`, `stream.go`, `natstest/`, `subscribe_test.go`, `go.mod` and `go.sum`.

---

## Step 2: CI workflow
**Commit:** `ci: run lint, tests and build on pull requests`

**Problem.** CI only builds the release image. Nothing checks a PR.

**Change:**
- `.github/workflows/ci.yml`, triggered on pull requests to `develop` and `main`:
  - **go job:** setup-go from `go.mod`, `go vet ./...`, `go test -race ./...`, golangci-lint.
  - **web job:** pnpm via corepack, `pnpm install --frozen-lockfile`, `pnpm lint`, `pnpm build`.
- `.golangci.yml` (v2 format), minimal: `govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`. Use `only-new-issues: true` in the action so existing code doesn't block PRs. Tightening the rules is its own later step.

**Note:** `pnpm typecheck` runs `tsc --noEmit` against the root `tsconfig.json`, which has `"files": []`, so it checks nothing. `pnpm build` already runs `tsc -b`, which checks properly. Either fix the script to `tsc -b --noEmit` here, or rely on `build`.

**Done when:** this PR itself shows both jobs green.

---

## Step 3: Close NATS connections properly
**Commit:** `fix(api): always close NATS connections when they are replaced or removed`

**Problem:**
- `Disconnect()` in `internal/pkg/nats_client.go` only calls `Close()` when the connection is connected. A connection that is reconnecting (with unlimited retries) is never closed, so it keeps redialling forever, even after logout or idle cleanup.
- `GetOrReconnect` in `internal/api/connection.go` replaces a dead connection with a new one and never closes the old one.

**Fix:** `Disconnect()` always calls `Close()`. Close the old connection when replacing it.

**Tests:**
- Add `Stop()` / `Restart()` (same port) to `natstest`.
- Stop the server → the connection is reconnecting → `RemoveConnection` → the connection is closed.
- Reconnect path → the old connection is closed.

**Manual check:** connect, stop NATS (`docker compose stop nats`), log out, start NATS again. The NATS monitoring page (`:8222/connz`) shows no leftover HeyNATS connection.

---

## Step 4: No network calls while holding the global lock
**Commit:** `fix(api): stop pinging NATS under the connection store lock`

**Problem.** Every API request goes through `GetOrReconnect`. It holds one global mutex while it flushes to NATS (up to 2s) and, if that fails, dials again (up to 10s). The UI polls several endpoints every 5–10s, so all users wait on each other, and one slow server stalls everyone.

**Fix:**
- Decide using the connection's status instead of a ping. If the connection is connected or reconnecting, return it immediately; nats.go already reconnects on its own, and handlers already answer 503 when it isn't connected.
- Only a connection that is permanently **closed** gets redialled, and the dial happens **outside** the lock.
- If two requests race to redial, keep the first new connection and close the other.

**Tests:**
- Stop the server → `GetOrReconnect` returns in under 100ms → restart → the same connection recovers.
- Closed connection → redialled once; a second call reuses it.

**Manual check:** with two browser tabs open, stop NATS. Both tabs show disconnected quickly and recover when NATS is back.

---

## Step 5: `/connect` with new credentials replaces the connection
**Commit:** `fix(api): connect with new credentials instead of reusing the old session`

**Problem.** If the browser already has a session cookie, `/connect` silently keeps the old connection and ignores the new host and credentials. Before even reaching that point, it may spend up to 10s redialling the old server.

**Fix:**
- Remove the `Handle()` middleware from `/connect`.
- Add `GetConfig(id)` to the connection store.
- Same credentials as stored → keep the connection. Different credentials → remove (and close) the old connection and dial the new one.

**Tests:** run two test NATS servers, A and B. Connect to A, then to B with the same cookie. `/status` shows B, the store holds one connection, and the connection to A is closed. Connecting to B again keeps that one connection.

**Manual check:** connect to one server, go back to login, connect to another. The dashboard shows the second server.

---

## Step 6: Small backend bug pack
**Commit:** `fix(api): small backend fixes` (split into two PRs if review is easier)

| Bug | Where | Fix | Test |
|---|---|---|---|
| Unchecked type casts on the account info reply can panic | `pkg/nats_client.go` `GetAccountInfo` | Check the reply shape before using it | Unit test with an error reply |
| "Expires: never" never shows (number compared as an int) | `pkg/nats_client.go` `infoAction` | Compare as a JSON number | Unit test |
| KV history above 255 wraps around (`uint8`) | `api/kv.go` `CreateBucket` | Allow 0–64, otherwise 400. Return 409 only for "already exists" | 300 → 400; duplicate → 409 |
| Consumers response uses the key `streams` | `api/stream.go` `ListConsumers` | Rename the key to `consumers` (no frontend uses it yet) | Response key test |
| Empty payloads rejected | `api/publish.go`, `api/subscribe.go` (`binding:"required"` on `data`) | Allow empty data | Publish `{subject}` only → delivered |
| Every request error returns 408 | `api/publish.go` `requestReply` | Timeout → 504, no responders → 503, other → 502 | One test per case |
| Custom reply subject ignored | `api/publish.go` | If set: subscribe to it, publish with it, wait for one reply | Responder sees the custom reply subject |
| Version "2.11" (no patch) fails the check | `util/util.go` | Missing minor/patch parts count as 0; compile the regexp once | Update the tests that currently expect the bug |

**Manual check:** publish an empty message; send a request with nobody listening (expect a "no responders" error, not a timeout); the dashboard account panel still loads.

---

## Step 7: KV fixes
**Commit:** `fix(kv): bucket creation, keys with "/" and full key list`

| Bug | Fix |
|---|---|
| Creating a bucket returns 404: the client calls `/kv/bucket`, the server route is `/kv/buckets` | Fix the path in `client/src/lib/api.ts` |
| Bucket and key names aren't URL-encoded | `encodeURIComponent` on every path segment in `api.ts` |
| Keys containing `/` (valid in KV) break routing | Router: `UseRawPath = true`, `UnescapePathValues = true` |
| Only the first 20 keys show (the server defaults to 20, the client never asks for more) | Server returns all keys when no `pageSize` is given. This is interim; the real paging comes with the KV workbench |
| Storing a JSON object/number fails with 500 | Store non-string JSON values as raw JSON |
| A proxy error page (502) crashes response parsing in the client | Guard `response.json()`; fix header merging in `apiRequest` |

**Tests:**
- Go: create bucket, put/get key `app/db.url`, list more than 20 keys, put an object value.
- Manual: create a bucket in the UI, add key `a/b`, edit it, delete it.

---

## Step 8: Stream fixes
**Commit:** `fix(streams): search paging and complete stream creation`

| Bug | Fix |
|---|---|
| Search paging is wrong: the client sends `offset = page × limit`, and the search path multiplies by `limit` again (`pkg/stream.go`) | Treat `offset` as a count of matching messages, the same unit as plain paging |
| Creating a stream silently drops settings: `allow_msg_ttl` is accepted but never applied; `max_msgs_per_subject`, `max_msg_size`, `duplicate_window`, `compression` are sent by the UI and discarded | Add the fields to `pkg.StreamConfig` and apply them |
| `ListConsumers` dereferences JetStream without a nil check | Add the check |

**Not in this step:** pages that skip over deleted messages (fixed properly by cursor paging in the JetStream migration).

**Tests:**
- 40 messages alternating "match"/"other": search with offset 5, limit 3 → sequences 11, 13, 15.
- Create a stream with all fields → stream info shows them.

**Manual check:** search a stream and page forward. Create a stream with a duplicate window and compression, then check its config.

---

## Step 9: Config, structured logs, graceful shutdown
**Commit:** `feat: configurable server with structured logs and graceful shutdown`

**Problem:**
- The port is hardcoded (`main.go`: `router.Run(":5000")`), and the `router.Run` error is ignored.
- gin runs in debug mode.
- Shutdown calls `os.Exit(0)` from the signal handler, which cuts off open requests.
- Logs are unstructured `log.Printf`.

**Change** (keep it small; a full config file comes later):
- Flags and env vars: `HEYNATS_HTTP_ADDR` (default `:5000`, so nothing changes for existing users), `HEYNATS_LOG_LEVEL`, `HEYNATS_LOG_FORMAT` (text/json).
- `log/slog` replaces `log.Printf`. gin runs in release mode unless the log level is debug.
- `http.Server` with timeouts (`ReadHeaderTimeout`, `IdleTimeout`; no global write timeout, because SSE responses stay open).
- `signal.NotifyContext` → `srv.Shutdown` (15s) → close NATS connections → exit. Remove `os.Exit`.

**Tests:** config parsing (defaults, env override).

**Manual check:** run with `HEYNATS_HTTP_ADDR=:5050`. Press Ctrl+C during an active subscription: the process exits cleanly within 15s.

---

## Step 10: Security basics
**Commit:** `fix(security): close CORS and cross-site holes, run container as non-root`

| Issue | Fix |
|---|---|
| Live endpoints send `Access-Control-Allow-Origin: *` (`subscribe.go`, `stream.go`) | Remove the CORS headers (the UI is same-origin) |
| `/api/nats/test` and `/connect` can be triggered by any website the user visits, making the server dial any host (SSRF) | Reject state-changing requests the browser marks as cross-site (`Sec-Fetch-Site`: `cross-site` / `same-site`) |
| No request body size limit | `http.MaxBytesReader`, 1 MiB default |
| The session cookie trusts `X-Forwarded-Proto` from anyone | Trust it only from configured proxies |
| The Docker image runs as root | Add a non-root user in the `Dockerfile` |

**Tests:**
- A cross-site POST gets 403; a same-origin POST gets 200.
- A body over the limit gets 413.

**Manual check:** the UI works in `make dev-full` (Vite proxy) and in the Docker image.

---

## Step 11: Dev NATS config with a system account
**Commit:** `chore(dev): proper system account in dev nats.conf`

**Problem.** In `nats.conf` the `SYS` account is an ordinary account with JetStream enabled. It isn't declared as the system account, so every monitoring feature in Phase 1 can't work in dev.

**Change:**
- Accounts: `APP` (JetStream enabled; users `admin` and `mukezhz`, adding `_INBOX.>` to `mukezhz`'s subscribe permissions) and `SYS` (user `sys`, no JetStream).
- Add `system_account: SYS`.

**Note:** existing dev JetStream data lives under the old SYS account, so wipe `nats_data/` once. Also update the README credentials section, which currently prints a password in plain text.

**Manual check:** connecting as `sys` can list servers (`nats --user sys --password … server list`). Connecting as `admin` sees and creates streams.

---

## After step 11: the bigger refactors
Still one PR at a time. Each gets its own short plan doc when we reach it.
1. New connection manager (pooled connections, health from NATS events, idle cleanup that respects live views).
2. Several clusters at once with URLs `/c/:ctx/...` and server-side encrypted saved connections.
3. Move to the `jetstream` package, one area per PR: streams (with cursor paging), then KV, then consumers.
4. Live updates over one WebSocket per tab (replaces SSE).
5. API contract (OpenAPI) + generated client.
6. App shell redesign (nav, inspector, dock, ⌘K, environment band, production safety).
7. Releases with goreleaser (binaries + Homebrew) and a distroless image.

Then [Phase 1](ROADMAP.md#phase-1-observe--debug), one screen at a time.
