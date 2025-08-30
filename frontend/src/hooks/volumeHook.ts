import { useEffect, useState } from "react";
import { type Disk, useGetApiVolumesQuery } from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/sseApi";

export function useVolume() {
	const {
		data: evdata,
		error: everror,
		isLoading: evloading,
	} = useGetServerEventsQuery();
	const { data, error, isLoading } = useGetApiVolumesQuery();

	const [disks, setDisks] = useState<Array<Disk>>([]);

	useEffect(() => {
		if (!isLoading) {
			console.log("Update Data from REST API");
			setDisks(data as Disk[]);
		}
	}, [data, isLoading]);

	useEffect(() => {
		if (!evloading && evdata?.volumes) {
			console.log("Update Data from SSE", evdata.volumes);
			setDisks(evdata.volumes);
		}
	}, [evdata?.volumes, evloading]);

	return { disks, isLoading: isLoading && evloading, error: error || everror };
}
