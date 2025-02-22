import { configureStore } from '@reduxjs/toolkit'
import { errorSlice } from './errorSlice'
import { sratApi } from './sratApi'
import { setupListeners } from '@reduxjs/toolkit/query'
import { useDispatch, useSelector, useStore } from 'react-redux'
import { dirtySlice } from './dirtySlice'

export const store = configureStore({
    reducer: {
        //        dirty: dirtySlice.reducer,
        errors: errorSlice.reducer,
        [sratApi.reducerPath]: sratApi.reducer,
    },
    middleware: (getDefaultMiddleware) =>
        getDefaultMiddleware().concat(sratApi.middleware),
})

setupListeners(store.dispatch)

// Infer the `RootState` and `AppDispatch` types from the store itself
export type RootState = ReturnType<typeof store.getState>
// Inferred type: {errors: ErrorsState, posts: PostsState, comments: CommentsState, users: UsersState}
export type AppDispatch = typeof store.dispatch

// Use throughout your app instead of plain `useDispatch` and `useSelector`
export const useAppDispatch = useDispatch.withTypes<AppDispatch>();
export const useAppSelector = useSelector.withTypes<RootState>();
//export const useAppStore = useStore.withTypes<store>();