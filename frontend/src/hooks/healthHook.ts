import { useEffect, useState } from "react";
import { DtoEventType, useGetHealthQuery, useGetSharesQuery, useGetVolumesQuery, type DtoBlockInfo, type DtoHealthPing, type DtoSharedResource } from "../store/sratApi";
import { useSSE } from "react-hooks-sse";

export function useHealth() {

    const [health, setHealth] = useState<DtoHealthPing>({} as DtoHealthPing);
    const { data, error, isLoading } = useGetHealthQuery();

    useSSE(DtoEventType.Heartbeat, {} as DtoHealthPing, {
        parser(input: any): DtoHealthPing {
            console.log("Got health data", input);
            const c = JSON.parse(input);
            setHealth(c);
            return c;
        },
    });

    useEffect(() => {
        if (data) {
            setHealth(data);
        }
        if (error) {
            console.error("Error fetching health data:", error);
        }
    }, [data]);

    return { health, isLoading, error };
}