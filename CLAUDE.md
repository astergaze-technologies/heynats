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

go test ./...                                     # only tests: internal/util
go test ./internal/util -run TestVersionComparison
go vet ./... && golangci-lint run ./...

cd client && pnpm lint        # biome@2.2.2 check (lint:fix / format to write)
cd client && pnpm typecheck   # tsc --noEmit
cd client && pnpm test        # node --test unit tests (src/**/*.test.ts)
cd client && pnpm build       # tsc -b && vite build
```

README mentions `make lint`, which does not exist. Frontend unit tests use Node's built-in runner and are excluded from `tsc`.

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
- Subscriptions (`internal/api/subscribe.go`) stream messages to the browser over SSE (`text/event-stream`).

**Frontend** (`client/src`), organised by feature:
- `app/`: router, `QueryProvider`, `ProtectedRoute` (gates `/dashboard/*` on connection status, redirects to `/`, toasts "connection lost" unless the user disconnected), layouts + `Sidebar`. New pages: add a route in `app/router.tsx` using the `page()` lazy helper and a `handle: { title, parent? }` — `DashboardLayout` renders the header title/breadcrumb and `document.title` from it. List-page search text lives in `?q=` via `lib/useSearchQuery`.
- `features/<name>/` (`connection`, `dashboard`, `streams`, `kv`, `messaging`): each holds its pages, components and hooks flat in one folder. `features/connection/useNATS.ts` is the connection-status hook used app-wide.
- `components/`: cross-feature pieces (`ErrorBoundary`, `StatsCard`); `components/ui/` is shadcn only.
- `lib/api.ts`: typed fetch client for `/api` (cookie-based, `ApiError`). Server state goes through per-feature TanStack Query hooks with their own key factory (`useNATS` → `queryKeys`, `useKV` → `kvQueryKeys`, `useStreams` → `streamKeys`); connect/disconnect removes every non-`nats` query so data from a previous server never shows.
- Live messages (SSE): always use `lib/useEventSources` (keyed connections in a ref, closed on error and on unmount; `appendCapped` keeps the last 1000 messages). Don't store `EventSource` objects in React state — closures over that state go stale and leak connections.
- Saved connection profiles ("contexts") live client-side only, in IndexedDB with localStorage fallback (`features/connection/contexts/contextStorage.ts`).
- Theming: class-based dark mode (`.dark` on `<html>`, set pre-paint by an inline script in `client/index.html`, toggled by `components/ThemeToggle`). Use only the semantic tokens from `index.css` (`bg-card`, `bg-muted`, `text-muted-foreground`, `border-border`, `bg-primary`/`text-primary-foreground`, `success`/`warning`/`destructive` with `/10` tints) — every pair is chosen for WCAG AA in both themes. No raw `gray-*`/`blue-*` classes and no `dark:` variants in app code. Shared page pieces: `components/PageStates` (`PageIntro`, `EmptyState`, `LoadingState`), `StatsCard`; lists of servers' resources are tables (`StreamsTable`, `BucketsTable`, `ConnectionsTable`).
- Imports: `./x` within the same folder, `@/…` (aliases `client/src`) everywhere else. Tailwind 4.

More detail: `docs/ARCHITECTURE.md`, `docs/CONNECTION_MANAGEMENT.md`, `client/ERROR_HANDLING.md`, `client/STREAMS_README.md`.
