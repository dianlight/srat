import { useEffect, useState } from "react";
import {
	type Disk,
	useGetApiVolumesQuery,
} from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/sseApi";

export function useVolume() {
	const { data: evdata, error: everror, isLoading: evloading, fulfilledTimeStamp: evfulfilledTimeStamp } = useGetServerEventsQuery();
	const { data, error, isLoading, fulfilledTimeStamp } = useGetApiVolumesQuery();

	const [disks, setDisks] = useState<Array<Disk>>([]);

	useEffect(() => {
		if (!isLoading) {
			console.log("Update Data from REST API");
			setDisks(data as Disk[]);
		}
	}, [data, fulfilledTimeStamp]);

	useEffect(() => {
		if (!evloading && evdata?.volumes) {
			console.log("Update Data from SSE", evdata.volumes);
			setDisks(evdata.volumes);
		}
	}, [evdata, evfulfilledTimeStamp]);

	return { disks, isLoading: (isLoading && evloading), error: (error || everror) };
}
