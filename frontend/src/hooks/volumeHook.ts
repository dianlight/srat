import { useEffect, useState } from "react";
import { Supported_events, useGetSharesQuery, useGetVolumesQuery, type Disk, type SharedResource } from "../store/sratApi";
import { useSSE } from "react-hooks-sse";

export function useVolume() {

    const [disks, setDisks] = useState<Disk[]>([] as Disk[]);
    const { data, error, isLoading } = useGetVolumesQuery();

    const statusSSE = useSSE(Supported_events.Volumes, [] as Disk[], {
        parser(input: any): Disk[] {
            const c = JSON.parse(input);
            console.log("Got disks", c)
            setDisks(c as Disk[]);
            return c;
        },
    });

    useEffect(() => {
        if (data) {
            setDisks(data as Disk[]);
        }
        if (error) {
            console.error("Error fetching volumes:", error);
        }
    }, [data]);

    return { disks: disks, isLoading, error };
}