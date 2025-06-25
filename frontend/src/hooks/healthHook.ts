import { useEffect, useState } from "react";
import { useGetHealthQuery, Supported_events, useGetVolumesQuery, type BlockInfo, type DataDirtyTracker, type HealthPing, type ReleaseAsset, type SambaProcessStatus, type SharedResource } from "../store/sratApi";
import { useSSE } from "react-hooks-sse";

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
        if (data && ((data as HealthPing).aliveTime || 0) > (health.aliveTime || 0)) {
            //console.log("Update Data from rest service", data);
            setHealth(data as HealthPing);
        }
        if (error) {
            console.error("Error fetching health data:", error);
        }
    }, [data]);

    useEffect(() => {
        if (ssedata && (ssedata.aliveTime || 0) > (health.aliveTime || 0)) {
            //console.log("Update Data from sse service", ssedata);
            setHealth(ssedata);
        }
    }, [ssedata]);

    return { health, isLoading, error };
}