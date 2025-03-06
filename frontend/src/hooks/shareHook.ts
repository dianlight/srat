import { useEffect, useState } from "react";
import { EventType, useGetSharesQuery, type SharedResource } from "../store/sratApi";
import { useSSE } from "react-hooks-sse";

export function useShare() {

    const [shares, setShares] = useState<SharedResource[]>([]);
    const { data, error, isLoading } = useGetSharesQuery();

    const statusSSE = useSSE(EventType.Share, {} as SharedResource, {
        parser(input: any): SharedResource {
            const c = JSON.parse(input);
            console.log("Got shares", c)
            setShares(c)
            return c;
        },
    });

    useEffect(() => {
        if (data) {
            setShares(data);
        }
        if (error) {
            console.error("Error fetching shares:", error);
        }
    }, [data]);

    return { shares, isLoading, error };
}