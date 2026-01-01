import { useEffect, useState } from "react";
import {
	Update_process_state,
	type UpdateProgress,
	useGetApiUpdateQuery,
} from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/sseApi";

export function useUpdate() {
	const { data, error, isLoading, isSuccess, refetch } = useGetApiUpdateQuery();
	const {
		data: evdata,
		error: everror,
		isLoading: evloading,
	} = useGetServerEventsQuery();

	const [update, setUpdate] = useState<{
		Available: boolean;
		Progress: UpdateProgress;
	}>({
		Available: false,
		Progress: {
			progress: 0,
			update_process_state: Update_process_state.Idle,
		},
	});

	useEffect(() => {
		if (!isLoading && isSuccess && data) {
			setUpdate({ Available: true, Progress: data as UpdateProgress });
		}
	}, [data, isLoading, isSuccess]);

	useEffect(() => {
		if (!evloading && evdata?.updating) {
			setUpdate((prev) => ({
				...prev,
				Progress: evdata.updating,
			}));
		} else if (
			!evloading &&
			evdata?.heartbeat &&
			evdata.heartbeat.update_available !== undefined
		) {
			setUpdate((prev) => ({
				Available: evdata.heartbeat.update_available,
				Progress: prev.Progress,
			}));
			if (evdata.heartbeat.update_available) {
				refetch();
			}
		}
	}, [evdata, evloading, refetch]);

	return {
		update: update,
		isLoading: isLoading && evloading,
		error: error || everror,
	};
}
