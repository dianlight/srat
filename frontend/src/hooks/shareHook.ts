import { useEffect, useMemo, useState } from "react";
import { type SharedResource, useGetApiSharesQuery } from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/wsApi";

function toRecords(shares: SharedResource[]): Record<string, SharedResource> {
  const result: Record<string, SharedResource> = {};
  for (const share of shares) {
    const key = share.name;
    if (key) {
      result[key] = share;
    }
  }
  return result;
}

export function useShare() {
  const { data, error, isLoading } = useGetApiSharesQuery();
  const {
    data: evdata,
    error: everror,
    isLoading: evloading,
  } = useGetServerEventsQuery();

  const [shares, setShares] = useState<SharedResource[]>([]);

  useEffect(() => {
    if (!isLoading && data) {
      setShares(data as SharedResource[]);
    }
  }, [data, isLoading]);

  useEffect(() => {
    if (!evloading && evdata?.shares) {
      setShares(evdata.shares);
    }
  }, [evdata, evloading]);

  const shareRecords = useMemo(() => toRecords(shares), [shares]);

  return {
    shares: shareRecords,
    isLoading: isLoading && evloading,
    error: error || everror,
  };
}
