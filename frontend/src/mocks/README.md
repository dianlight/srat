# MSW (Mock Service Worker) Integration

This directory contains the MSW setup for mocking API requests in tests and development.

## Overview

We use a **hybrid mocking solution** combining:
- **msw-auto-mock**: Auto-generated handlers for REST endpoints from OpenAPI 3.1 spec
- **Manual handlers**: Custom streaming handlers for SSE (Server-Sent Events) and WebSocket

## Directory Structure

```
src/mocks/
├── generatedHandlers.ts   # Auto-generated REST API handlers (placeholder)
├── streamingHandlers.ts   # Manual SSE and WebSocket handlers
├── node.ts                # MSW server setup for Bun/Node environment
└── browser.ts             # MSW worker setup for browser environment

test/
└── bun-setup.ts          # MSW lifecycle hooks for Bun tests
```

## Quick Start

### For Testing (Bun)

MSW is automatically configured in the test environment. Just write your tests:

```typescript
import { describe, it, expect } from "bun:test";

describe("My Component", () => {
  it("fetches data from API", async () => {
    // MSW will automatically intercept and mock API calls
    // See test/__tests__/msw-integration.test.tsx for examples
  });
});
```

### For Browser/Development

1. Initialize MSW in your public directory:
```bash
bunx msw init public/ --save
```

2. Conditionally start MSW in development (e.g., `src/index.tsx`):
```typescript
if (process.env.NODE_ENV === 'development' && process.env.ENABLE_MSW === 'true') {
  import('./mocks/browser').then(({ startMockWorker }) => {
    startMockWorker();
  });
}
```

3. Run your app with MSW enabled:
```bash
ENABLE_MSW=true bun run dev
```

## Features

### Server-Sent Events (SSE)

The SSE handler mocks `/api/sse` endpoint and emits events every 500ms:

- `hello` - Welcome message with server info
- `heartbeat` - Server health status
- `volumes` - Disk volumes list
- `shares` - Shares list
- `updating` - Update progress
- `dirty_data_tracker` - Data change tracking
- `smart_test_status` - SMART test status

Example usage:
```typescript
import { useGetServerEventsQuery } from '../store/sseApi';

function MyComponent() {
  const { data } = useGetServerEventsQuery();
  
  // data.hello - Welcome message
  // data.heartbeat - Latest heartbeat
  // data.volumes - Volumes list
  // etc.
}
```

### WebSocket

The WebSocket handler mocks `/ws` endpoint and:
- Sends initial `hello` message on connection
- Sends `heartbeat` messages every 500ms
- Responds to `SUBSCRIBE` messages with requested event data

Example usage:
```typescript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const [id, eventType, data] = event.data.split('\n');
  // Handle streamed data
};

// Subscribe to specific events
ws.send(JSON.stringify({ type: 'SUBSCRIBE', event: 'volumes' }));
```

### REST API Endpoints

All REST endpoints defined in the OpenAPI spec are mocked with realistic data.

Currently available (examples in `generatedHandlers.ts`):
- `GET /api/health` - Health check
- `GET /api/shares` - List shares
- `GET /api/volumes` - List volumes
- `GET /api/settings` - Get settings

## Customizing Handlers

### Adding Custom Handlers for Specific Tests

```typescript
import { mswServer } from '../../test/bun-setup';
import { http, HttpResponse } from 'msw';

describe("Custom test", () => {
  it("uses custom handler", () => {
    mswServer.use(
      http.get('/api/custom', () => {
        return HttpResponse.json({ custom: 'data' });
      })
    );
    
    // Your test here
  });
});
```

### Modifying SSE Events

Edit `src/mocks/streamingHandlers.ts` and update the `mockEventData` object:

```typescript
const mockEventData = {
  heartbeat: (): HealthPing => ({
    // Customize mock data here
    alive: true,
    aliveTime: Date.now(),
    // ...
  }),
};
```

### Generating Handlers from OpenAPI

To auto-generate handlers from the OpenAPI spec:

1. Add script to `package.json`:
```json
{
  "scripts": {
    "gen:mocks": "msw-auto-mock ../../backend/docs/openapi.json -o src/mocks/generatedHandlers.ts"
  }
}
```

2. Run generation:
```bash
bun run gen:mocks
```

## Integration with RTK Query

MSW works seamlessly with RTK Query's `onCacheEntryAdded` for streaming:

```typescript
export const api = createApi({
  endpoints: (build) => ({
    getServerEvents: build.query({
      query: () => '/api/sse',
      async onCacheEntryAdded(
        arg,
        { updateCachedData, cacheDataLoaded, cacheEntryRemoved }
      ) {
        // MSW will provide mocked SSE stream
        // Your streaming logic here
      },
    }),
  }),
});
```

See `test/__tests__/msw-integration.test.tsx` for complete examples.

## React 19 Features

The test examples demonstrate React 19 features:
- `use` hook for async data
- Transitions for non-blocking updates
- Modern component patterns

## Troubleshooting

### MSW not intercepting requests

1. Check that MSW is started in `test/bun-setup.ts`
2. Verify fetch is not mocked globally in `test/setup.ts`
3. Ensure handlers are registered in `src/mocks/node.ts`

### SSE/WebSocket not working

1. Verify `streamingHandlers` are included in `node.ts` handlers array
2. Check console for MSW warnings about unhandled requests
3. Ensure proper event names match `Supported_events` enum

### Tests hanging

1. SSE/WebSocket streams may keep connections open
2. Use `waitFor` with appropriate timeouts
3. Ensure cleanup in `afterEach` hooks

## Migration from Existing Mocks

To migrate from existing mock implementations:

1. Remove global fetch mocks in favor of MSW handlers
2. Convert HAR files to MSW handlers (if needed)
3. Replace manual mocks with MSW handlers
4. Update tests to use MSW's runtime handler overrides

See the main `COPILOT.md` for testing patterns and guidelines.

## Resources

- [MSW Documentation](https://mswjs.io/)
- [MSW with Bun](https://bun.sh/guides/test/mock-http-api)
- [msw-auto-mock](https://www.npmjs.com/package/msw-auto-mock)
- [RTK Query Streaming](https://redux-toolkit.js.org/rtk-query/usage/streaming-updates)
