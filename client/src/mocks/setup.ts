import { cleanup } from '@testing-library/react';
import { setupServer } from 'msw/node';
import { afterAll, afterEach, beforeAll } from 'vitest';
import { handlers } from './handlers';

// jsdom has no matchMedia; the Sidebar reads it for the mobile layout.
window.matchMedia = (query) =>
  ({
    matches: false,
    media: query,
    addEventListener() {},
    removeEventListener() {},
  }) as unknown as MediaQueryList;

// jsdom has no ResizeObserver; charts use it to follow their card's width.
window.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

export const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => {
  cleanup();
  server.resetHandlers();
});
afterAll(() => server.close());
