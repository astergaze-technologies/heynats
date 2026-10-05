# HeyNATS: Web UI

## 1. Goal
A fast, keyboard-first web workbench for NATS that feels like browser DevTools, not an admin panel.

**Principles**
1. Always show the cluster, environment and connection status.
2. Everything is a link: subjects, streams, consumers, connections, messages.
3. Select → inspect → act: list in the middle, details on the right, actions in the details and ⌘K.
4. Live but controlled: pause, rate shown, drops never hidden.
5. Show where data comes from ("via system account", "sampled", "inferred").
6. Safe in production: read-only by default, previews, typed confirmation.
7. Copy as CLI/code for every action.

## 2. Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ ENVIRONMENT BAND  (prod = red · staging = amber · dev = blue)     │
├──────┬──────────────────────────────────────────────────────────┤
│      │ [Cluster ▾ ● connected]   [⌘K Search…]   [Live 3 · 0 drops] │
│ Nav  ├──────────────────────────────────────┬───────────────────┤
│      │ Title · breadcrumb · source badge · actions               │
│      │                                      │                   │
│      │ Main area (tables, browsers, charts) │ Inspector         │
│      │                                      │ (selected item)   │
│      ├──────────────────────────────────────┴───────────────────┤
│      │ Dock: Live tails · Publish · Events · Tasks (replay/export)│
└──────┴──────────────────────────────────────────────────────────┘
```

- **Nav:** grouped by Explore / Operate / Debug / Observe / Automate; collapsible.
- **Inspector:** details of whatever is selected; can be pinned or opened full page.
- **Dock:** live tails and long tasks keep running while you navigate.
- **Cluster switcher:** recent connections grouped by environment, with health dots.
- **Unavailable features** stay visible but dimmed, with a tooltip saying why.

## 3. Navigation & URLs
Everything for a cluster lives under `/c/:ctx`. Filters, tabs, time range and selection live in the URL, so every view can be shared.

| Area | Pages |
|---|---|
| Start | `/` connections · `/fleet` all clusters · `/settings` |
| Overview | `/c/:ctx` |
| Explore | subjects · topology · servers · connections · accounts · services |
| Operate | streams · streams/:stream (tabs) · streams/:stream/consumers/:consumer · kv · kv/:bucket · objects · access |
| Debug | messages · replay · trace · diagnose · events · dead-letters |
| Observe | monitor · health |
| Automate | export · diff · changes |

Message permalink: `/c/:ctx/streams/ORDERS/messages?seq=12345`.

**Keyboard:** `⌘K` palette · `⌘⇧K` switch cluster · `g s` streams · `g k` KV · `g m` messages · `g d` diagnose · `/` search · `j/k` move · `Enter` open · `Esc` close · `` ` `` dock · `p` pause live · `?` help.

## 4. Screens

| Screen | What it shows | Key actions | Data needs |
|---|---|---|---|
| Connections home | Saved connections by environment, health, version | Connect, test, import from nats CLI | exists / new |
| Fleet | Tile per cluster: health, versions, rates, findings | Open cluster | new |
| Overview | Key stats, traffic chart, findings, top lagging consumers, recent events | Drill into anything | new (partly exists) |
| Subjects | Subject tree with badges: listening 👂 · stored 💾 · flowing 📈 | Live tail, sample 60s, test wildcard | new |
| Topology | Graph of servers, clusters, gateways, leaf nodes with live rates | Select node → details | new (system account) |
| Servers | Table + detail (charts, config, routes, raw JSON) | — | new (system account) |
| Connections | Big table; tabs open / closed (reason) / slow consumers | Inspect subs, kick | new (system account) |
| Accounts | Usage vs limits; imports/exports; JWT details | — | new |
| Streams | Table with health, size, growth sparkline; saved views | New stream, import, bulk export | exists + new |
| Stream workspace | Tabs: Overview · Messages · Consumers · Subjects · Config · Lineage · Activity | Edit (diff preview), purge, seal, backup, delete | exists + new |
| Message browser | Rows by seq/time/subject; infinite scroll both ways; live mode | Jump to seq/time, filter, query (with cost), replay, export, compare | new (interim exists) |
| Consumer workspace | Lag, ack-pending gauge, redeliveries, health verdict | Edit, pause/resume, safe peek, dev console | new |
| Key-Value | Key tree + table; tabs Keys / Watch / Config | Edit JSON, conflict-safe save, history diff, bulk | exists + new |
| Object Store | Buckets and files | Upload, download, metadata, delete | new |
| Access | Permission checker | "Can user X publish Y?" | new |
| Messages | Tabs Publish / Subscribe / Request | Templates, repeat, mock responders, latency histogram | exists + new |
| Replay | Steps: source → preview → transform → target → options → dry run | Run, see progress in Dock | new |
| Trace | Hop timeline + graph (2.11+) | Trace-only test message | new |
| Diagnose | Findings + "Why didn't my message arrive?" wizard | One-click fixes, export as markdown | new |
| Events / Dead letters | Live typed events; failed messages | Replay, delete, capture history | new |
| Monitor / Health | Charts with time range, synced crosshair; health checks | Zoom, split by server | new |
| Export / Diff / Changes | Resource picker; environment diff; change timeline | Plan → apply, sync plan | new |
| Settings | Theme, density, time format, payload decoders, live limits, shortcuts | — | — |

Every screen has: loading, empty, error (with request id + retry) and "not available" states.

## 5. Design system
**Keep:** the current semantic colour tokens (WCAG AA in both themes) and the rule of no raw colour classes.

**Add:**
- Colours for environments (prod / staging / dev / local), info, a colour-blind-safe chart palette, and status (ok / warn / critical / unknown).
- A monospace font for subjects, sequences, IDs and payloads; tabular numbers in tables.
- Density switch: compact / comfortable.
- Shared formatting for bytes, rates, counts (1.2k, 3.4M), durations, and local/UTC time.
- Respect reduced motion; live updates never animate layout.
- Status always uses icon + text, never colour alone.

**Shared components**

| Group | Components |
|---|---|
| Shell | AppShell, NavRail, TopBar, EnvBand, Dock, Inspector, ClusterSwitcher, StatusPill |
| Data | DataTable (virtual rows, sort, filters, column picker, saved views), MessageList, SubjectTree |
| Messages | MessageInspector, PayloadView (pretty/raw/hex/tree), HeadersTable, CodeEditor, DiffView |
| Input | SubjectInput (wildcard-aware autocomplete), SeqNavigator, TimeRangePicker |
| Charts | StatTile, Sparkline, TimeSeriesChart, Histogram, UsageBar, Graph |
| Trust & safety | SourceBadge, Gate ("needs NATS 2.11"), DangerConfirm, OperationPreview, CopyAsMenu |
| Feedback | FindingCard, EmptyState, LoadingState, ErrorState, shortcut hints |

Add from shadcn/radix: dropdown menu, popover, select, checkbox, switch, sheet, badge, context menu, resizable panels, command (cmdk), skeleton, progress.

## 6. Frontend architecture
**Keep:** React 19, Vite, TypeScript, Tailwind 4, React Router 7, TanStack Query, react-virtual, shadcn/radix, sonner, lucide, Biome.

**Add:** TanStack Table, cmdk, react-resizable-panels, zustand (UI state), react-hook-form + zod (big forms), CodeMirror 6, uPlot, React Flow + ELK, payload decoders (msgpack, CBOR, protobuf), a JSON diff library. All heavy libraries load only on the pages that need them.

**Where state lives**
| Kind | Where |
|---|---|
| Server data | TanStack Query; keys always start with the cluster id, so switching clusters never shows stale data |
| Live data | One LiveClient per tab (SSE now, WebSocket later); ring buffer per subscription; screen updates at most once per frame |
| Filters, tabs, selection | URL |
| Layout & preferences | zustand, saved locally (never credentials) |

**Live data rules:** buffer of 5,000 messages per subscription by default (no array copying per message); show rate and drops; stop auto-scroll at high rates; preview large payloads and load the full one on demand; decode only visible rows.

**Folder layout**

```
client/src/
  app/          router, shell (layout, nav, top bar, dock, inspector, palette)
  components/   shared components (+ ui/ for shadcn only)
  lib/          api/, live/, codecs/, format/, subject/, shortcuts/, state/
  features/     one folder per area: contexts, fleet, overview, subjects, topology,
                servers, connections, accounts, services, streams, consumers, kv,
                objects, access, messaging, replay, trace, diagnose, events,
                dead-letters, monitor, health, export, diff, changes, settings
```

## 7. Quality
- **Performance budgets:** first load ≤ 250 KB gzip; each page ≤ 150 KB; 100k-row tables scroll smoothly; 5,000 msg/s live without input lag.
- **Tests:** component tests (vitest + Testing Library + MSW); end-to-end with Playwright against the real app and NATS; screenshots in light/dark and both densities.
- **Accessibility:** axe checks on every page; full keyboard use; focus handled in dialogs; screen-reader status updates throttled.
- **Mock-first:** agree on data shapes, build against realistic mock data (3-node cluster, 50 streams, 10k connections), then connect the real backend. Lets UI work run alongside backend work.

**A screen is done when** it has its own URL, survives refresh, keeps filters in the URL, has all states (loading/empty/error/not available), works by keyboard and in ⌘K, labels non-exact data with its source, sends changes through preview + confirmation with "Copy as CLI", works in both themes, passes accessibility checks, and has tests.

## 8. Build order
UI work follows the phases in ROADMAP.md. The app shell, message inspector, data table and live client come first (Phase 0) because every later screen uses them.
