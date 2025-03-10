import { useEffect, useState } from "react";
import { Supported_events, useGetSharesQuery, useGetVolumesQuery, type BlockInfo, type SharedResource } from "../store/sratApi";
import { useSSE } from "react-hooks-sse";

export function useVolume() {

    const [volumes, setVolumes] = useState<BlockInfo>({} as BlockInfo);
    const { data, error, isLoading } = useGetVolumesQuery();

    const statusSSE = useSSE(Supported_events.Volumes, {} as BlockInfo, {
        parser(input: any): BlockInfo {
            console.log("Got volumes", input)
            const c = JSON.parse(input);
            setVolumes(c);
            return c;
        },
    });

    useEffect(() => {
        if (data) {
            setVolumes(data as BlockInfo);
        }
        if (error) {
            console.error("Error fetching volumes:", error);
        }
    }, [data]);

    return { volumes, isLoading, error };
}