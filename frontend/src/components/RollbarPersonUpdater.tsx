import { useRollbarPerson } from "@rollbar/react";
import { useHealth } from "../hooks/healthHook";

/**
 * Component that updates Rollbar person information when machine_id becomes available
 */
export const RollbarPersonUpdater: React.FC = () => {
    const { health } = useHealth();

    // Pass the person object directly to the hook
    // The hook will automatically update when health.machine_id changes
    const personData = health.machine_id ? { id: health.machine_id } : {};
    useRollbarPerson(personData);

    // This component renders nothing
    return null;
};
