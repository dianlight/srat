import { useHealth } from "./healthHook";

export function useReadOnly() {
    const mode = useHealth();
    return mode.health?.read_only || false;
}