# Phase 1: Observe & Debug, step by step

The second phase of [ROADMAP.md](ROADMAP.md#phase-1-observe--debug), broken into small steps that ship one at a time.
Same rules as [Phase 0](PHASE-0-FOUNDATION.md): **one branch, one PR, tested, checked, merged and deployed** before the next one starts.

**Goal of the phase:** HeyNATS stops being a single snapshot and becomes the first place to look in an incident: trends, health verdicts, every server/connection/account, and plain-language answers to "why didn't my message arrive?".

**Phase is done when** (from the roadmap):
- On a 3-node test cluster, stopping a node shows in health within seconds (target: ≤ 10s).
- Each common failure scenario is diagnosed correctly: no subscriber, wrong filter, paused consumer, full ack-pending, stream limit, permission denied.
- Features that need a system account or a newer NATS are shown disabled with the reason.

## Before starting
Phase 1 builds on Phase 0. Hard requirements:

| From Phase 0 | Why Phase 1 needs it |
|---|---|
| Steps 1–11 | No crashes or leaks; `natstest`; CI; dev `nats.conf` with a real system account (step 11) |
| New connection manager | The metrics sampler and event collector hang off the pooled connection, not off a browser session |
| `/c/:ctx` URLs | Every new page lives under the cluster prefix and is shareable |
| OpenAPI + generated client | Every new endpoint is defined in the contract; the client never hand-writes a path |

Soft requirements (the step that needs it says so): WebSocket live updates (events, sampling, trace), the app shell (Inspector, Dock, ⌘K), production safety (`OperationPreview`, `DangerConfirm`, `CopyAsMenu`), the `jetstream` package migration.

**Open decision:** the roadmap still allows swapping Phases 1 and 2. This plan assumes Phase 1 goes first.

## How every step works
1. Branch from `develop` without tracking it: `git switch -c feat/... --no-track origin/develop`.
2. Change **one** thing. Leave anything else you notice for its own step.
3. Backend: test against a real NATS server (`natstest`, the 3-node cluster from step 1 where it matters). Every diagnosis rule gets a scenario test that must produce its finding.
4. Frontend: component tests against MSW fixtures (step 2). Build against fixtures first, then wire the real endpoint.
5. Run the checks:
   - Go: `go vet ./... && go test -race ./...`
   - Client: `cd client && pnpm lint && pnpm test && pnpm build`
6. Manual check in the running app: `docker compose -f docker-compose.cluster.yml up -d` (step 1), then `make dev-full`.
7. Open PR → review → merge to `develop` → deploy → watch logs → tick the step below.

**Every new screen** must meet the "screen is done" rule in [UI.md §7](UI.md#7-quality): own URL, survives refresh, filters in the URL, loading/empty/error/not-available states, keyboard + ⌘K, source badge on non-exact data, both themes, accessibility check, tests.

## Progress

| # | Step | Branch | Size | Status |
|---|---|---|---|---|
| 1 | 3-node test cluster (dev + tests) | `chore/test-cluster` | S | ☐ |
| 2 | Frontend test setup + fixtures | `chore/client-tests` | S | ☐ |
| 3 | Monitoring data source + capabilities | `feat/monitor-source` | M | ☐ |
| 4 | Gates and source badges in the UI | `feat/capability-gates` | S | ☐ |
| 5 | Metrics sampler + history API | `feat/metrics-sampler` | M | ☐ |
| 6 | Chart kit | `feat/chart-kit` | S | ☐ |
| 7 | Overview page | `feat/overview` | M | ☐ |
| 8 | Servers page | `feat/servers` | M | ☐ |
| 9 | Diagnostics engine + Health page | `feat/health` | M | ☐ |
| 10 | Connections explorer | `feat/connections-explorer` | M | ☐ |
| 11 | Kick a connection | `feat/connection-kick` | S | ☐ |
| 12 | Accounts view | `feat/accounts` | M | ☐ |
| 13 | Topology graph | `feat/topology` | M | ☐ |
| 14 | JetStream health | `feat/jetstream-health` | M | ☐ |
| 15 | Consumer diagnostics | `feat/consumer-diagnostics` | M | ☐ |
| 16 | Subject explorer | `feat/subject-explorer` | M | ☐ |
| 17 | Traffic sampling + real subject autocomplete | `feat/subject-sampling` | S | ☐ |
| 18 | Event collector + Events feed | `feat/events` | M | ☐ |
| 19 | Dead letters | `feat/dead-letters` | M | ☐ |
| 20 | Message trace (2.11+) | `feat/message-trace` | M | ☐ |
| 21 | "Why didn't my message arrive?" wizard | `feat/delivery-wizard` | L | ☐ |
| 22 | Latency probes | `feat/latency-probes` | M | ☐ |
| 23 | Phase acceptance: failure scenarios + version matrix | `test/phase-1-acceptance` | M | ☐ |

Order = test bed first, then the data layer every screen reads from, then read-only screens (overview → servers → health → connections → accounts → topology → JetStream), then the debugging tools that combine them (diagnostics → subjects → events → trace → wizard → probes), then the proof.

## Where the data comes from
One rule for the whole phase: **system account first, monitoring port second, account scope last.** Every response says which one it used, and the UI shows it.

| Source | How | Sees |
|---|---|---|
| System account | `$SYS.REQ.SERVER.PING.<KIND>` (fan-out to all servers), `$SYS.REQ.SERVER.<id>.<KIND>`, `$SYS.REQ.ACCOUNT.<acc>.<KIND>` | Every server, every account |
| Monitoring port | `GET http://host:8222/<kind>z` per configured URL | Only the servers whose URLs are configured |
| Account scope | `$SYS.REQ.ACCOUNT.PING.CONNZ` / `STATZ`, `$SYS.REQ.USER.INFO`, JetStream API (`$JS.API.*`) | Own account only |

Kinds used in this phase: `VARZ`, `STATZ`, `HEALTHZ`, `CONNZ`, `SUBSZ`, `JSZ`, `ACCOUNTZ`, `ROUTEZ`, `GATEWAYZ`, `LEAFZ`.

---

## Step 1: 3-node test cluster
**Commit:** `chore(dev): 3-node NATS cluster for development and tests`

**Problem.** Every Phase 1 feature is about clusters (topology, replica lag, a node going down), but dev and tests only run one server.

**Change:**
- `docker-compose.cluster.yml`: three `nats:2.12` servers (`n1`–`n3`), routes between them, JetStream on all three, the step-11 system account, monitoring ports `8222`–`8224`. Same accounts/users as `nats.conf`.
- `natstest.StartCluster(t, 3)`: three in-process servers with JetStream clustering, returning URLs plus `Stop(i)` / `Restart(i)` per node.
- `make cluster-up` / `make cluster-down`.

**Tests:** a stream with R3 created on the test cluster reports three replicas; stopping one node leaves the stream with a leader.

**Manual check:** `nats --user sys … server list` shows three servers.

---

## Step 2: Frontend test setup + fixtures
**Commit:** `test(client): vitest, Testing Library and MSW with captured NATS fixtures`

**Problem.** There are no frontend tests, and Phase 1 screens should be built against realistic data before the backend is ready ([UI.md §7](UI.md#7-quality), "mock-first").

**Change** (skip whatever Phase 0 already added):
- `vitest`, `@testing-library/react`, `msw`; `pnpm test` script; add it to CI and the pre-push hook.
- `client/src/mocks/`: MSW handlers plus fixtures captured from the step-1 cluster with a small script (`scripts/capture-fixtures.sh`): varz, statz, healthz, connz (incl. closed), jsz, accountz, routez, subsz. A "big" fixture set: 3 servers, 50 streams, 10k connections.
- `pnpm dev:mock` runs Vite with MSW on, no Go server needed.

**Tests:** one smoke test rendering the current dashboard against fixtures.

**Done when:** `pnpm test` runs in CI.

---

## Step 3: Monitoring data source + capabilities
**Commit:** `feat(monitor): read server, connection and JetStream data from the system account`

**Problem.** The only monitoring today is `GetAccountInfo` (`internal/pkg/nats_client.go`), which sends `$SYS.REQ.ACCOUNT.PING.CONNZ`, takes the first reply, and falls back to `"N/A"` strings. In a cluster it shows one server. Nothing tells the UI what is available.

**Change:**
- New package `internal/monitor`:
  - `Source` interface with three implementations: `sysSource`, `httpSource` (monitoring port), `accountSource`. Picked at connect time, re-checked on reconnect.
  - **Scatter-gather** for `PING` requests: subscribe to an inbox, publish once, collect replies until the expected server count is reached (known from the last round), or 300ms after the last reply, or a 2s hard timeout. Each reply keeps its `server` envelope; errors per server are returned, not swallowed.
  - Typed structs with only the fields we use, plus the raw JSON for the "Raw" tabs. No import of `nats-server` in non-test code.
  - Results carry `source` (`system` / `monitor` / `account`) and `partial: true` when some servers didn't answer.
- Monitoring URLs: optional `monitor_urls` field on the connection profile (one per server). Covered by the Phase 0 cross-site and allowed-targets checks.
- `GET /api/c/:ctx/capabilities`:
  ```json
  { "source": "system", "server_version": "2.12.0", "servers": 3,
    "features": { "connections.kick": {"ok": true},
                  "trace": {"ok": false, "reason": "Requires NATS 2.11"} } }
  ```
  One table in code maps feature → requirement (source + minimum version), so the UI never decides this itself.
- Replace `GetAccountInfo` with the new package; delete the `"N/A"` fallbacks.

**Not in this step:** any new screen.

**Tests:**
- 3-node cluster, sys user: `VARZ` returns 3 servers, `source=system`.
- Stop one node: still returns 2 servers, `partial=true`, within 2s.
- App user (no system account): falls back to account scope; `capabilities` marks system-only features with a reason.
- Monitoring-port source against `natstest` with the HTTP port on.

**Manual check:** connect as `sys`, then as `admin`; `/capabilities` differs and the dashboard still loads for both.

---

## Step 4: Gates and source badges in the UI
**Commit:** `feat(client): show where data comes from and why features are unavailable`

**Change:**
- `useCapabilities()` (keyed by cluster id, refetched on reconnect).
- `Gate feature="trace"`: renders children when available; otherwise a dimmed version with a tooltip/inline reason ("Requires system account"). Nav items use the same gate.
- `SourceBadge`: "via system account" · "via monitoring port" · "account scope" · "sampled 60s" · "inferred" · "partial: 2 of 3 servers".
- Shared formatters in `lib/format/`: bytes, rates (`1.2k msg/s`), counts, durations, relative/absolute time, local/UTC.

**Tests:** gate renders the reason; badge text per source; formatter unit tests (edges: 0, 999, 1000, negatives, NaN).

---

## Step 5: Metrics sampler + history API
**Commit:** `feat(metrics): sample server and JetStream metrics into in-memory history`

**Problem.** The dashboard is a snapshot; there's no "is this new or has it always been like this?".

**Change:**
- New package `internal/metrics`, one `Sampler` per pooled connection (started and stopped by the connection manager):
  - Every 5s: `STATZ`/`VARZ` per server → msgs/bytes in/out rate, connections, subscriptions, slow consumers, CPU, memory.
  - Every 15s: `JSZ` → storage/memory used per server and account; `num_pending` / `num_ack_pending` / `num_redelivered` per consumer (top 500 by pending, to cap memory).
  - Rates come from counter deltas; a negative delta means the server restarted and starts a new baseline (no spikes).
- Storage: two rings per series, 5s resolution for 1h and 1m resolution for 24h. Fixed-size arrays, no allocation per sample. Memory budget logged at startup (target ≤ 20 MB per cluster at 3 servers / 500 consumers).
- `GET /api/c/:ctx/metrics?series=msgs_in,msgs_out&by=server&range=1h` → columnar arrays (`{ t: [...], series: { n1: [...] } }`), ready for uPlot.
- Response includes `history_since`; the UI says "history since 10:42" instead of pretending to have 24h.
- Settings: `HEYNATS_METRICS_INTERVAL` (default 5s), `HEYNATS_METRICS_ENABLED` (default true).

**Not in this step:** Prometheus (Phase 4), persisting history across restarts.

**Tests:** ring wraparound and downsampling; rate with a counter reset; publish 1,000 messages on the test cluster → the rate series shows them; sampler stops when the connection is removed (no goroutine leak, checked with `goleak`).

---

## Step 6: Chart kit
**Commit:** `feat(client): time-series chart, sparkline and time range picker`

**Change:**
- `uPlot` (lazy-loaded). Components: `TimeSeriesChart` (multi-series, synced crosshair via a shared cursor key, legend with icon + text), `Sparkline`, `StatTile` (value + delta + sparkline), `UsageBar` (with 80%/95% markers), `TimeRangePicker` (5m · 15m · 1h · 6h · 24h, kept in `?range=`).
- Colour-blind-safe chart palette added as semantic tokens in `index.css` for both themes.
- Charts respect reduced motion and never animate layout on live updates.

**Tests:** components render from fixtures; empty and "history since" states; range is read from and written to the URL.

---

## Step 7: Overview page
**Commit:** `feat(overview): cluster overview with traffic trends`

**Problem.** `DashboardPage` shows the connected server's client stats and one account's connections, nothing about the cluster or trends.

**Change** (`/c/:ctx`, replaces the current dashboard):
- Stat tiles: servers up / total · connections · msgs in/out per second · JetStream storage used vs limit · streams / consumers.
- Traffic chart with `TimeRangePicker`.
- Slots for "Open findings", "Top lagging consumers" and "Recent events", filled by steps 9, 14 and 18 (hidden until then, not shown empty).
- Every tile has a `SourceBadge`; tiles that need the system account use `Gate`.
- The old `ConnectionsTable` moves to the Connections explorer in step 10.

**Tests:** renders with system / account-scope fixtures; partial-data banner.

**Manual check:** publish a burst (`nats bench pub test --msgs 100000`), see it on the chart within 10s.

---

## Step 8: Servers page
**Commit:** `feat(servers): server list and detail`

**Change:**
- `/c/:ctx/servers`: table (name, cluster, version, uptime, CPU, memory, connections, subscriptions, slow consumers, msgs in/out rate, JetStream on/off, healthz). Version mismatch across servers is highlighted.
- `/c/:ctx/servers/:id`: tabs **Overview** (charts from step 5), **Config** (limits and settings from `VARZ`), **Routes / Gateways / Leafs**, **Raw** (JSON viewer per kind).
- API: `GET /servers`, `GET /servers/:id`, `GET /servers/:id/raw/:kind`.

**Tests:** 3 servers listed; stopped node shows as "not responding" (from step 3's partial result), not missing.

---

## Step 9: Diagnostics engine + Health page
**Commit:** `feat(health): diagnostics engine and cluster health verdict`

**Problem.** No single "is everything OK?" answer.

**Change:**
- New package `internal/diagnose`:
  - `Snapshot` (everything the sources and sampler know right now) → `Rule` (pure function) → `[]Finding`.
  - `Finding`: `rule_id`, `severity` (critical / warning / info), `resource` (e.g. `server:n2`, `consumer:ORDERS/billing`), `title`, `detail`, `evidence` (label, value, source), `fixes` (an operation id + params, a link, or a CLI command to copy), `first_seen`, `last_seen`.
  - Rules registered in one list; each has an id, doc string and a scenario test.
- Initial rules:
  | Rule | Fires when |
  |---|---|
  | `server.unreachable` | A known server missed 2 rounds (≤ 10s) or sent a shutdown event |
  | `server.healthz` | `HEALTHZ` not ok (e.g. JetStream not current) |
  | `server.version_mismatch` | Servers in one cluster run different versions (info) |
  | `server.slow_consumers` | Slow-consumer count increased in the last 5m |
  | `server.connections_near_limit` | Connections > 90% of `max_connections` |
  | `js.meta_no_leader` | JetStream meta group has no leader / lost quorum |
  | `js.storage_near_limit` | Storage > 80% (warning) / 95% (critical) of server or account limit |
- Engine runs on every sampler tick; findings are kept with `first_seen` so a flapping issue isn't "new" every 5s.
- `/c/:ctx/health`: big verdict (Healthy / Degraded / Critical, icon + text), checks per server, list of open findings (`FindingCard`). Overview gets its "Open findings" card. Findings export as markdown.

**Tests:** one scenario per rule on the test cluster (e.g. stop `n3` → `server.unreachable` within 10s; restart → finding resolves). Engine unit tests with hand-built snapshots.

**Manual check:** `docker compose -f docker-compose.cluster.yml stop n3` → health turns Degraded within 10s; start it → Healthy.

---

## Step 10: Connections explorer
**Commit:** `feat(connections): searchable connections across all servers`

**Problem.** Can't find a misbehaving client; the dashboard table shows one account's connections from one server.

**Change:**
- `/c/:ctx/connections` with tabs **Open** · **Closed** (with reason) · **Slow consumers** (closed with a slow-consumer reason + open with pending bytes above a threshold).
- Server-side merge across servers: `CONNZ` supports `sort`, `offset`, `limit`, `state`; ask each server for its top `offset+limit` by the chosen sort and merge. Columns: cid, server, name, account, user, lang/version, IP, uptime, idle, RTT, subs, pending, msgs/bytes in/out.
- Filters in the URL: account, name, IP, lang, user, server. Virtualised table (TanStack Table + react-virtual).
- Inspector: details + subscription list (`subs_detail`).
- Account scope: own account only, with the badge.

**Tests:** 10k-connection fixture scrolls smoothly; merge/sort unit test across 3 servers; closed tab shows the reason after a client disconnects on the test cluster.

---

## Step 11: Kick a connection
**Commit:** `feat(connections): disconnect a client from the explorer`

**Needs:** production safety from Phase 0.

**Change:** `POST /connections/:server/:cid/kick` → `$SYS.REQ.SERVER.<id>.KICK` with `{"cid": N}`. Goes through `OperationPreview` (who, which server, what happens: the client will reconnect unless it's broken) and `DangerConfirm` in production contexts. Copy as CLI. Gated: system account + NATS 2.10+ (**verify** the exact version against nats-server source).

**Tests:** kick a test client → it's closed with reason "Kicked"; non-system user gets the gate, not a 500.

---

## Step 12: Accounts view
**Commit:** `feat(accounts): account usage, limits, imports and exports`

**Change:**
- `/c/:ctx/accounts`: table from `ACCOUNTZ` + account `STATZ` (connections, leaf nodes, subscriptions, msgs/bytes sent/received, JetStream usage vs limits as `UsageBar`).
- `/c/:ctx/accounts/:name`: tabs **Usage** · **Imports / Exports** (subject, type stream/service, from/to account, mapping) · **Mappings** · **JWT** (decoded claims, operator mode only) · **Raw**.
- Account scope: only the current account (from `$SYS.REQ.USER.INFO` + `$JS.API.INFO`).

**Tests:** test cluster config gets a second account with one export/import; both show up and link to each other.

---

## Step 13: Topology graph
**Commit:** `feat(topology): live graph of servers, routes, gateways and leaf nodes`

**Change:**
- `/c/:ctx/topology`: React Flow + ELK (lazy-loaded). Nodes = servers grouped by cluster; edges = routes, gateways (between clusters), leaf nodes (remote side shown as an external node). Edge labels: msgs/s from the sampler.
- Select a node → Inspector with the server summary and a link to its detail page.
- Unreachable servers stay on the graph, greyed with "not responding".
- Gated: system account (a single server via monitoring port shows a one-node graph with the badge).

**Tests:** graph model built from routez/gatewayz/leafz fixtures (unit test, not pixels); stopping a node greys it.

---

## Step 14: JetStream health
**Commit:** `feat(jetstream): leaders, replica lag and consumer lag`

**Problem.** Replication and lag problems are found late.

**Change:**
- `/c/:ctx/health?tab=jetstream`: meta group (leader, peers: current / lag / offline), then streams (leader, each replica's `current`, `lag`, `active`) and consumers (leader, pending, ack pending, redelivered, lag sparkline from step 5).
- Data: `JSZ` with `accounts`, `streams`, `consumers` per account, paged. Account scope uses `StreamInfo.Cluster` / `ConsumerInfo.Cluster` from the JetStream API, which carry the same replica info for the own account.
- Streams table (existing) gets a health column and a growth sparkline.
- New rules: `js.stream_no_leader`, `js.replica_offline`, `js.replica_lagging` (lag > threshold for > 1m), `js.consumer_lag_growing` (pending trending up over 15m).
- Overview gets "Top lagging consumers".

**Tests:** R3 stream, stop a node → `js.replica_offline` for that stream; restart → resolves. Steady publishing with no consumer → `js.consumer_lag_growing`.

---

## Step 15: Consumer diagnostics
**Commit:** `feat(diagnose): automatic findings for stuck and misconfigured consumers`

**Problem.** Consumer problems are found when customers complain.

**Change:** new rules, each with evidence and a fix:

| Rule | Fires when | Fix offered |
|---|---|---|
| `consumer.paused` | Consumer is paused (2.11+) | Resume (operation, with preview) |
| `consumer.ack_pending_full` | `num_ack_pending ≥ max_ack_pending` | Explain; copy CLI to raise the limit |
| `consumer.high_redelivery` | Redelivered / delivered > 10% over 15m | Show `ack_wait`, link to dead letters |
| `consumer.no_interest` | Push consumer with no active subscriber | Show deliver subject |
| `consumer.nobody_pulling` | Pull consumer with pending > 0, no waiting pulls, inactive > 5m | Show last active |
| `consumer.filter_matches_nothing` | Filter subject(s) match no subject in the stream | Show stream subjects that look close |
| `consumer.max_deliver_hit` | Max-deliveries advisories in the last hour (after step 18; skipped until then) | Link to dead letters |

- Findings show on the Health page, on the stream detail's consumers list, and in the Inspector of a consumer.
- Only actions that exist in Phase 1 are one-click (resume). Others are "Copy as CLI" until the consumer workspace (Phase 2).

**Tests:** one scenario per rule on `natstest` (e.g. `max_ack_pending=5`, receive 5 without acking → `consumer.ack_pending_full`).

---

## Step 16: Subject explorer
**Commit:** `feat(subjects): subject tree showing who listens and what is stored`

**Problem.** Subjects are invisible.

**Change:**
- `/c/:ctx/subjects`: tree by token (`orders` → `orders.eu` → `orders.eu.created`), lazy-expanded, search by prefix.
- Badges: 👂 **listening** (from `SUBSZ` with details; wildcard subscriptions shown on their own node), 💾 **stored** (stream subjects + per-subject counts via stream info `subjects_filter`, capped at 10k subjects per stream with a "truncated" note).
- Inspector: subscribers (connection, queue group), streams that capture it, consumers whose filter matches it.
- Wildcard tester: "does `orders.*.created` match `orders.eu.created`?" — one matcher in Go and one in TS sharing a table of test cases (JSON file).
- Account scope: subscriptions from own-account `CONNZ` with subs (**verify** `$SYS.REQ.ACCOUNT.PING.CONNZ` accepts the `subscriptions` option).

**Tests:** matcher table; tree from fixtures; subscriber on the test cluster appears under the right node.

---

## Step 17: Traffic sampling + real subject autocomplete
**Commit:** `feat(subjects): opt-in traffic sampling and real subject suggestions`

**Needs:** WebSocket live updates.

**Change:**
- 📈 **flowing** badge: "Sample 60s" button subscribes to `>` (or the selected prefix) for 60s, capped at 10k msg/s and 100k messages, excluding `$SYS.>` and `_INBOX.>`; counts per subject only (payloads not kept). Badge: "sampled 60s at 14:02". Permission errors shown as a reason, not a failure.
- Replace the hardcoded list in `getSubjects` (`internal/api/subscribe.go`, `publish.go`) with real suggestions: stream subjects + subscriptions + last sample. `SubjectInput` uses it everywhere (publish, subscribe, wizard).

**Tests:** sampling stops at 60s and at the cap; suggestions contain a stream subject; the old hardcoded list is gone.

---

## Step 18: Event collector + Events feed
**Commit:** `feat(events): live JetStream and system events`

**Needs:** WebSocket live updates.

**Change:**
- `internal/events`: one collector per pooled connection, started with the sampler. Subscribes to:
  - JetStream advisories in the account: `$JS.EVENT.ADVISORY.>` (leader elected, quorum lost, max deliveries, message terminated, stream/consumer created/deleted, API audit).
  - With system account: `$SYS.ACCOUNT.*.CONNECT` / `DISCONNECT`, `$SYS.SERVER.*.SHUTDOWN`, `$SYS.SERVER.*.STATSZ` heartbeats (also used by `server.unreachable` for faster detection).
- Typed events (type, time, server, account, resource, summary, raw); ring of the last 10k per cluster.
- `/c/:ctx/events`: live list with filters (type, account, stream, server) in the URL, pause, rate shown; Dock tab "Events". Overview gets "Recent events".
- `GET /events?since=` for history from the ring; live via WebSocket topic `events`.

**Tests:** create/delete a stream → two events; disconnect a client (sys) → disconnect event; ring cap.

---

## Step 19: Dead letters
**Commit:** `feat(dead-letters): messages that hit max deliveries, with recovery`

**Problem.** Failed messages quietly drop out of processing.

**Change:**
- Sources: `CONSUMER.MAX_DELIVERIES` and `CONSUMER.MSG_TERMINATED` advisories (from step 18). Each entry is enriched by fetching the message by `stream_seq` (direct get when the stream allows it, otherwise `STREAM.MSG.GET`). If it's gone: "no longer in stream (limits/retention)".
- `/c/:ctx/dead-letters`: table (time, stream, consumer, seq, subject, deliveries, reason) + message inspector.
- Actions (preview + confirm): **republish** to the original subject or another subject; **delete** from the stream. Bulk for selected rows, rate-limited.
- **Capture history** (opt-in, writes to the cluster, so preview + confirm): creates a `HEYNATS_DLQ` stream that stores those advisories, so history survives HeyNATS restarts and covers time HeyNATS wasn't running. Without it, the page says "only since HeyNATS started listening".

**Tests:** consumer with `max_deliver=2`, nack twice → entry with the right seq and payload; republish delivers again; capture stream receives advisories while HeyNATS is stopped.

---

## Step 20: Message trace (NATS 2.11+)
**Commit:** `feat(trace): trace a test message through the system`

**Problem.** Routing across accounts, leaf nodes and gateways is a black box.

**Change:**
- `/c/:ctx/trace`: subject + optional headers/payload → publish with `Nats-Trace-Dest: <inbox>` and, by default, `Nats-Trace-Only: true` (the message is not delivered). Unticking "trace only" sends a real message, which needs confirmation in production.
- Collect trace events for a few seconds (ingress, subject mapping, stream export, service import, JetStream, egress) and render a hop timeline + graph (reusing the topology graph component).
- Result summary: "delivered to 3 subscribers on 2 servers, stored in ORDERS" or "no egress: nobody is listening".
- Gated: NATS 2.11+ on every server in the path.

**Tests:** on the cluster, subscriber on `n2`, publish on `n1` → trace shows ingress `n1`, route, egress `n2`; stream capture shows a JetStream hop; no subscriber → no egress.

---

## Step 21: "Why didn't my message arrive?" wizard
**Commit:** `feat(diagnose): delivery wizard with evidence and fixes`

**Problem.** Debugging delivery needs NATS expertise and many CLI commands.

**Change:**
- `/c/:ctx/diagnose`: input subject, where it should arrive (core subscriber · stream · consumer · KV key) and optionally "published as" (a saved connection).
- Runs a checklist; each check is ✓ / ✗ / ? with evidence and source, ✗ stops at the most likely cause but still shows the rest:

| Check | Evidence | Catches |
|---|---|---|
| Publisher may publish to the subject | `$SYS.REQ.USER.INFO` run with the chosen saved connection; else test publish | permission denied |
| Someone listens | `SUBSZ` / own-account subs; trace egress (2.11+) | no subscriber |
| A stream captures it | stream subjects (matcher from step 16) | wrong stream subjects |
| Stream accepted it | limits, discard policy, `max_msg_size`, `max_msgs_per_subject`, last publish error from a test publish | stream limit |
| Consumer filter matches | filter subjects vs. subject | wrong filter |
| Consumer is delivering | step-15 rules (paused, ack pending full, nobody pulling, no interest) | paused consumer, full ack-pending |
| Cross-account path | imports/exports (step 12) | missing import/export |

- "Send a test message": trace-only on 2.11+, otherwise a real message with header `HeyNATS-Probe: <id>` after confirmation.
- Fixes link to operations that exist (resume consumer, kick) or offer Copy as CLI; everything else is explained, not guessed.
- Export the result as markdown for an incident ticket.
- "Published as" is honest: without that connection's credentials, permission checks say "unknown (can't check another user's permissions)".

**Tests:** one end-to-end scenario per failure in the "done when" list, on `natstest`, each asserting the wizard's top finding.

---

## Step 22: Latency probes
**Commit:** `feat(probes): measure request, publish and JetStream latency`

**Change:**
- `/c/:ctx/probes` (Debug group in nav). Probe types:
  - **Request/reply:** subject, payload, count (≤ 1,000), concurrency, timeout.
  - **Publish round trip:** publish to a unique subject and receive it on the same connection.
  - **JetStream publish ack:** to a chosen stream (writes data, so confirmation; offers a temporary memory stream that's deleted afterwards).
- Runs on the HeyNATS server (labelled "measured from the HeyNATS host"), as a cancellable task in the Dock.
- Result: p50 / p90 / p99 / max, histogram, errors by kind (timeout, no responders, permission).
- Rate-limited; at most one running probe per cluster per user.

**Tests:** responder with a 20ms delay → p50 ≈ 20ms; no responder → errors counted as "no responders"; cancel stops it.

---

## Step 23: Phase acceptance
**Commit:** `test: Phase 1 failure scenarios and NATS version matrix`

**Change:**
- `internal/acceptance` (build tag `acceptance`): on the 3-node `natstest` cluster, one test per "done when" item:
  - stop a node → `server.unreachable` within 10s;
  - no subscriber, wrong filter, paused consumer, full ack-pending, stream limit, permission denied → the wizard's top finding is right;
  - app user without system account → each gated feature reports the right reason in `/capabilities`.
- CI matrix job: `natstest` can run an external `nats-server` binary (`NATSTEST_SERVER_BIN`); CI downloads 2.10, 2.11 and 2.12 and runs the acceptance suite against each. Features above a version must be gated, not fail.
- Playwright smoke: Overview, Health, Connections, Diagnose against the Docker cluster.

**Done when:** all green on all three versions; tick Phase 1 in ROADMAP.md.

---

## Open questions
1. **Sampling lifetime.** The sampler lives as long as the pooled connection. Should a saved connection keep sampling with nobody watching (useful for team deployments, costs memory)? Proposal: off in local mode, opt-in setting in team mode.
2. **Probes page placement.** UI.md doesn't list it; proposal: `/c/:ctx/probes` under Debug. Update UI.md when step 22 lands.
3. **Version checks marked (verify):** kick (step 11), account-scope subscriptions (step 16), exact trace event names (step 20).
4. **Phase 1 vs Phase 2 order** (see "Before starting").
