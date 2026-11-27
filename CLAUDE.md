# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

HeyNATS: web admin UI for NATS (connections, JetStream streams/consumers, KV, pub/sub). Go/Gin backend in repo root, React 19 + Vite SPA in `client/`. Go server serves both `/api/*` and the embedded SPA on `:5000`; unknown non-`/api` paths fall back to `index.html` (`internal/infrastructure/router.go`).

## Commands

```bash
make install          # go mod tidy + pnpm install in client/
make run              # build client, then go run main.go (:5000)
make dev-full         # ./dev.sh: Go server :5000 + Vite dev :5173 (proxies /api -> :5000)
make build            # client + linux/amd64 server binary -> bin/server
air                   # Go hot reload (runs under dlv on :2345, see .air.toml)
docker-compose up -d  # app + NATS 2.12 (JetStream, nats.conf)
make cluster-up       # 3-node NATS cluster n1-n3 (4222-4224, monitor 8222-8224); same host ports as docker-compose, run one

go test ./...                                     # only tests: internal/util
go test ./internal/util -run TestVersionComparison
go vet ./... && golangci-lint run ./...

cd client && pnpm lint        # biome@2.2.2 check (lint:fix / format to write)
cd client && pnpm typecheck   # tsc --noEmit
cd client && pnpm test        # vitest + Testing Library; MSW serves /api from src/mocks/fixtures
cd client && pnpm dev:mock    # Vite with MSW, no Go server needed
cd client && pnpm build       # tsc -b && vite build
```

README mentions `make lint`, which does not exist. Frontend tests (`*.test.ts(x)`) and `src/mocks/setup.ts` are excluded from `tsc`. Refresh fixtures with `scripts/capture-fixtures.sh` (app on :5000, `NATS_PASS` set).

The SPA is embedded into the Go binary (`//go:embed all:client/dist` in `main.go`), so the client must be built before `go build`/`go run` or the server serves a stale/empty UI. `client/dist/.gitkeep` is tracked (re-copied from `client/public/` on each Vite build) so Go compiles on a fresh clone.

## Hooks & commits

- lefthook: pre-commit runs biome, gofmt, goimports, go vet; pre-push runs golangci-lint, `go test`, tsc, client build.
- commitlint (`.commitlint.yml`): Conventional Commits, types `feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert`, header 10–150 chars.
- Default branch for PRs: `main`; work happens on `develop`. Release workflow: `.github/workflows/release.yml` (multi-arch Docker image).

## Architecture

**Backend wiring** (`main.go`): creates one `NatsConnectionStore` and one `ConnectionMiddleware`, then each API struct in `internal/api/` (`HeyNats`, `KVAPI`, `StreamAPI`, `PublishAPI`, `SubscribeAPI`) is built with `(routerGroup, store, middleware)` and registers its own routes via `RegisterRoutes()` under `/api/nats`. New API areas follow the same constructor + `RegisterRoutes` pattern and get registered in `main.go`.

**Connection/session model** — the key cross-cutting concept:
- `POST /api/nats/connect` dials NATS with host/port/user/pass, stores it in `NatsConnectionStore` under a UUID, and sets an HTTP-only `connection_id` cookie. The server holds live NATS connections per browser session; the client never holds credentials after connect.
- `NatsConnectionStore` (`internal/api/connection.go`) keeps the original `ConnectionRequest` so `GetOrReconnect` can transparently redial dead connections. A background goroutine closes connections idle > 5 min.
- Middleware (`internal/api/middleware.go`): `Handle()` optionally attaches the connection; `RequireConnection()` returns 401 (no/invalid cookie) or 503 (reconnect failed). Both put the connection in gin context under `NatsConnectionKey`; handlers fetch it via `GetNatsCredentialFromContext` (`internal/api/context.go`).
- All NATS operations live as methods on `pkg.NATSCredential` (`internal/pkg/nats_client.go`, KV in `internal/pkg/kv.go`); API handlers are thin wrappers.
- Monitoring (`internal/monitor`, via `NATSCredential.Monitor`): asks every server (`$SYS.REQ.SERVER.PING.*` for system-account users, else `$SYS.REQ.ACCOUNT.PING.*`) and gathers all replies; results say `source` and `partial`. `GET /api/nats/capabilities` reports which features the user/server support (table in `capabilities.go`). System-account only: `GET /api/nats/servers` (every known server; one that stops answering stays listed with `responding: false`) and `GET /api/nats/servers/:id/raw/:kind` (id and kind are checked before going into the NATS subject). `GET /api/nats/connections?state=open|closed|slow&sort=…` asks every server for its top rows with the same CONNZ sort and merges them (`MergeConns`; comparators must match CONNZ order); `account`/`user` filter on the servers, `server`/`name`/`ip`/`lang` after the merge. `GET /api/nats/connections/:server/:cid` adds subscriptions. `POST …/:server/:cid/kick` disconnects a client (system account + NATS 2.10+, checked with `Monitor.Feature`; the server name is resolved to its id from known servers, never put in the subject). `GET /api/nats/accounts` sums account STATZ, JSZ and ACCOUNTZ over servers (account scope: own account only); `GET /api/nats/accounts/:name` is ACCOUNTZ detail (imports, exports, mappings, JWT; system only). Dev `nats.conf` has APP exporting `events.>`/`billing.quote` to USER and one mapping, so these have data.
- Metrics (`internal/metrics`, via `NATSCredential.Metrics`): the connection store starts a `Sampler` per session (kept across reconnects, stopped on disconnect). Server series every `HEYNATS_METRICS_INTERVAL` (5s), JetStream every 3rd tick; history in fixed rings (fine for 1h, minute averages for 24h). `GET /api/nats/metrics?series=msgs_in,msgs_out&range=1h` returns `{history_since, step, t, series: {name: {member: [...]}}}`. `HEYNATS_METRICS_ENABLED=false` turns it off.
- Diagnostics (`internal/diagnose`, via `NATSCredential.Diagnose`): started with the sampler, checks every tick. `Snapshot` -> `Rules` (pure functions, one list in `rules.go`, each with a unit case in `rules_test.go`) -> `Finding`s kept open across checks so `first_seen` holds. `GET /api/nats/diagnostics` returns verdict, servers, findings and which rules ran. (`/api/nats/health` is the older connection ping.)
- Subscriptions (`internal/api/subscribe.go`) stream messages to the browser over SSE (`text/event-stream`).

**Frontend** (`client/src`), organised by feature:
- `app/`: router, `QueryProvider`, `ProtectedRoute` (gates `/dashboard/*` on connection status, redirects to `/`, toasts "connection lost" unless the user disconnected), layouts + `Sidebar`. New pages: add a route in `app/router.tsx` using the `page()` lazy helper and a `handle: { title, parent? }` — `DashboardLayout` renders the header title/breadcrumb and `document.title` from it. List-page search text lives in `?q=` via `lib/useSearchQuery`.
- `features/<name>/` (`accounts`, `connection`, `connections`, `dashboard`, `health`, `servers`, `streams`, `kv`, `messaging`): each holds its pages, components and hooks flat in one folder. `features/connection/useNATS.ts` is the connection-status hook used app-wide. `features/dashboard` is the Overview page (`/dashboard`): tiles from capabilities, `/api/nats/account`, `/api/nats/jetstream` and metrics, plus traffic charts.
- `components/`: cross-feature pieces (`ErrorBoundary`, `StatsCard`); `components/ui/` is shadcn only.
- `lib/api.ts`: typed fetch client for `/api` (cookie-based, `ApiError`). Server state goes through per-feature TanStack Query hooks with their own key factory (`useNATS` → `queryKeys`, `useKV` → `kvQueryKeys`, `useStreams` → `streamKeys`); connect/disconnect removes every non-`nats` query so data from a previous server never shows.
- Live messages (SSE): always use `lib/useEventSources` (keyed connections in a ref, closed on error and on unmount; `appendCapped` keeps the last 1000 messages). Don't store `EventSource` objects in React state — closures over that state go stale and leak connections.
- Saved connection profiles ("contexts") live client-side only, in IndexedDB with localStorage fallback (`features/connection/contexts/contextStorage.ts`).
- Theming: class-based dark mode (`.dark` on `<html>`, set pre-paint by an inline script in `client/index.html`, toggled by `components/ThemeToggle`). Use only the semantic tokens from `index.css` (`bg-card`, `bg-muted`, `text-muted-foreground`, `border-border`, `bg-primary`/`text-primary-foreground`, `success`/`warning`/`destructive` with `/10` tints) — every pair is chosen for WCAG AA in both themes. No raw `gray-*`/`blue-*` classes and no `dark:` variants in app code. Shared page pieces: `components/PageStates` (`PageIntro`, `EmptyState`, `LoadingState`), `StatsCard`, `Gate` (wrap anything that needs a capability; never decide availability in the UI), `SourceBadge` (where data came from, partial answers); lists of servers' resources are tables (`StreamsTable`, `BucketsTable`, `features/connections` (virtualised, merged across servers)).
- Charts (`components/charts`): `TimeSeriesChart` (uPlot, lazy-loaded; takes a `MetricsWindow` from `/api/nats/metrics`, legend+readout, table view, "history since"), `Sparkline`, `UsageBar` (80%/95% marks), `TimeRangePicker` + `useTimeRange` (`?range=`). Series colours are `--chart-1..8` in `index.css`, assigned in that order by member name, never cycled; pass stable `format` functions from `lib/format` (counts, rates, bytes, clock).
- Imports: `./x` within the same folder, `@/…` (aliases `client/src`) everywhere else. Tailwind 4.

More detail: `docs/ARCHITECTURE.md`, `docs/CONNECTION_MANAGEMENT.md`, `client/ERROR_HANDLING.md`, `client/STREAMS_README.md`.
