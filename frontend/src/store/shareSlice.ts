import { createSlice } from '@reduxjs/toolkit'
import type { PayloadAction } from '@reduxjs/toolkit'
import type { DtoSharedResource } from '../srat'
import { apiContext } from '../Contexts'

export interface SharesState {
    value: DtoSharedResource[]
}

const initialState: SharesState = {
    value: [],
}

export const sharesSlice = createSlice({
    name: 'shares',
    initialState,
    reducers: {
        refresh: (state) => {
            apiContext.shares.sharesList().then((res) => {
                console.log("Got shares", res.data)
                state.value.push(...res.data);
            }).catch(err => {
                console.error(err);
                // Handle error state if needed
            })
        },
    },
})

// Action creators are generated for each case reducer function
export const { refresh } = sharesSlice.actions

export default sharesSlice.reducer