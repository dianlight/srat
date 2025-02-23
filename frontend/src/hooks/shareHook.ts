import { useEffect, useState } from "react";
import { DtoEventType, useGetSharesQuery, type DtoSharedResource } from "../store/sratApi";
import { useSSE } from "react-hooks-sse";

export function useShare() {

    const [shares, setShares] = useState<DtoSharedResource[]>([]);
    const { data, error, isLoading } = useGetSharesQuery();

    const statusSSE = useSSE(DtoEventType.Share, {} as DtoSharedResource, {
        parser(input: any): DtoSharedResource {
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