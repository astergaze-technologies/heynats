import { createBrowserRouter, type Params } from 'react-router-dom';
import { DashboardLayout } from '@/app/layouts/DashboardLayout';
import { RootLayout } from '@/app/layouts/RootLayout';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import { LoginPage } from '@/features/connection/LoginPage';
import { ProtectedRoute } from './ProtectedRoute';

/** Read by DashboardLayout to render the header title and breadcrumb. */
export interface RouteHandle {
  title: string | ((params: Params) => string);
  parent?: { label: string; to: string };
}

// Each page is its own chunk, loaded on first visit.
const page = <K extends string>(
  load: () => Promise<Record<K, React.ComponentType>>,
  name: K
) => ({
  lazy: async () => ({ Component: (await load())[name] }),
});

export const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    errorElement: <ErrorBoundary />,
    children: [
      { index: true, element: <LoginPage /> },
      {
        path: 'dashboard',
        element: (
          <ProtectedRoute>
            <DashboardLayout />
          </ProtectedRoute>
        ),
        children: [
          {
            index: true,
            handle: { title: 'Overview' } satisfies RouteHandle,
            ...page(
              () => import('@/features/dashboard/DashboardPage'),
              'DashboardPage'
            ),
          },
          {
            path: 'health',
            handle: { title: 'Health' } satisfies RouteHandle,
            ...page(() => import('@/features/health/HealthPage'), 'HealthPage'),
          },
          {
            path: 'servers',
            handle: { title: 'Servers' } satisfies RouteHandle,
            ...page(
              () => import('@/features/servers/ServersPage'),
              'ServersPage'
            ),
          },
          {
            path: 'servers/:serverName',
            handle: {
              title: (p) => p.serverName ?? '',
              parent: { label: 'Servers', to: '/dashboard/servers' },
            } satisfies RouteHandle,
            ...page(
              () => import('@/features/servers/ServerDetailPage'),
              'ServerDetailPage'
            ),
          },
          {
            path: 'streams',
            handle: { title: 'Streams' } satisfies RouteHandle,
            ...page(
              () => import('@/features/streams/StreamsPage'),
              'StreamsPage'
            ),
          },
          {
            path: 'streams/:streamName',
            handle: {
              title: (p) => p.streamName ?? '',
              parent: { label: 'Streams', to: '/dashboard/streams' },
            } satisfies RouteHandle,
            ...page(
              () => import('@/features/streams/StreamDetailPage'),
              'StreamDetailPage'
            ),
          },
          {
            path: 'kv',
            handle: { title: 'Key-Value Store' } satisfies RouteHandle,
            ...page(() => import('@/features/kv/KeyValuePage'), 'KeyValuePage'),
          },
          {
            path: 'kv/:bucketName',
            handle: {
              title: (p) => p.bucketName ?? '',
              parent: { label: 'Key-Value Store', to: '/dashboard/kv' },
            } satisfies RouteHandle,
            ...page(
              () => import('@/features/kv/KVBucketDetailPage'),
              'KVBucketDetailPage'
            ),
          },
          {
            path: 'publish',
            handle: { title: 'Publish' } satisfies RouteHandle,
            ...page(
              () => import('@/features/messaging/PublishPage'),
              'PublishPage'
            ),
          },
          {
            path: 'subscribe',
            handle: { title: 'Subscribe' } satisfies RouteHandle,
            ...page(
              () => import('@/features/messaging/SubscribePage'),
              'SubscribePage'
            ),
          },
        ],
      },
    ],
  },
]);

export default router;
