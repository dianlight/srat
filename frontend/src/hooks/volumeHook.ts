import { useEffect, useState } from "react";
import { Supported_events, useGetSharesQuery, useGetVolumesQuery, type Disk, type SharedResource } from "../store/sratApi";
import { useSSE } from "react-hooks-sse";
import { useAppDispatch, useAppSelector } from "../store/store";
import { setDisks } from "../store/sseSlice";

let useVolume_lastReadTimestamp = 0;
export function useVolume() {

    const dispatch = useAppDispatch();
    const disks = useAppSelector((state) => state.sse.disks)

    const { data, error, isLoading, fulfilledTimeStamp } = useGetVolumesQuery();

    const statusSSE = useSSE(Supported_events.Volumes, [] as Disk[], {
        parser(input: any): Disk[] {
            const c = JSON.parse(input);
            console.log("Got disks", c)
            dispatch(setDisks(c as Disk[]));
            useVolume_lastReadTimestamp = Date.now();
            return c;
        },
    });

    useEffect(() => {
        if (data && fulfilledTimeStamp && useVolume_lastReadTimestamp < fulfilledTimeStamp) {
            console.log("Update Data from rest service", data, fulfilledTimeStamp, useVolume_lastReadTimestamp);
            dispatch(setDisks(data as Disk[]));
            useVolume_lastReadTimestamp = fulfilledTimeStamp;
        }
        if (error) {
            console.error("Error fetching volumes:", error);
        }
    }, [data]);

    return { disks, isLoading, error };
}