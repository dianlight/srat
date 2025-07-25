import { configureStore } from "@reduxjs/toolkit";
import { setupListeners } from "@reduxjs/toolkit/query";
import { useDispatch, useSelector } from "react-redux";
import { errorSlice } from "./errorSlice";
import { sratApi } from "./sratApi";
import { sseSlice } from "./sseSlice";
//import { dirtySlice } from './dirtySlice'

export const store = configureStore({
	reducer: {
		//        dirty: dirtySlice.reducer,
		sse: sseSlice.reducer,
		errors: errorSlice.reducer,
		[sratApi.reducerPath]: sratApi.reducer,
	},
	middleware: (getDefaultMiddleware) =>
		getDefaultMiddleware().concat(sratApi.middleware),
	devTools: process.env.NODE_ENV !== "production",
});

setupListeners(store.dispatch);

// Infer the `RootState` and `AppDispatch` types from the store itself
export type RootState = ReturnType<typeof store.getState>;
// Inferred type: {errors: ErrorsState, posts: PostsState, comments: CommentsState, users: UsersState}
export type AppDispatch = typeof store.dispatch;

// Use throughout your app instead of plain `useDispatch` and `useSelector`
export const useAppDispatch = useDispatch.withTypes<AppDispatch>();
export const useAppSelector = useSelector.withTypes<RootState>();
//export const useAppStore = useStore.withTypes<store>();
