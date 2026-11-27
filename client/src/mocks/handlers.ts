import { HttpResponse, http } from 'msw';
import account from './fixtures/account.json';
import accountApp from './fixtures/account-app.json';
import accountUser from './fixtures/account-user.json';
import accounts from './fixtures/accounts.json';
import capabilities from './fixtures/capabilities.json';
import detail from './fixtures/connection-detail.json';
import connections from './fixtures/connections.json';
import closedConnections from './fixtures/connections-closed.json';
import diagnostics from './fixtures/diagnostics.json';
import info from './fixtures/info.json';
import jetstream from './fixtures/jetstream.json';
import kvBuckets from './fixtures/kv-buckets.json';
import metrics from './fixtures/metrics.json';
import routez from './fixtures/server-routez.json';
import varz from './fixtures/server-varz.json';
import servers from './fixtures/servers.json';
import status from './fixtures/status.json';
import streams from './fixtures/streams.json';

// Captured from a real cluster by scripts/capture-fixtures.sh.
let connected = true;

export const handlers = [
  http.get('/api/nats/status', () =>
    HttpResponse.json(connected ? status : { connected: false })
  ),
  http.post('/api/nats/connect', () => {
    connected = true;
    return HttpResponse.json({ connected, message: 'Connected (mock)' });
  }),
  http.post('/api/nats/disconnect', () => {
    connected = false;
    return HttpResponse.json({ connected, message: 'Disconnected (mock)' });
  }),
  http.get('/api/nats/info', () => HttpResponse.json(info)),
  http.get('/api/nats/account', () => HttpResponse.json(account)),
  http.get('/api/nats/capabilities', () => HttpResponse.json(capabilities)),
  http.get('/api/nats/streams', () => HttpResponse.json(streams)),
  http.get('/api/nats/kv/buckets', () => HttpResponse.json(kvBuckets)),
  http.get('/api/nats/metrics', () => HttpResponse.json(metrics)),
  http.get('/api/nats/jetstream', () => HttpResponse.json(jetstream)),
  http.get('/api/nats/servers', () => HttpResponse.json(servers)),
  http.get('/api/nats/accounts', () => HttpResponse.json(accounts)),
  http.get('/api/nats/accounts/:name', ({ params }) =>
    params.name === 'USER'
      ? HttpResponse.json(accountUser)
      : params.name === 'APP'
        ? HttpResponse.json(accountApp)
        : HttpResponse.json({ error: 'not found' }, { status: 404 })
  ),
  http.get('/api/nats/diagnostics', () => HttpResponse.json(diagnostics)),
  http.get('/api/nats/connections', ({ request }) =>
    HttpResponse.json(
      new URL(request.url).searchParams.get('state') === 'open'
        ? connections
        : closedConnections
    )
  ),
  http.get('/api/nats/connections/:server/:cid', () =>
    HttpResponse.json(detail)
  ),
  http.get('/api/nats/servers/:id/raw/:kind', ({ params }) =>
    HttpResponse.json(
      ({ VARZ: varz, ROUTEZ: routez } as Record<string, unknown>)[
        params.kind as string
      ] ?? {}
    )
  ),
];
