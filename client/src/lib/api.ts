// API client functions for NATS operations
export interface ConnectionCredentials {
  host: string;
  port: string;
  username: string;
  password: string;
  token?: string;
  nkeySeed?: string;
  creds?: string;
}

export interface ConnectionStatus {
  connected: boolean;
  host?: string;
  port?: string;
  username?: string;
}

export interface NATSInfo {
  server_info: {
    server_id: string;
    server_name: string;
    connected_url: string;
    last_error: string | null;
  };
  stats: {
    in_msgs: number;
    out_msgs: number;
    in_bytes: number;
    out_bytes: number;
    reconnects: number;
  };
  is_connected: boolean;
  connected_url: string;
}

export interface Connection {
  cid: number;
  idle: string;
  in_bytes: number;
  in_msgs: number;
  ip: string;
  kind: string;
  lang: string;
  last_activity: string;
  name: string;
  out_bytes: number;
  out_msgs: number;
  pending_bytes: number;
  port: number;
  rtt: string;
  start: string;
  subscriptions: number;
  type: string;
  uptime: string;
  version: string;
}

export interface AccountInformation {
  account: string;
  client_id: number;
  client_ip: string;
  connected_addr: string;
  connected_url: string;
  expires: string;
  header_supported: boolean;
  local_ip: string;
  max_payload: string;
  permissions: any;
  rtt: string;
  server_id: string;
  server_name: string;
  server_version: string;
  user: string;
}

// CONNZ summed over every server that answered.
export interface ConnectionLimits {
  source: DataSource;
  partial: boolean;
  servers: number;
  expected: number;
  connections: Connection[];
  limit: number;
  now: string;
  num_connections: number;
  total: number;
  errors?: string[];
}

export type DataSource = 'system' | 'account';

// Mirrors the requirements table in internal/monitor/capabilities.go.
export type FeatureName =
  | 'servers'
  | 'connections.all_accounts'
  | 'connections.kick'
  | 'consumers.pause'
  | 'trace';

export interface Capabilities {
  source: DataSource;
  server_version: string;
  servers: number;
  expected: number;
  partial: boolean;
  features: Record<FeatureName, { ok: boolean; reason?: string }>;
}

/** Chart-ready history: series[name][member][i] is the value at t[i] (unix s). */
export interface MetricsWindow {
  source?: DataSource;
  history_since: string;
  step: number;
  t: number[];
  series: Record<string, Record<string, (number | null)[]>>;
}

export interface JetStreamSummary {
  source: DataSource;
  partial: boolean;
  servers: number;
  expected: number;
  memory: number;
  storage: number;
  max_memory: number; // -1 = unlimited
  max_storage: number;
  streams: number;
  consumers: number;
}

export interface ServerRow {
  name: string;
  id: string;
  cluster?: string;
  version: string;
  responding: boolean;
  health?: string; // "ok" or the server's reason
  start?: string;
  cpu: number;
  mem: number;
  connections: number;
  subscriptions: number;
  slow_consumers: number;
  jetstream: boolean;
  msgs_in: number | null;
  msgs_out: number | null;
}

export interface ServerList {
  partial: boolean;
  expected: number;
  servers: ServerRow[];
}

export type ServerKind =
  | 'VARZ'
  | 'HEALTHZ'
  | 'ROUTEZ'
  | 'GATEWAYZ'
  | 'LEAFZ'
  | 'JSZ'
  | 'SUBSZ'
  | 'CONNZ'
  | 'ACCOUNTZ';

export type Severity = 'critical' | 'warning' | 'info';

export interface Finding {
  rule_id: string;
  severity: Severity;
  resource: string;
  title: string;
  detail: string;
  evidence: { label: string; value: string; source: string }[];
  fixes?: {
    label: string;
    link?: string;
    cli?: string;
    action?: 'consumer.resume';
    params?: Record<string, string>;
  }[];
  first_seen: string;
  last_seen: string;
}

export interface DiagnosticsReport {
  verdict: 'healthy' | 'degraded' | 'critical';
  checked_at: string;
  source: DataSource;
  servers: { name: string; responding: boolean; health?: string }[];
  findings: Finding[];
  rules: { id: string; doc: string; available: boolean; reason?: string }[];
}

/** A client connection; cid is only unique per server. */
export interface Conn {
  server: string;
  cid: number;
  name?: string;
  account?: string;
  authorized_user?: string;
  lang?: string;
  version?: string;
  ip: string;
  port: number;
  start: string;
  last_activity: string;
  stop?: string;
  reason?: string;
  rtt?: string;
  uptime: string;
  idle: string;
  pending_bytes: number;
  in_msgs: number;
  out_msgs: number;
  in_bytes: number;
  out_bytes: number;
  subscriptions: number;
}

export interface ConnPage {
  source: DataSource;
  partial: boolean;
  servers: number;
  expected: number;
  total: number;
  matched: number;
  connections: Conn[];
}

export interface ConnDetail extends Conn {
  server_id: string;
  tls_version?: string;
  subscriptions_list_detail?: {
    subject: string;
    qgroup?: string;
    msgs: number;
    max?: number;
    sid: string;
  }[];
}

export type ConnState = 'open' | 'closed' | 'slow';

/** One account summed over all servers; limits are -1 when unknown or unlimited. */
export interface AccountRow {
  name: string;
  system: boolean;
  connections: number;
  leafnodes: number;
  subscriptions: number;
  msgs_sent: number;
  msgs_received: number;
  bytes_sent: number;
  bytes_received: number;
  slow_consumers: number;
  jetstream: boolean;
  js_memory: number;
  js_storage: number;
  js_max_memory: number;
  js_max_storage: number;
}

export interface AccountList {
  source: DataSource;
  partial: boolean;
  servers: number;
  expected: number;
  accounts: AccountRow[];
}

export interface AccountImport {
  name?: string;
  subject: string;
  account: string;
  type: 'stream' | 'service';
  local_subject?: string;
  share?: boolean;
  invalid?: boolean;
}

export interface AccountExport {
  name?: string;
  subject: string;
  type: 'stream' | 'service';
  token_req?: boolean;
  approved_accounts?: string[];
  description?: string;
}

export interface AccountDetail {
  name: string;
  update_time?: string;
  detail: {
    account_name: string;
    jetstream_enabled: boolean;
    exports?: AccountExport[];
    imports?: AccountImport[];
    mappings?: Record<
      string,
      { subject: string; weight: number; cluster?: string }[]
    >;
    decoded_jwt?: unknown;
    [key: string]: unknown;
  };
}

export interface TopoNode {
  id: string;
  kind: 'server' | 'gateway' | 'leaf';
  label: string;
  cluster?: string;
  responding: boolean;
  version?: string;
}

/** Counters are cumulative; rates come from two polls. */
export interface TopoEdge {
  from: string;
  to: string;
  kind: 'route' | 'gateway' | 'leaf';
  links: number;
  in_msgs: number;
  out_msgs: number;
}

export interface Topology {
  partial: boolean;
  expected: number;
  nodes: TopoNode[];
  edges: TopoEdge[];
}

/** A replica as its group leader sees it; the leader itself isn't listed. */
export interface Peer {
  name: string;
  current: boolean;
  offline?: boolean;
  active: number; // nanoseconds since last seen
  lag?: number;
}

export interface StreamHealth {
  account: string;
  name: string;
  leader: string;
  replicas: Peer[] | null;
  clustered: boolean;
  messages: number;
  bytes: number;
  consumers: number;
}

export interface ConsumerHealth {
  account: string;
  stream: string;
  name: string;
  leader: string;
  replicas: Peer[] | null;
  pending: number;
  ack_pending: number;
  redelivered: number;
  deliver_subject?: string;
  push_bound?: boolean;
  waiting: number;
  max_ack_pending: number;
  paused?: boolean;
}

export interface JetStreamHealth {
  source: DataSource;
  partial: boolean;
  meta?: { leader: string; cluster_size: number; peers: Peer[] | null };
  streams: StreamHealth[];
  consumers: ConsumerHealth[];
}

export interface AccountInfo {
  account_information: AccountInformation;
  connection_limits: ConnectionLimits;
  stats: ConnectionLimits; // Same structure as connection_limits
}

const API_BASE = '/api';

class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public details?: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

async function apiRequest<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  // Proxies and gateways can answer with HTML, e.g. a 502 page.
  const data = await response.json().catch(() => ({}));

  if (!response.ok) {
    throw new ApiError(
      data.error || 'An error occurred',
      response.status,
      data.details
    );
  }

  return data;
}

// NATS API functions
export const natsApi = {
  // Check connection status
  getStatus: (): Promise<ConnectionStatus> => apiRequest('/nats/status'),

  // Connect to NATS server
  connect: (
    credentials: ConnectionCredentials
  ): Promise<{ message: string; connected: boolean }> =>
    apiRequest('/nats/connect', {
      method: 'POST',
      body: JSON.stringify(credentials),
    }),

  // Try credentials without creating a session
  test: (
    credentials: ConnectionCredentials
  ): Promise<{ ok: boolean; server_name: string; version: string }> =>
    apiRequest('/nats/test', {
      method: 'POST',
      body: JSON.stringify(credentials),
    }),

  // Disconnect from NATS server
  disconnect: (): Promise<{ message: string; connected: boolean }> =>
    apiRequest('/nats/disconnect', {
      method: 'POST',
    }),

  // Get NATS server info
  getInfo: (): Promise<NATSInfo> => apiRequest('/nats/info'),

  // Get account information
  getAccountInfo: (): Promise<AccountInfo> => apiRequest('/nats/account'),

  getCapabilities: (): Promise<Capabilities> =>
    apiRequest('/nats/capabilities'),

  getJetStream: (): Promise<JetStreamSummary> => apiRequest('/nats/jetstream'),

  getJetStreamHealth: (): Promise<JetStreamHealth> =>
    apiRequest('/nats/jetstream/health'),

  getServers: (): Promise<ServerList> => apiRequest('/nats/servers'),

  getAccounts: (): Promise<AccountList> => apiRequest('/nats/accounts'),

  getTopology: (): Promise<Topology> => apiRequest('/nats/topology'),

  getAccount: (name: string): Promise<AccountDetail> =>
    apiRequest(`/nats/accounts/${encodeURIComponent(name)}`),

  getDiagnostics: (): Promise<DiagnosticsReport> =>
    apiRequest('/nats/diagnostics'),

  resumeConsumer: (
    stream: string,
    consumer: string
  ): Promise<{ resumed: boolean }> =>
    apiRequest(
      `/nats/streams/consumers/${encodeURIComponent(stream)}/${encodeURIComponent(consumer)}/resume`,
      { method: 'POST' }
    ),

  getConnections: (params: Record<string, string>): Promise<ConnPage> =>
    apiRequest(`/nats/connections?${new URLSearchParams(params)}`),

  getConnection: (server: string, cid: number): Promise<ConnDetail> =>
    apiRequest(`/nats/connections/${encodeURIComponent(server)}/${cid}`),

  kickConnection: (server: string, cid: number): Promise<{ kicked: boolean }> =>
    apiRequest(`/nats/connections/${encodeURIComponent(server)}/${cid}/kick`, {
      method: 'POST',
    }),

  getServerRaw: (id: string, kind: ServerKind): Promise<any> =>
    apiRequest(`/nats/servers/${encodeURIComponent(id)}/raw/${kind}`),

  getMetrics: (series: string[], range: string): Promise<MetricsWindow> =>
    apiRequest(
      `/nats/metrics?${new URLSearchParams({ series: series.join(','), range })}`
    ),
};

// JetStream Stream interfaces
export interface StreamConfig {
  name?: string;
  subjects?: string[];
  retention?: string;
  max_consumers?: number;
  max_msgs?: number;
  max_bytes?: number;
  discard?: string;
  max_age?: number;
  max_msgs_per_subject?: number;
  max_msg_size?: number;
  storage?: string;
  num_replicas?: number;
  duplicate_window?: number;
  compression?: string;
  allow_direct?: boolean;
  mirror_direct?: boolean;
  consumer_limits?: any;
  metadata?: any;
  allow_msg_ttl?: boolean;
}

export interface StreamState {
  messages?: number;
  bytes?: number;
  first_seq?: number;
  first_ts?: string;
  last_seq?: number;
  last_ts?: string;
  consumer_count?: number;
  deleted?: any;
  num_deleted?: number;
  num_subjects?: number;
  subjects?: any;
}

export interface Stream {
  config?: StreamConfig;
  created?: string;
  state?: StreamState;
}

export interface StreamsResponse {
  streams: Stream[];
  total: number;
}

// Stream Message interfaces
export interface StreamMessage {
  sequence: string;
  subject: string;
  data: string;
  headers?: Record<string, string>;
  timestamp: string;
  size: number;
}

export interface StreamMessagesResponse {
  messages: StreamMessage[];
  total: number;
  offset: number;
  limit: number;
  stream_name: string;
}

// Streams API functions
export const streamsApi = {
  // Get all streams
  getStreams: (): Promise<StreamsResponse> => apiRequest('/nats/streams'),

  // Get stream details
  getStream: (streamName: string): Promise<Stream> =>
    apiRequest(`/nats/streams/${encodeURIComponent(streamName)}`),

  // Get stream messages with pagination and search
  getStreamMessages: (
    streamName: string,
    offset?: number,
    limit?: number,
    search?: string
  ): Promise<StreamMessagesResponse> => {
    const params = new URLSearchParams();
    params.append('offset', String(offset || 0));
    params.append('limit', String(limit || 10));
    if (search) {
      params.append('search', search);
    }
    return apiRequest(
      `/nats/streams/${encodeURIComponent(streamName)}/messages?${params.toString()}`
    );
  },

  // Create a new stream
  createStream: (config: Partial<StreamConfig>): Promise<Stream> =>
    apiRequest('/nats/streams', {
      method: 'POST',
      body: JSON.stringify(config),
    }),

  // Delete a stream
  deleteStream: (streamName: string): Promise<{ message: string }> =>
    apiRequest(`/nats/streams/${encodeURIComponent(streamName)}`, {
      method: 'DELETE',
    }),
};

// Key-Value Store interfaces
export interface KVBucket {
  bucket: string;
  values: number;
  history: number;
  ttl: string;
  backing_store: string;
  bytes: number;
  is_compressed: boolean;
}

export interface KVBucketsResponse {
  buckets: KVBucket[];
}

export interface CreateBucketRequest {
  bucket: string;
  history?: number;
  ttl?: string;
}

export interface CreateBucketResponse {
  message: string;
  bucket: string;
}

export interface KVEntry {
  key: string;
  value: string;
  created: string;
  revision: string;
}

export interface KVEntriesResponse {
  items: KVEntry[];
  bucket: string;
  page: number;
  pageSize: number;
  total: number;
}

// Key-Value API functions
export const kvApi = {
  // Get all KV buckets
  getBuckets: (): Promise<KVBucketsResponse> => apiRequest('/nats/kv/buckets'),

  // Create a new KV bucket
  createBucket: (config: CreateBucketRequest): Promise<CreateBucketResponse> =>
    apiRequest('/nats/kv/buckets', {
      method: 'POST',
      body: JSON.stringify(config),
    }),

  // Get bucket details (placeholder - will need backend implementation)
  getBucket: (bucketName: string): Promise<KVBucket> =>
    apiRequest(`/nats/kv/buckets/${encodeURIComponent(bucketName)}`),

  // Delete a bucket (placeholder - will need backend implementation)
  deleteBucket: (bucketName: string): Promise<{ message: string }> =>
    apiRequest(`/nats/kv/buckets/${encodeURIComponent(bucketName)}`, {
      method: 'DELETE',
    }),

  // Get all keys in a bucket (placeholder - will need backend implementation)
  getBucketKeys: (bucketName: string): Promise<KVEntriesResponse> =>
    apiRequest(`/nats/kv/buckets/${encodeURIComponent(bucketName)}/keys`),

  // Get a specific key value (placeholder - will need backend implementation)
  getKey: (bucketName: string, key: string): Promise<KVEntry> =>
    apiRequest(
      `/nats/kv/buckets/${encodeURIComponent(bucketName)}/keys/${encodeURIComponent(key)}`
    ),

  // Set a key value (placeholder - will need backend implementation)
  setKey: (bucketName: string, key: string, value: string): Promise<KVEntry> =>
    apiRequest(
      `/nats/kv/buckets/${encodeURIComponent(bucketName)}/keys/${encodeURIComponent(key)}`,
      {
        method: 'PUT',
        body: JSON.stringify({ value }),
      }
    ),

  // Delete a key (placeholder - will need backend implementation)
  deleteKey: (bucketName: string, key: string): Promise<{ message: string }> =>
    apiRequest(
      `/nats/kv/buckets/${encodeURIComponent(bucketName)}/keys/${encodeURIComponent(key)}`,
      {
        method: 'DELETE',
      }
    ),
};

// Publish API interfaces
export interface PublishRequest {
  subject: string;
  data: string;
  headers?: Record<string, string>;
}

export interface PublishMessage {
  subject: string;
  data: string;
  headers?: Record<string, string>;
}

export interface BatchPublishRequest {
  messages: PublishMessage[];
}

export interface PublishResponse {
  success: boolean;
  subject: string;
  message_id?: string;
  error?: string;
}

export interface BatchPublishResponse {
  success: boolean;
  results: PublishResponse[];
  total: number;
  succeeded: number;
  failed: number;
}

export interface RequestReplyRequest {
  subject: string;
  data: string;
  headers?: Record<string, string>;
  timeout?: number; // in seconds
  reply_subject?: string;
}

export interface RequestReplyResponse {
  success: boolean;
  subject: string;
  reply_subject: string;
  request_data: string;
  reply_data?: string;
  error?: string;
  timeout?: boolean;
}

// Publish API
export const publishApi = {
  // Publish a single message
  publishMessage: (request: PublishRequest): Promise<PublishResponse> =>
    apiRequest('/nats/publish/message', {
      method: 'POST',
      body: JSON.stringify(request),
    }),

  // Publish multiple messages in batch
  publishBatch: (request: BatchPublishRequest): Promise<BatchPublishResponse> =>
    apiRequest('/nats/publish/batch', {
      method: 'POST',
      body: JSON.stringify(request),
    }),

  // Send request and wait for reply
  requestReply: (request: RequestReplyRequest): Promise<RequestReplyResponse> =>
    apiRequest('/nats/publish/request', {
      method: 'POST',
      body: JSON.stringify(request),
    }),

  // Get subject suggestions
  getSubjects: (): Promise<{ subjects: string[] }> =>
    apiRequest('/nats/publish/subjects'),
};

// Subscribe API interfaces
export interface SubscribeRequest {
  subject: string;
  queue_group?: string;
  max_messages?: number;
  subscription_type?: 'regular' | 'queue' | 'reply' | 'request-handler';
  auto_reply?: boolean;
  reply_template?: string;
}

export interface ReplyMessage {
  subject: string;
  reply_subject: string;
  data: string;
  timestamp: string;
  headers?: Record<string, string>;
}

// Subscribe API
export const subscribeApi = {
  // Get subject suggestions for autocomplete
  getSubjects: (): Promise<{ subjects: string[] }> =>
    apiRequest('/nats/subscribe/subjects'),

  // Send reply to a request message
  sendReply: (
    replySubject: string,
    data: string,
    headers?: Record<string, string>
  ): Promise<{ success: boolean }> =>
    apiRequest('/nats/subscribe/reply', {
      method: 'POST',
      body: JSON.stringify({ reply_subject: replySubject, data, headers }),
    }),
};

// Health check API
export const healthApi = {
  getHealth: (): Promise<{ status: string; server: string }> =>
    apiRequest('/health'),
};

export { ApiError };
