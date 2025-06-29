/*
import { createSlice } from '@reduxjs/toolkit'
import type { PayloadAction } from '@reduxjs/toolkit'

export interface DirtyState {
    shares: boolean,
    volumes: boolean,
    users: boolean,
    configs: boolean,
}

const initialState: DirtyState = {
    shares: false,
    volumes: false,
    users: false,
    configs: false,
}

export const dirtySlice = createSlice({
    name: 'dirty',
    initialState,
    reducers: {
        setDirty: (state, action: PayloadAction<keyof DirtyState>) => {
            state[action.payload] = true;
        },
        clearDirty: (state, action: PayloadAction<keyof DirtyState>) => {
            state[action.payload] = false;
        },
    },
})

// Action creators are generated for each case reducer function
export const { setDirty, clearDirty } = dirtySlice.actions

export default dirtySlice.reducer
*/
export {};
