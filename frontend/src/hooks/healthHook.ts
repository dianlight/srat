import { useEffect, useState } from "react";
import { useSSE } from "react-hooks-sse";
import {
	type DataDirtyTracker,
	type DiskHealth,
	type HealthPing,
	type NetworkStats,
	type ReleaseAsset,
	type SambaProcessStatus,
	type SambaStatus,
	Supported_events,
	useGetHealthQuery,
} from "../store/sratApi";

export function useHealth() {
	const [health, setHealth] = useState<HealthPing>({
		alive: false,
		read_only: true,
		aliveTime: 0,
		startTime: 0,
		dirty_tracking: {} as DataDirtyTracker,
		last_error: "",
		last_release: {} as ReleaseAsset,
		samba_process_status: {} as SambaProcessStatus,
		secure_mode: false,
		addon_stats: {},
		build_version: "",
		protected_mode: false,
		disk_health: {} as DiskHealth,
		network_health: {} as NetworkStats,
		samba_status: {} as SambaStatus,
	});

	const { data, error, isLoading } = useGetHealthQuery();
	const ssedata = useSSE(Supported_events.Heartbeat, {} as HealthPing, {
		parser(input: any): HealthPing {
			const c = JSON.parse(input);
			//console.log("Got sse health data", c);
			return c;
		},
	});

	useEffect(() => {
		if (
			data &&
			((data as HealthPing).aliveTime || 0) > (health.aliveTime || 0)
		) {
			//console.log("Update Data from rest service", data);
			setHealth(data as HealthPing);
		}
		if (error) {
			console.error("Error fetching health data:", error);
		}
	}, [data, error, health.aliveTime]);

	useEffect(() => {
		if (ssedata && (ssedata.aliveTime || 0) > (health.aliveTime || 0)) {
			//console.log("Update Data from sse service", ssedata);
			setHealth(ssedata);
		}
	}, [ssedata, health.aliveTime]);

	return { health, isLoading, error };
}
