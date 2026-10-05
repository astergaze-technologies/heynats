# HeyNATS: Product

## 1. What HeyNATS is
**HeyNATS is the interactive workbench for NATS.** Connect to any NATS deployment and get answers: what is running, where messages go, why something is broken, and how to fix it safely.
It's meant to be a debugger and workbench, not just a nicer JetStream admin panel.

One engine (HeyNATS Core) serves three ways in:

```
          Web UI          CLI (heynats)         API (REST + WebSocket)
             └──────────────────┼──────────────────────┘
                          HeyNATS Core
        Explore · Operate · Debug · Observe · Automate
                               │
                              NATS
```

## 2. Who it's for
| Person | Main need |
|---|---|
| App developer | See messages, publish/test, understand why a consumer isn't getting data |
| Platform / SRE | Health, topology, capacity, safe configuration changes across environments |
| On-call engineer | Find the root cause fast, recover (replay), know what changed |
| Team lead | Safe access for the whole team, audit, consistency between environments |

## 3. Questions HeyNATS answers
✅ yes · 🟡 partly / with conditions · ❌ not natively (best effort)

| Question | | How |
|---|---|---|
| What servers are in this cluster? | ✅ | Server info from the system account or monitoring port. A normal user sees only its own server. |
| Which streams and consumers exist? | ✅ | JetStream API; with system access, across all accounts (read-only). |
| Where is message traffic going? | 🟡 | Server-to-server rates, a JetStream lineage graph, message trace (NATS 2.11+), sampling. |
| Why isn't my consumer receiving messages? | ✅ | Diagnostics checklist with evidence and fixes. |
| What messages are in this stream? | ✅ | Message browser by sequence, time or subject. |
| Which consumers are falling behind? | ✅ | Lag, ack-pending and redelivery numbers, plus trends. |
| What subjects are active? | 🟡 | Who listens + what's stored + opt-in sampling of live traffic. |
| What is publishing to `orders.*`? | ❌ | NATS doesn't tell subscribers who sent a message. Shown as "likely publishers" (inferred), or exact when apps send trace headers. |
| How many messages flow through a subject? | 🟡 | Exact for subjects stored in streams; sampled for others. |
| Can I replay these 100 messages? | ✅ | Replay studio with preview, dry run and rate limit. |
| Can I inspect/edit a KV entry? | ✅ | KV workbench with history and conflict-safe saves. |
| What changed in this stream's config? | 🟡 | NATS keeps no history. HeyNATS records changes (who + diff) from the moment it starts capturing them. |
| Create a stream/consumer with a CLI-like command? | ✅ | Command bar + `heynats` CLI. |
| Export the configuration? | ✅ | YAML/JSON, nats CLI, Kubernetes (NACK), Terraform. |
| Reproduce a production problem locally? | ✅ | Investigation bundle → local playground server. |
| Monitor several clusters from one place? | ✅ | Fleet view across saved connections. |

**Rule:** every view says where its data comes from ("via system account", "account scope", "sampled 60s", "inferred").

## 4. What's in the product (by pillar)

### Explore: understand the system
- **Subjects:** a tree of subjects showing who listens, what's stored, and what's flowing.
- **Topology:** servers, clusters, gateways and leaf nodes as a live graph.
- **Servers:** details, limits, config, health.
- **Connections:** every client with sort/filter, closed connections with reasons, slow consumers.
- **Accounts:** imports/exports, limits, usage, JWT details.
- **Services:** NATS micro services discovery and stats.

### Operate: manage resources safely
- **Streams:** full config (form + YAML, diff before saving), state, limits, replicas, mirrors/sources, purge/seal, backup/restore.
- **Consumers:** create/edit, pause/resume (2.11+), lag and health, safe peek at waiting messages.
- **Key-Value:** browse/search keys, edit JSON, history and diffs, live watch, bulk import/export.
- **Object Store:** buckets and files, upload/download, metadata.
- **Access:** "can this user publish/subscribe X?" checker. Account/user management in JWT setups (team mode).

### Debug: find and fix problems
- **Message inspector:** one viewer everywhere (headers, payload decoding, copy, permalink, replay).
- **Messaging workbench:** publish with templates, subscribe live, request/reply with latency, mock responders.
- **Replay studio:** pick messages → preview → optional transform → target → rate limit → dry run.
- **Diagnose:** "Why didn't my message arrive?" wizard plus automatic findings (stalled consumers, high redelivery, filter mistakes, stream limits, permission problems).
- **Trace (NATS 2.11+):** the exact path of a test message through the system.
- **Events & dead letters:** live JetStream/system events; messages that hit max deliveries, with recovery.
- **Flow map:** where a subject's messages go (subscribers → streams → consumers → mirrors).

### Observe: is it healthy, what's the trend?
- **Overview & dashboards:** message rates, connections, JetStream storage, consumer lag (5 minutes to 24 hours).
- **Health:** per-server checks and open findings.
- **History:** longer periods through Prometheus if available.
- **Alerts & synthetic checks** (later).

### Automate: repeatable work
- **CLI:** `heynats` with the same operations as the UI.
- **Copy as CLI / code:** every action can be copied as a nats/heynats command, curl, Go, TypeScript or Python.
- **Export / import:** configs with a plan view before applying.
- **Diff:** compare environments and generate a sync plan.
- **Change log:** who changed what, when.
- **GitOps (later):** `heynats plan/apply` in CI.

## 5. How it's built: four ideas that make everything else cheap
1. **Operation registry.** Each action ("purge stream") is defined once. From that definition you get the API endpoint, CLI command, ⌘K palette entry, "copy as code", safety confirmation, audit entry and a dry-run preview.
2. **Pluggable data sources.** Live connection, system account, monitoring port, Prometheus, an investigation bundle file, or a local playground server. The UI works the same on all of them.
3. **Diagnostics engine.** Plain, testable rules that turn evidence into findings with fixes. AI (later) only summarises them and must cite the evidence.
4. **Message pipeline.** Source → filter → transform → destination. Shared by browsing, search, replay and export.

## 6. Product principles
- **Answers over forms.** Prioritise what answers real questions.
- **Honest data.** Show source and confidence; never present a guess as fact.
- **Safe by default.** Production is read-only until unlocked; changes are previewed; destructive actions need typed confirmation.
- **Everything is copyable and linkable.** Every view has a URL; every action can be reproduced outside the UI.
- **Keyboard-first and fast.** ⌘K, shortcuts, smooth with large data and busy subjects.
- **Evidence first, AI second.**

## 7. Two modes, one binary
| | Local mode | Team mode |
|---|---|---|
| For | One developer on their machine | A shared deployment for a team |
| Login | None (bound to localhost) | SSO (OIDC) / basic |
| Saved connections | Encrypted on this machine | Encrypted on the server, shared by role |
| Permissions | You are admin | Roles per connection (viewer / operator / admin) |
| Audit log | Local file | Central, exportable |
| Allowed NATS targets | Any | Allowlist |
| Distribution | Homebrew / binary / Docker | Docker / Helm |

## 8. What we won't do
- Claim to know who published a message: show "likely publishers" only.
- Sample all traffic by default: sampling is opt-in, time-limited and capped.
- Build our own long-term metrics database: use Prometheus for long ranges.
- Replace Terraform or Kubernetes operators: export to and import from them instead.
- Run arbitrary user scripts on a shared server.
