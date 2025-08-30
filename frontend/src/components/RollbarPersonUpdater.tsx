import { useRollbarPerson } from "@rollbar/react";
import { useHealth } from "../hooks/healthHook";
import { useGetServerEventsQuery } from "../store/sseApi";
import { useEffect } from "react";

/**
 * Component that updates Rollbar person information when machine_id becomes available
 */
export const RollbarPersonUpdater: React.FC = () => {
    const { data, isLoading } = useGetServerEventsQuery();


    useRollbarPerson({ id: data?.hello?.machine_id });

    // This component renders nothing
    return null;
};
