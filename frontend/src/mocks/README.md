<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [MSW (Mock Service Worker) Integration](#msw-mock-service-worker-integration)
  - [Overview](#overview)
  - [Directory Structure](#directory-structure)
  - [Quick Start](#quick-start)
    - [For Testing (Bun)](#for-testing-bun)
    - [For Browser/Development](#for-browserdevelopment)
  - [Features](#features)
    - [WebSocket (Using MSW's Experimental ws API)](#websocket-using-msws-experimental-ws-api)
    - [REST API Endpoints](#rest-api-endpoints)
  - [Customizing Handlers](#customizing-handlers)
    - [Adding Custom Handlers for Specific Tests](#adding-custom-handlers-for-specific-tests)
    - [Modifying WebSocket Events](#modifying-websocket-events)
    - [Generating Handlers from OpenAPI](#generating-handlers-from-openapi)
  - [Integration with RTK Query](#integration-with-rtk-query)
  - [React 19 Features](#react-19-features)
  - [Troubleshooting](#troubleshooting)
    - [MSW not intercepting requests](#msw-not-intercepting-requests)
    - [WebSocket not working](#websocket-not-working)
    - [Tests hanging](#tests-hanging)
  - [Migration from Existing Mocks](#migration-from-existing-mocks)
  - [Resources](#resources)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# MSW (Mock Service Worker) Integration

This directory contains the MSW setup for mocking API requests in tests and development.

## Overview

We use a **hybrid mocking solution** combining:

- **msw-auto-mock**: Auto-generated handlers for REST endpoints from OpenAPI 3.1 spec
- **Manual handlers**: WebSocket handlers using MSW's experimental `ws` API

**Note**: SSE (Server-Sent Events) is deprecated for this project. Use WebSocket for real-time streaming.

## Directory Structure

```plaintext
src/mocks/
├── generatedHandlers.ts   # Auto-generated REST API handlers (placeholder)
├── streamingHandlers.ts   # WebSocket handlers (SSE deprecated)
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
if (
  process.env.NODE_ENV === "development" &&
  process.env.ENABLE_MSW === "true"
) {
  import("./mocks/browser").then(({ startMockWorker }) => {
    startMockWorker();
  });
}
```

3. Run your app with MSW enabled:

```bash
ENABLE_MSW=true bun run dev
```

## Features

### WebSocket (Using MSW's Experimental ws API)

The WebSocket handler uses MSW's native `ws` API to mock `/ws` endpoint:

- Sends initial `hello` message on connection
- Sends `heartbeat` messages every 500ms
- Responds to `SUBSCRIBE` messages with requested event data
- Supports all event types: hello, heartbeat, volumes, shares, updating, dirty_data_tracker, smart_test_status

Example usage:

```typescript
const ws = new WebSocket("ws://localhost:8080/ws");

ws.onmessage = (event) => {
  const [id, eventType, data] = event.data.split("\n");
  // Handle streamed data
};

// Subscribe to specific events
ws.send(JSON.stringify({ type: "SUBSCRIBE", event: "volumes" }));
```

**Note**: SSE (Server-Sent Events) is deprecated for this project. All real-time streaming should use WebSocket.

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
import { mswServer } from "../../test/bun-setup";
import { http, HttpResponse } from "msw";

describe("Custom test", () => {
  it("uses custom handler", () => {
    mswServer.use(
      http.get("/api/custom", () => {
        return HttpResponse.json({ custom: "data" });
      }),
    );

    // Your test here
  });
});
```

### Modifying WebSocket Events

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

**Note**: SSE handlers have been removed as SSE is deprecated for this project.

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
      query: () => "/api/sse",
      async onCacheEntryAdded(
        arg,
        { updateCachedData, cacheDataLoaded, cacheEntryRemoved },
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

### WebSocket not working

1. Verify `streamingHandlers` are included in `node.ts` handlers array (currently empty as WebSocket handler is exported separately)
2. Check console for MSW warnings about unhandled requests
3. Ensure proper event names match `Supported_events` enum
4. The WebSocket handler uses MSW's experimental `ws` API - ensure you're using a compatible MSW version

**Note**: SSE is deprecated and not supported. Use WebSocket for real-time streaming.

### Tests hanging

1. WebSocket streams may keep connections open
2. Use `waitFor` with appropriate timeouts
3. Ensure cleanup in `afterEach` hooks

## Migration from Existing Mocks

**See [MSW_TEST_MIGRATION.md](MSW_TEST_MIGRATION.md) for detailed test migration tracking and status.**

To migrate from existing mock implementations:

1. Remove global fetch mocks in favor of MSW handlers
2. Convert HAR files to MSW handlers (if needed)
3. Replace manual mocks with MSW handlers
4. Update tests to use MSW's runtime handler overrides

Most tests will work automatically since they already import `test/setup.ts` which now includes MSW setup.

See the main `COPILOT.md` for testing patterns and guidelines.

## Resources

- [MSW Documentation](https://mswjs.io/)
- [MSW with Bun](https://bun.sh/guides/test/mock-http-api)
- [msw-auto-mock](https://www.npmjs.com/package/msw-auto-mock)
- [RTK Query Streaming](https://redux-toolkit.js.org/rtk-query/usage/streaming-updates)
