import { useEffect, useState } from "react";
import {
	type DataDirtyTracker,
	type DiskHealth,
	type HealthPing,
	type NetworkStats,
	type ProcessStatus,
	type SambaStatus,
	useGetApiHealthQuery,
} from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/sseApi";

export function useHealth() {
	const [health, setHealth] = useState<HealthPing>({
		alive: false,
		aliveTime: 0,
		dirty_tracking: {} as DataDirtyTracker,
		last_error: "",
		update_available: false,
		samba_process_status: {} as { [key: string]: ProcessStatus },
		addon_stats: {},
		disk_health: {} as DiskHealth,
		network_health: {} as NetworkStats,
		samba_status: {} as SambaStatus,
		uptime: 0,
	});
	const {
		data: evdata,
		error: everror,
		isLoading: evloading,
	} = useGetServerEventsQuery();
	const { data, error, isLoading } = useGetApiHealthQuery();

	useEffect(() => {
		if (!isLoading) {
			//console.log("Update Healt Data from REST API");
			setHealth(data as HealthPing);
		}
	}, [data, isLoading]);

	useEffect(() => {
		if (!evloading && evdata?.heartbeat) {
			//console.log("Update Healt Data from SSE", evdata.heartbeat);
			setHealth(evdata.heartbeat);
		}
	}, [evdata?.heartbeat, evloading]);

	return { health, isLoading: isLoading && evloading, error: error || everror };
}
