import { useEffect, useState } from "react";
import {
	type SmartTestStatus,
	useGetApiDiskByDiskIdSmartTestQuery,
} from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/sseApi";

export function useSmartTestStatus(diskId: string) {
	const { data, error, isLoading, isSuccess, refetch } =
		useGetApiDiskByDiskIdSmartTestQuery({ diskId: diskId });
	const {
		data: evdata,
		error: everror,
		isLoading: evloading,
	} = useGetServerEventsQuery();

	const [smartTestStatus, setSmartTestStatus] = useState<SmartTestStatus>({
		disk_id: diskId,
		running: false,
		status: "idle",
		percent_complete: 0,
		test_type: "none",
	});

	useEffect(() => {
		if (!isLoading && isSuccess && data) {
			//console.log("Update data:", data);
			setSmartTestStatus(data as SmartTestStatus);
		}
	}, [data, isLoading, isSuccess]);

	useEffect(() => {
		if (!evloading && evdata?.updating) {
			setSmartTestStatus((prev) => ({
				...prev,
				Progress: evdata.updating,
			}));
		} else if (
			!evloading &&
			evdata?.smart_test_status &&
			evdata.smart_test_status.disk_id === diskId
		) {
			setSmartTestStatus(evdata.smart_test_status);
			if (evdata.smart_test_status.percent_complete === 100) {
				setTimeout(() => {
					refetch();
				}, 5000);
			}
		} else if (!evloading && everror) {
			console.error("Error receiving smart test status via SSE:", everror);
		} else if (!evloading && evdata?.smart_test_status) {
			console.log(
				"Received smart test status for different disk:",
				evdata.smart_test_status,
			);
		}
	}, [evdata, evloading, diskId, everror, refetch]);

	return {
		smartTestStatus: smartTestStatus,
		isLoading: isLoading && evloading,
		error: error || everror,
	};
}
