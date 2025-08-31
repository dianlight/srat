import { useEffect, useState } from "react";
import { type SharedResource, useGetApiSharesQuery } from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/sseApi";

export function useShare() {
	const { data, error, isLoading } = useGetApiSharesQuery();
	const {
		data: evdata,
		error: everror,
		isLoading: evloading,
	} = useGetServerEventsQuery();

	const [shares, setShares] = useState<Array<SharedResource>>([]);

	useEffect(() => {
		if (!isLoading) {
			//console.log("Update Shares Data from REST API");
			setShares(data as SharedResource[]);
		}
	}, [data, isLoading]);

	useEffect(() => {
		if (!evloading && evdata?.share) {
			//console.log("Update Shares Data from SSE", evdata.share);
			setShares(evdata.share);
		}
	}, [evdata, evloading]);

	return {
		shares: shares,
		isLoading: isLoading && evloading,
		error: error || everror,
	};
}
