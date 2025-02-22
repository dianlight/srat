import { useEffect, useState } from "react";
import { DtoEventType, useGetHealthQuery, useGetSharesQuery, useGetVolumesQuery, type DtoBlockInfo, type DtoDataDirtyTracker, type DtoHealthPing, type DtoReleaseAsset, type DtoSambaProcessStatus, type DtoSharedResource } from "../store/sratApi";
import { useSSE } from "react-hooks-sse";

export function useHealth() {

    const [health, setHealth] = useState<DtoHealthPing>({
        alive: false,
        read_only: true,
        aliveTime: 0,
        dirty_tracking: {} as DtoDataDirtyTracker,
        last_error: "",
        last_release: {} as DtoReleaseAsset,
        samba_process_status: {} as DtoSambaProcessStatus
    });

    const { data, error, isLoading } = useGetHealthQuery();
    const ssedata = useSSE(DtoEventType.Heartbeat, {} as DtoHealthPing, {
        parser(input: any): DtoHealthPing {
            const c = JSON.parse(input);
            console.log("Got sse health data", c);
            return c;
        },
    });

    useEffect(() => {
        if (data && (data.aliveTime || 0) > (health.aliveTime || 0)) {
            console.log("Update Data from rest service", data);
            setHealth(data);
        }
        if (error) {
            console.error("Error fetching health data:", error);
        }
    }, [data]);

    useEffect(() => {
        if (ssedata && (ssedata.aliveTime || 0) > (health.aliveTime || 0)) {
            console.log("Update Data from sse service", ssedata);
            setHealth(ssedata);
        }
    }, [ssedata]);

    return { health, isLoading, error };
}