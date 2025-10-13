# Redux DevTools Integration

This project uses **Redux Toolkit**'s built-in Redux DevTools support through the `configureStore()` function.

## Configuration

The Redux store is configured in `src/store/store.ts`:

```typescript
export const store = configureStore({
  // ... reducers and middleware
  devTools: process.env.NODE_ENV !== "production",
});
```

## Using Redux DevTools

1. Install the [Redux DevTools Extension](https://github.com/reduxjs/redux-devtools) for your browser:
   - [Chrome Extension](https://chrome.google.com/webstore/detail/redux-devtools/lmhkpmbekcpmknklioeibfkpmmfibljd)
   - [Firefox Extension](https://addons.mozilla.org/en-US/firefox/addon/reduxdevtools/)
   - [Edge Extension](https://microsoftedge.microsoft.com/addons/detail/redux-devtools/nnkgneoiohoecpdiaponcejilbhhikei)

2. Start the development server:
   ```bash
   cd frontend
   bun run dev
   ```

3. Open your browser and navigate to the application
4. Open the Redux DevTools in your browser's developer tools
5. You can now:
   - Inspect state changes
   - View dispatched actions
   - Time-travel debug (replay actions)
   - Export/import state snapshots

## Why No `@redux-devtools/extension` Package?

Redux Toolkit's `configureStore()` has built-in support for Redux DevTools Extension. The `@redux-devtools/extension` package is only needed when using vanilla Redux with `createStore()`. Since we're using Redux Toolkit, the package is unnecessary and has been removed to keep dependencies lean.

## Production Builds

Redux DevTools integration is automatically disabled in production builds (`process.env.NODE_ENV === "production"`) for performance and security reasons.

## Testing

Store configuration and DevTools integration are tested in `src/store/__tests__/store.test.ts`.
