import { useEffect } from "react";
import { useSSE } from "react-hooks-sse";
import {
	type Disk,
	Supported_events,
	useGetApiVolumesQuery,
} from "../store/sratApi";
import { setDisks } from "../store/sseSlice";
import { useAppDispatch, useAppSelector } from "../store/store";

let useVolume_lastReadTimestamp = 0;
export function useVolume() {
	const dispatch = useAppDispatch();
	const disks = useAppSelector((state) => state.sse.disks);

	const { data, error, isLoading, fulfilledTimeStamp } =
		useGetApiVolumesQuery();

	const _statusSSE = useSSE(Supported_events.Volumes, [] as Disk[], {
		parser(input: string): Disk[] {
			const c = JSON.parse(input);
			//console.log("Got disks", c);
			dispatch(setDisks(c as Disk[]));
			useVolume_lastReadTimestamp = Date.now();
			return c;
		},
	});

	useEffect(() => {
		if (
			data &&
			fulfilledTimeStamp &&
			useVolume_lastReadTimestamp < fulfilledTimeStamp
		) {
			/* 		console.trace(
						"Update Data from rest service",
						data,
						fulfilledTimeStamp,
						useVolume_lastReadTimestamp,
					); */
			dispatch(setDisks(data as Disk[]));
			useVolume_lastReadTimestamp = fulfilledTimeStamp;
		}
		if (error) {
			console.error("Error fetching volumes:", error);
		}
	}, [data, dispatch, error, fulfilledTimeStamp]);

	return { disks, isLoading, error };
}
