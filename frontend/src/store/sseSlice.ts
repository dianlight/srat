import { createSlice } from '@reduxjs/toolkit'
import type { PayloadAction } from '@reduxjs/toolkit'
import type { Disk } from './sratApi'

export interface SSEState {
    disks: Disk[]
}

const initialState: SSEState = {
    disks: [] as Disk[]
}

export const sseSlice = createSlice({
    name: 'sse',
    initialState,
    reducers: {
        setDisks: (state, action: PayloadAction<Disk[]>) => {
            state.disks = action.payload;
        },
    },
})

// Action creators are generated for each case reducer function
export const { setDisks } = sseSlice.actions

export default sseSlice.reducer