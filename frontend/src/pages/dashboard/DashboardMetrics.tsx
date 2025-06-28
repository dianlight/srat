import { useHealth } from "../../hooks/healthHook";
import { useVolume } from "../../hooks/volumeHook";
import { useMemo, useState, useEffect } from "react";
import { SystemMetricsAccordion } from "./metrics/SystemMetricsAccordion";
import { ProcessMetricsAccordion } from "./metrics/ProcessMetricsAccordion";
import { DiskHealthMetricsAccordion } from "./metrics/DiskHealthMetricsAccordion";
import { NetworkHealthMetricsAccordion } from "./metrics/NetworkHealthMetricsAccordion";
import { VolumeMetricsAccordion } from "./metrics/VolumeMetricsAccordion";
import type { ProcessStatus } from "./metrics/types";

const MAX_HISTORY_LENGTH = 10;

export function DashboardMetrics() {
    const { health, isLoading, error } = useHealth();
    //const { disks, isLoading: isLoadingVolumes, error: errorVolumes } = useVolume();

    const [connectionsHistory, setConnectionsHistory] = useState<Record<string, number[]>>({});
    const [cpuHistory, setCpuHistory] = useState<Record<string, number[]>>({});
    const [memoryHistory, setMemoryHistory] = useState<Record<string, number[]>>({});

    const processData = useMemo((): ProcessStatus[] => {
        if (!health?.samba_process_status) {
            return [];
        }
        return Object.entries(health.samba_process_status).map(([name, details]) => {
            return {
                name,
                pid: details?.pid || null,
                status: details?.pid ? 'Running' : 'Stopped',
                cpu: details?.cpu_percent ?? null,
                connections: details?.connections ?? null,
                memory: details?.memory_percent ?? null,
            };
        });
    }, [health]);

    useEffect(() => {
        if (isLoading || error || !health) {
            return;
        }

        if (health.samba_process_status) {
            setCpuHistory(prevHistory => {
                const newHistory = { ...prevHistory };
                for (const [name, details] of Object.entries(health.samba_process_status!)) {
                    const cpu = details?.cpu_percent ?? 0;
                    const history = newHistory[name] ? [...newHistory[name]] : [];
                    history.push(cpu);
                    if (history.length > MAX_HISTORY_LENGTH) {
                        history.shift();
                    }
                    newHistory[name] = history;
                }
                return newHistory;
            });

            setConnectionsHistory(prevHistory => {
                const newHistory = { ...prevHistory };
                for (const [name, details] of Object.entries(health.samba_process_status!)) {
                    const connections = details?.connections ?? 0;
                    const history = newHistory[name] ? [...newHistory[name]] : [];
                    history.push(connections);
                    if (history.length > MAX_HISTORY_LENGTH) {
                        history.shift();
                    }
                    newHistory[name] = history;
                }
                return newHistory;
            });

            setMemoryHistory(prevHistory => {
                const newHistory = { ...prevHistory };
                for (const [name, details] of Object.entries(health.samba_process_status!)) {
                    const memory = details?.memory_percent ?? 0;
                    const history = newHistory[name] ? [...newHistory[name]] : [];
                    history.push(memory);
                    if (history.length > MAX_HISTORY_LENGTH) {
                        history.shift();
                    }
                    newHistory[name] = history;
                }
                return newHistory;
            });
        }
    }, [health, isLoading, error]);

    return (
        <>
            <SystemMetricsAccordion health={health} isLoading={isLoading} error={error} />
            <ProcessMetricsAccordion
                processData={processData}
                cpuHistory={cpuHistory}
                memoryHistory={memoryHistory}
                connectionsHistory={connectionsHistory}
            />
            <DiskHealthMetricsAccordion diskHealth={health?.disk_health} />
            <NetworkHealthMetricsAccordion networkHealth={health?.network_health} />
            <VolumeMetricsAccordion diskHealth={health?.disk_health} />
        </>
    );
}
