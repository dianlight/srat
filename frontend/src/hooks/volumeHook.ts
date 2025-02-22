import { useEffect, useState } from "react";
import { DtoEventType, useGetSharesQuery, useGetVolumesQuery, type DtoBlockInfo, type DtoSharedResource } from "../store/sratApi";
import { useSSE } from "react-hooks-sse";

export function useVolume() {

    const [volumes, setVolumes] = useState<DtoBlockInfo>({} as DtoBlockInfo);
    const { data, error, isLoading } = useGetVolumesQuery();

    const statusSSE = useSSE(DtoEventType.Volumes, {} as DtoBlockInfo, {
        parser(input: any): DtoBlockInfo {
            console.log("Got volumes", input)
            const c = JSON.parse(input);
            setVolumes(c);
            return c;
        },
    });

    useEffect(() => {
        if (data) {
            setVolumes(data);
        }
        if (error) {
            console.error("Error fetching volumes:", error);
        }
    }, [data]);

    return { volumes, isLoading, error };
}