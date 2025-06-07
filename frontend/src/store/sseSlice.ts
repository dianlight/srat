import { createSlice } from '@reduxjs/toolkit'
import type { PayloadAction } from '@reduxjs/toolkit'
import type { Disk, SharedResource } from './sratApi'

export interface SSEState {
    disks: Disk[]
    shares: SharedResource[]
}

const initialState: SSEState = {
    disks: [] as Disk[],
    shares: [] as SharedResource[]
}

export const sseSlice = createSlice({
    name: 'sse',
    initialState,
    reducers: {
        setDisks: (state, action: PayloadAction<Disk[]>) => {
            state.disks = action.payload;
        },
        setShares: (state, action: PayloadAction<SharedResource[]>) => {
            state.shares = action.payload;
        },
    },
})

// Action creators are generated for each case reducer function
export const { setDisks, setShares } = sseSlice.actions

export default sseSlice.reducer