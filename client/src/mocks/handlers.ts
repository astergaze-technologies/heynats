import { HttpResponse, http } from 'msw';
import account from './fixtures/account.json';
import info from './fixtures/info.json';
import kvBuckets from './fixtures/kv-buckets.json';
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
  http.get('/api/nats/streams', () => HttpResponse.json(streams)),
  http.get('/api/nats/kv/buckets', () => HttpResponse.json(kvBuckets)),
];
