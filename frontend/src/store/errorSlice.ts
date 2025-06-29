import type { PayloadAction } from "@reduxjs/toolkit";
import { createSlice } from "@reduxjs/toolkit";

export interface ErrorState {
	messages: string[];
}

const initialState: ErrorState = {
	messages: [],
};

export const errorSlice = createSlice({
	name: "errors",
	initialState,
	reducers: {
		addMessage: (state, action: PayloadAction<string>) => {
			state.messages.push(action.payload);
		},
		clearMessages: (state) => {
			state.messages = [];
		},
	},
});

// Action creators are generated for each case reducer function
export const { addMessage, clearMessages } = errorSlice.actions;

export default errorSlice.reducer;
