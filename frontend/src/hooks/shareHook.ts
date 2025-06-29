import { useEffect } from "react";
import { useSSE } from "react-hooks-sse";
import {
	type SharedResource,
	Supported_events,
	useGetSharesQuery,
} from "../store/sratApi";
import { setShares } from "../store/sseSlice";
import { useAppDispatch, useAppSelector } from "../store/store";

let shareHook_lastReadTimestamp = 0;

export function useShare() {
	const dispatch = useAppDispatch();
	const shares = useAppSelector((state) => state.sse.shares);

	const { data, error, isLoading, fulfilledTimeStamp } = useGetSharesQuery();

	// statusSSE variable is not directly used, but useSSE hook initializes the SSE connection
	// and its parser handles data dispatching.
	useSSE(Supported_events.Share, [] as SharedResource[], {
		parser(input: any): SharedResource[] {
			const c = JSON.parse(input);
			// Assuming 'c' is SharedResource[] as per API spec for 'share' event
			console.log("Got shares from SSE", c);
			dispatch(setShares(c as SharedResource[]));
			shareHook_lastReadTimestamp = Date.now();
			return c;
		},
	});

	useEffect(() => {
		if (
			data &&
			fulfilledTimeStamp &&
			shareHook_lastReadTimestamp < fulfilledTimeStamp
		) {
			console.log(
				"Update Shares from REST service",
				data,
				fulfilledTimeStamp,
				shareHook_lastReadTimestamp,
			);
			// Data from GetSharesApiResponse is SharedResource[] | null
			dispatch(setShares(data as SharedResource[]));
			shareHook_lastReadTimestamp = fulfilledTimeStamp;
		}
		if (error) {
			console.error("Error fetching shares:", error);
		}
	}, [data, fulfilledTimeStamp, dispatch, error]);

	return { shares, isLoading, error };
}
