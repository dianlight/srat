import type { PayloadAction } from "@reduxjs/toolkit";
import { createSlice } from "@reduxjs/toolkit";

export interface MDCState {
	spanId: string | null;
	traceId: string | null;
}

const makeUUID = (): string => {
	if (typeof crypto !== "undefined") {
		const webCrypto = crypto as typeof crypto & { randomUUID?: () => string };
		if (typeof webCrypto.randomUUID === "function") {
			return webCrypto.randomUUID();
		}
		if (typeof crypto.getRandomValues === "function") {
			const bytes = new Uint8Array(16);
			crypto.getRandomValues(bytes);
			// Set version 4 and variant bits for UUID v4
			const versionByte = bytes[6];
			const variantByte = bytes[8];
			if (versionByte !== undefined && variantByte !== undefined) {
				bytes[6] = (versionByte & 0x0f) | 0x40; // version 4
				bytes[8] = (variantByte & 0x3f) | 0x80; // variant 10
			}
			const hex = Array.from(bytes, (b) =>
				b.toString(16).padStart(2, "0"),
			).join("");
			return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
		}
	}
	// Fallback (non-crypto)
	return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
		const r = (Math.random() * 16) | 0;
		const v = c === "x" ? r : (r & 0x3) | 0x8;
		return v.toString(16);
	});
};

const initialState: MDCState = {
	spanId: makeUUID(),
	traceId: makeUUID(),
};

export const mdcSlice = createSlice({
	name: "mdc",
	initialState,
	reducers: {
		setSpanId: (state, action: PayloadAction<string | null>) => {
			state.spanId = action.payload;
		},
		setTraceId: (state, action: PayloadAction<string | null>) => {
			state.traceId = action.payload;
		},
		setAllData: (state, action: PayloadAction<MDCState>) => {
			state.spanId = action.payload.spanId;
			state.traceId = action.payload.traceId;
		},
	},
});

// Action creators are generated for each case reducer function
export const { setSpanId, setTraceId } = mdcSlice.actions;

export default mdcSlice.reducer;
