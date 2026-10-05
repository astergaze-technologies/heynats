# HeyNATS: Roadmap

Each phase lists the features with **the problem they solve** and **how they improve HeyNATS**.
Order follows the agreed priority: foundation first (required), then Observe & Debug, JetStream depth, Dev workbench, Automate, Team mode.

| Phase | Theme | Product shift |
|---|---|---|
| 0 | Foundation | A stable, safe base to build everything on |
| 1 | Observe & Debug | Snapshot → monitoring and root-cause finding |
| 2 | JetStream depth | Full visibility and control of JetStream |
| 3 | Dev workbench | Act directly on messages |
| 4 | Automate & multi-cluster | Repeatable and comparable across environments |
| 5 | Team mode | Safe for whole teams |
| Later | Extensions | Ecosystem and intelligence |

---

## Phase 0: Foundation
Must happen first. The current app has crash bugs and a design that won't scale to the features below.
Step-by-step plan, one PR per step: [PHASE-0-FOUNDATION.md](PHASE-0-FOUNDATION.md).

| Feature | Problem it solves | How it improves HeyNATS |
|---|---|---|
| Critical bug fixes | The server can crash (live subscription with a message limit), leaks NATS connections, serialises all users behind one lock, ignores new credentials on reconnect, and the UI has broken calls (create KV bucket, keys with "/", only 20 KV keys shown) | Stable and trustworthy; no more crashes or silent failures |
| New connection manager | Every request pings NATS while holding a global lock; dead connections pile up | Fast requests, correct reconnects, several connections per user |
| Multiple connections + shareable URLs | One cluster per browser session; links don't say which cluster | Several clusters open at once; links can be shared |
| Live updates over one WebSocket | Browser allows only ~6 live streams; busy subjects freeze the tab; drops are silent | Unlimited tails, smooth at high rates, drops visible |
| Clear API + generated client | Hand-written client drifts from the server (404s) | Fewer bugs; the API is usable by the CLI and scripts |
| Config, logging, graceful shutdown | Port hardcoded, no settings, debug mode, abrupt exit | Configurable, production-ready |
| Security basics | Unauthenticated endpoint can dial any host (SSRF); credentials stored in the browser in plain text; container runs as root | Safe to run on a laptop or a shared server |
| Local / Team mode skeleton | No way to run safely for a team | Room to add SSO, roles and audit later without a rewrite |
| App shell redesign | Pages are islands; live tails die on navigation; nothing shows production risk | Workbench layout: inspector, dock, ⌘K, environment colour band |
| Production safety | Deleting a stream is one click, even in prod | Read-only prod by default, previews, typed confirmation, Copy as CLI |
| Tests, CI and releases | Almost no tests; no lint/test CI; Docker only | Confidence to ship; binaries via Homebrew, Docker, later Helm |

**Done when:** no known crash or leak; several clusters open at once; live views stay smooth under heavy traffic; production contexts are protected; CI runs lint and tests (including against a real embedded NATS server).

---

## Phase 1: Observe & Debug

| Feature | Problem it solves | How it improves HeyNATS |
|---|---|---|
| Overview & dashboards (rates, connections, JetStream, lag; 5m–24h) | Dashboard is a single snapshot and, in a cluster, shows one server | See trends and spikes without Grafana; the first stop in an incident |
| Health page & findings | No single "is everything OK?" answer | Green/red verdict with reasons |
| Servers & topology graph | Cluster structure only visible as raw JSON | Understand the deployment in seconds |
| Connections explorer (incl. closed & slow consumers, kick) | Can't find a misbehaving client | Find slow or leaking clients quickly |
| Accounts view | Imports/exports and limits are hidden in config | Clear picture of multi-tenant setups |
| JetStream health (leaders, replica lag, consumer lag) | Replication and lag problems found late | Spot unhealthy streams/consumers early |
| Subject explorer | Subjects are invisible; autocomplete is fake | Browse the subject space; better naming and onboarding |
| "Why didn't my message arrive?" wizard | Needs NATS expertise and many CLI commands | Anyone can debug delivery problems, with evidence and one-click fixes |
| Consumer diagnostics (automatic findings) | Consumer problems found when customers complain | Problems found and explained proactively |
| Events feed & dead letters | Failed messages quietly drop out of processing | No silent loss; built-in recovery path |
| Message trace (NATS 2.11+) | Routing across accounts/leafs/gateways is a black box | See the exact path of a message |
| Latency probes | No quick way to measure request or publish latency | Measured numbers (p50/p99) instead of guesses |

**Done when:** on a 3-node test cluster, stopping a node shows in health within seconds; each common failure scenario (no subscriber, wrong filter, paused consumer, full ack-pending, stream limit, permission denied) is diagnosed correctly; features that need a system account or newer NATS are shown disabled with the reason.

---

## Phase 2: JetStream depth

| Feature | Problem it solves | How it improves HeyNATS |
|---|---|---|
| Stream workspace (config form ⇄ YAML, diff before save) | Creating a stream silently drops settings; streams can't be edited | Full lifecycle, safe edits, nothing lost |
| Stream message browser (by sequence/time/subject, permalinks) | Paging is wrong when searching; search is very slow | Instant, accurate "what's in this stream?" |
| Stream subjects (counts, top subjects) | Subject explosions and per-subject limits are invisible | Catch cardinality problems early |
| Purge / seal / delete message / backup & restore | Needs the CLI | Common operations in one place, with previews |
| Consumer workspace (create/edit, pause, priority groups) | Consumers aren't visible in the UI at all | See and manage consumers directly |
| Safe peek at pending messages | Looking at queued messages risks affecting delivery | Inspect without side effects |
| Consumer "what-if" preview | Wrong filters/deliver policy found only after creation | Mistakes caught before they happen |
| KV workbench (search, history, diff, watch, conflict-safe save) | Only 20 keys shown; edits overwrite each other; no history | KV as a real data browser, safe with several editors |
| Object Store | Not supported | Covers the full JetStream feature set |
| Lineage graph (mirrors, sources, republish) | Data flows between streams are hard to follow | See where stream data comes from and goes |

**Done when:** every JetStream resource can be created, viewed, edited and deleted from the UI with previews, and large streams/buckets stay fast.

---

## Phase 3: Dev workbench

| Feature | Problem it solves | How it improves HeyNATS |
|---|---|---|
| Unified message inspector + payload decoding (JSON, msgpack, CBOR, protobuf) | Three different viewers; binary payloads mangled; only the first header value shown | One correct viewer that leads to actions |
| Publish composer (templates, variables, repeat, JetStream ack) | Empty payloads rejected; no templates; can't see if JetStream stored it | Replaces throwaway scripts for test traffic |
| Subscriptions in the dock + mock responders | Tails tied to one page; "auto-reply" does nothing | Test against services that don't exist yet |
| Request/reply with latency & clear errors | Every failure shows as "408"; custom reply ignored | Fast diagnosis: "nobody listening" vs "slow service" |
| Replay studio (+ transforms) | Replays need custom code, risk duplicates | Safe, repeatable recovery with dry run and rate limit |
| Search & query language (with cost estimate) | Finding messages by content needs code | Predictable search, no surprise full scans |
| Command palette & shortcuts | Everything is mouse-driven | CLI speed inside the UI |
| `heynats` CLI + Copy as CLI/code | UI actions can't be reproduced or scripted | Every action works in scripts and docs |
| Micro services explorer | Services are invisible | Discover, call and measure services |

**Done when:** a developer can publish, subscribe, inspect, replay and script any action without leaving HeyNATS.

---

## Phase 4: Automate & multi-cluster

| Feature | Problem it solves | How it improves HeyNATS |
|---|---|---|
| Export / import with plan view (YAML, nats CLI, NACK, Terraform) | Config lives only in the cluster; copying between environments is manual | Reproducible infrastructure that works with existing GitOps tools |
| Environment diff | "Works in staging, not prod" caused by invisible config drift | Drift visible in seconds; promotion becomes a reviewed plan |
| Change log (who changed what) | NATS keeps no config history | Answers "what changed?" immediately in incidents |
| Fleet view | Each cluster checked separately | One place to watch dev, staging, prod and regions |
| Investigation bundle + local playground | Production problems can't be safely shared or reproduced | Capture → share → open offline → reproduce locally, with redaction |
| Historical metrics via Prometheus | Built-in history covers hours, not weeks | Long-term trends without a new database |

**Done when:** configuration can be exported, compared and applied between two environments through a reviewed plan.

---

## Phase 5: Team mode

| Feature | Problem it solves | How it improves HeyNATS |
|---|---|---|
| SSO login (OIDC) | No authentication for shared deployments | Company login, no shared passwords |
| Roles per connection, payload visibility control | Everyone can do everything and see all data | Least privilege; sensitive data protected |
| Server-side encrypted connections, shared by role | Each person manages their own credentials | Central, secure credential handling |
| Audit log | No record of who did what | Accountability and compliance |
| Allowed NATS targets | Users could point the server anywhere | Controlled network access |
| Shared views, saved queries, investigations | Debugging knowledge stays in one browser | Teams reuse each other's work |
| Helm chart | No standard Kubernetes deployment | One-command team install |
| Account/user management (JWT setups) | Needs `nsc` and expertise | Visual, audited administration |

---

## Later
| Feature | Problem it solves |
|---|---|
| Alerts & synthetic checks | Problems noticed only when someone looks |
| Distributed-trace links (Jaeger/Tempo) | No link from a NATS message to the app trace |
| GitOps (`heynats plan/apply` in CI) | Manual config promotion |
| Plugins | Custom codecs/panels need a fork |
| AI explanations (evidence-cited) | Findings still need interpretation for newcomers |
| Schema contracts per subject | Breaking payload changes found in production |

---

## Appendix A: Key engineering decisions
| Topic | Decision | Why |
|---|---|---|
| JetStream client | Move fully to the new `jetstream` package | Supports newer features, cancellation, clearer errors |
| Connections | Pool per saved connection; health from NATS events, not pings | No global lock, no leaks, correct reconnects |
| Live data | One WebSocket per tab, batched, with backpressure | Scales to busy subjects; no browser connection limit |
| API | Code-first OpenAPI (huma) + generated TypeScript client | No drift between server and client |
| Config | File + env + flags (koanf) | Works for laptop, Docker and Kubernetes |
| Logging / errors | Structured logs (slog); one standard error format | Easier debugging and support |
| Monitoring data | System account first, monitoring port second, account-only fallback | Works with whatever access the user has |
| Charts | uPlot (charts), React Flow (graphs) | Fast with lots of data; graphs loaded only when needed |
| Payload decoding | In the browser, pluggable | Instant switching, no schemas uploaded in local mode |
| Distribution | goreleaser (binaries + Homebrew), Docker, Helm | Easy install for every audience |

## Appendix B: NATS version requirements
| Feature | Needs |
|---|---|
| Monitoring (servers, connections, subjects, JetStream, health) | NATS 2.10+, system account or monitoring port |
| Connection kick | 2.10+ (verify), system account |
| Message trace, consumer pause, priority groups, per-message TTL | 2.11+ |
| Atomic batch publish, counters, message schedules | 2.12+ (verify each) |

The UI shows unavailable features as disabled with the reason ("Requires NATS 2.11", "Requires system account").

## Appendix C: How we test
- Backend: tests against a real NATS server started inside the tests, with the race detector; compatibility runs against NATS 2.10, 2.11 and 2.12.
- Frontend: component tests with mocked API; end-to-end tests (Playwright) against the real app and NATS; accessibility checks on every page; performance check at 5,000 msg/s.
- Every diagnosis rule has a test scenario that must produce the right finding.
