import { Accordion, AccordionDetails, AccordionSummary, Typography, Box, Grid, useTheme } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { useHealth } from "../../hooks/healthHook";
import { useEffect, useMemo, useRef, useState } from "react";
import { useVolume } from "../../hooks/volumeHook";
import { MetricCard } from "./metrics/MetricCard";
import { ProcessMetrics } from "./metrics/ProcessMetrics";
import { DiskHealthMetrics } from "./metrics/DiskHealthMetrics";
import { NetworkHealthMetrics } from "./metrics/NetworkHealthMetrics";
import { VolumeMetrics } from "./metrics/VolumeMetrics";
import type { ProcessStatus, AddonStatsData } from "./metrics/types";
import { humanizeBytes, formatUptime } from "./metrics/utils";

const MAX_HISTORY_LENGTH = 10;


export function DashboardMetrics() {
    const { health, isLoading, error } = useHealth();
    const { disks, isLoading: isLoadingVolumes, error: errorVolumes } = useVolume();
    const theme = useTheme();
    const [connectionsHistory, setConnectionsHistory] = useState<Record<string, number[]>>({});
    const [cpuHistory, setCpuHistory] = useState<Record<string, number[]>>({});
    const [memoryHistory, setMemoryHistory] = useState<Record<string, number[]>>({});

    // New states for addon metrics
    const [addonCpuHistory, setAddonCpuHistory] = useState<number[]>([]);
    const [addonMemoryHistory, setAddonMemoryHistory] = useState<number[]>([]);
    const [addonDiskReadRateHistory, setAddonDiskReadRateHistory] = useState<number[]>([]);
    const [addonDiskWriteRateHistory, setAddonDiskWriteRateHistory] = useState<number[]>([]);
    const [addonNetworkRxRateHistory, setAddonNetworkRxRateHistory] = useState<number[]>([]);
    const [addonNetworkTxRateHistory, setAddonNetworkTxRateHistory] = useState<number[]>([]);
    const prevAddonStatsRef = useRef<AddonStatsData | null>(null);

    // New states for global metrics
    const [diskIopsHistory, setDiskIopsHistory] = useState<number[]>([]);
    const [networkTrafficHistory, setNetworkTrafficHistory] = useState<number[]>([]);

    const processData = useMemo((): ProcessStatus[] => {
        if (!health?.samba_process_status) {
            return [];
        }
        // The 'details' object from the health endpoint is expected to have cpu_percent and memory_usage.
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
        // Don't update history if the initial load is happening or there's an error
        if (isLoading || error || !health) {
            return;
        }

        if (health.samba_process_status) {
            setCpuHistory(prevHistory => {
                const newHistory = { ...prevHistory };
                for (const [name, details] of Object.entries(health.samba_process_status!)) {
                    const cpu = details?.cpu_percent ?? 0; // Default to 0 if null
                    const history = newHistory[name] ? [...newHistory[name]] : [];
                    history.push(cpu);
                    if (history.length > MAX_HISTORY_LENGTH) {
                        history.shift(); // Remove the oldest entry
                    }
                    newHistory[name] = history;
                }
                return newHistory;
            });

            setConnectionsHistory(prevHistory => {
                const newHistory = { ...prevHistory };
                for (const [name, details] of Object.entries(health.samba_process_status!)) {
                    const connections = details?.connections ?? 0; // Default to 0 if null
                    const history = newHistory[name] ? [...newHistory[name]] : [];
                    history.push(connections);
                    if (history.length > MAX_HISTORY_LENGTH) {
                        history.shift(); // Remove the oldest entry
                    }
                    newHistory[name] = history;
                }
                return newHistory;
            });

            setMemoryHistory(prevHistory => {
                const newHistory = { ...prevHistory };
                for (const [name, details] of Object.entries(health.samba_process_status!)) {
                    const memory = details?.memory_percent ?? 0; // Default to 0 if null
                    const history = newHistory[name] ? [...newHistory[name]] : [];
                    history.push(memory);
                    if (history.length > MAX_HISTORY_LENGTH) {
                        history.shift(); // Remove the oldest entry
                    }
                    newHistory[name] = history;
                }
                return newHistory;
            });
        }

        // Update addon stats history
        if (health.addon_stats) {
            const { addon_stats } = health;
            const intervalInSeconds = 5; // Assuming 5s refresh interval from backend

            // CPU and Memory % history
            setAddonCpuHistory(prev => {
                const newHistory = [...prev, addon_stats.cpu_percent ?? 0];
                if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
                return newHistory;
            });
            setAddonMemoryHistory(prev => {
                const newHistory = [...prev, addon_stats.memory_percent ?? 0];
                if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
                return newHistory;
            });

            // Disk and Network rate history
            if (prevAddonStatsRef.current) {
                const prevStats = prevAddonStatsRef.current;

                const calculateRate = (current?: number | null, prev?: number | null) => {
                    const delta = (current ?? 0) - (prev ?? 0);
                    return delta >= 0 ? delta / intervalInSeconds : 0;
                };

                const updateRateHistory = (setter: React.Dispatch<React.SetStateAction<number[]>>, current?: number | null, prev?: number | null) => {
                    setter(h => {
                        const rate = calculateRate(current, prev);
                        const newHistory = [...h, rate];
                        if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
                        return newHistory;
                    });
                };

                updateRateHistory(setAddonDiskReadRateHistory, addon_stats.blk_read, prevStats.blk_read);
                updateRateHistory(setAddonDiskWriteRateHistory, addon_stats.blk_write, prevStats.blk_write);
                updateRateHistory(setAddonNetworkRxRateHistory, addon_stats.network_rx, prevStats.network_rx);
                updateRateHistory(setAddonNetworkTxRateHistory, addon_stats.network_tx, prevStats.network_tx);
            }
            prevAddonStatsRef.current = addon_stats;
        }

        // Update global disk IOPS history
        if (health.disk_health?.global) {
            setDiskIopsHistory(prev => {
                const newHistory = [...prev, health.disk_health!.global.total_iops ?? 0];
                if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
                return newHistory;
            });
        }

        // Update global network traffic history
        if (health.network_health?.global) {
            const totalTraffic = (health.network_health.global.totalInboundTraffic ?? 0) + (health.network_health.global.totalOutboundTraffic ?? 0);
            setNetworkTrafficHistory(prev => {
                const newHistory = [...prev, totalTraffic];
                if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
                return newHistory;
            });
        }
    }, [health, isLoading, error]);



    const renderUptimeMetric = () => {
        return (
            <MetricCard
                title="Server Uptime"
                value={health?.startTime ? formatUptime(health.startTime) : 'N/A'}
                isLoading={isLoading}
                error={!!error || !health?.startTime}
            />
        );
    };

    const renderAddonCpuMetric = () => {
        return (
            <MetricCard
                title="Addon CPU"
                value={`${(health?.addon_stats?.cpu_percent ?? 0).toFixed(1)}%`}
                history={addonCpuHistory}
                isLoading={isLoading}
                error={!!error || !health?.addon_stats}
            />
        );
    };

    const renderAddonMemoryMetric = () => {
        const { memory_percent, memory_usage, memory_limit } = health?.addon_stats || {};
        return (
            <MetricCard
                title="Addon Memory"
                value={`${(memory_percent ?? 0).toFixed(1)}%`}
                history={addonMemoryHistory}
                isLoading={isLoading}
                error={!!error || !health?.addon_stats}
            >
                <Typography variant="body2" color="text.secondary" align="center">
                    {`${humanizeBytes(memory_usage ?? 0)} / ${humanizeBytes(memory_limit ?? 0)}`}
                </Typography>
            </MetricCard>
        );
    };

    const renderAddonDiskIoMetric = () => {
        const readRate = addonDiskReadRateHistory.length > 0 ? addonDiskReadRateHistory[addonDiskReadRateHistory.length - 1] : 0;
        const writeRate = addonDiskWriteRateHistory.length > 0 ? addonDiskWriteRateHistory[addonDiskWriteRateHistory.length - 1] : 0;
        const totalHistory = addonDiskReadRateHistory.map((r, i) => r + (addonDiskWriteRateHistory[i] ?? 0));

        return (
            <MetricCard
                title="Addon Disk I/O"
                subheader="per second"
                value={`${humanizeBytes(readRate + writeRate)}/s`}
                history={totalHistory}
                isLoading={isLoading}
                error={!!error || !health?.addon_stats}
                historyType='bar'
            >
                <Typography variant="body2">Read: {humanizeBytes(readRate)}/s</Typography>
                <Typography variant="body2">Write: {humanizeBytes(writeRate)}/s</Typography>
            </MetricCard>
        );
    };

    const renderAddonNetworkMetric = () => {
        const rxRate = addonNetworkRxRateHistory.length > 0 ? addonNetworkRxRateHistory[addonNetworkRxRateHistory.length - 1] : 0;
        const txRate = addonNetworkTxRateHistory.length > 0 ? addonNetworkTxRateHistory[addonNetworkTxRateHistory.length - 1] : 0;
        const totalHistory = addonNetworkRxRateHistory.map((r, i) => r + (addonNetworkTxRateHistory[i] ?? 0));

        return (
            <MetricCard
                title="Addon Network I/O"
                subheader="per second"
                value={`${humanizeBytes(rxRate + txRate)}/s`}
                history={totalHistory}
                isLoading={isLoading}
                error={!!error || !health?.addon_stats}
                historyType='bar'
            >
                <Typography variant="body2">Received: {humanizeBytes(rxRate)}/s</Typography>
                <Typography variant="body2">Sent: {humanizeBytes(txRate)}/s</Typography>
            </MetricCard>
        );
    };

    const renderGlobalDiskIoMetric = () => {
        const { total_read_latency_ms, total_write_latency_ms } = health?.disk_health?.global || {};
        return (
            <MetricCard
                title="Global Disk I/O"
                subheader="IOPS"
                value={`${(health?.disk_health?.global?.total_iops ?? 0).toFixed(1)}`}
                history={diskIopsHistory}
                isLoading={isLoading}
                error={!!error || !health?.disk_health?.global}
            >
                <Typography variant="body2" color="text.secondary" align="center">
                    Latency (r/w): {(total_read_latency_ms ?? 0).toFixed(2)}ms / {(total_write_latency_ms ?? 0).toFixed(2)}ms
                </Typography>
            </MetricCard>
        );
    };

    const renderGlobalNetworkIoMetric = () => {
        const { totalInboundTraffic, totalOutboundTraffic } = health?.network_health?.global || {};
        const totalTraffic = (totalInboundTraffic ?? 0) + (totalOutboundTraffic ?? 0);

        return (
            <MetricCard
                title="Global Network I/O"
                subheader="per second"
                value={`${humanizeBytes(totalTraffic)}/s`}
                history={networkTrafficHistory}
                isLoading={isLoading}
                error={!!error || !health?.network_health?.global}
            >
                <Typography variant="body2" color="text.secondary" align="center">
                    In: {humanizeBytes(totalInboundTraffic ?? 0)}/s | Out: {humanizeBytes(totalOutboundTraffic ?? 0)}/s
                </Typography>
            </MetricCard>
        );
    };







    return (
        <Accordion defaultExpanded>
            <AccordionSummary
                expandIcon={<ExpandMoreIcon />}
                aria-controls="panel-metrics-content"
                id="panel-metrics-header"
            >
                <Typography variant="h6">System Metrics</Typography>
            </AccordionSummary>
            <AccordionDetails>
                <Grid container spacing={3} sx={{ mb: 4 }}>
                    <Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
                        {renderUptimeMetric()}
                    </Grid>
                    <Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
                        {renderAddonCpuMetric()}
                    </Grid>
                    <Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
                        {renderAddonMemoryMetric()}
                    </Grid>
                    <Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
                        {renderAddonDiskIoMetric()}
                    </Grid>
                    <Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
                        {renderAddonNetworkMetric()}
                    </Grid>
                    <Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
                        {renderGlobalDiskIoMetric()}
                    </Grid>
                    <Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
                        {renderGlobalNetworkIoMetric()}
                    </Grid>
                </Grid>
                <ProcessMetrics
                    processData={processData}
                    cpuHistory={cpuHistory}
                    memoryHistory={memoryHistory}
                    connectionsHistory={connectionsHistory}
                />
                <DiskHealthMetrics diskHealth={health?.disk_health} />
                <NetworkHealthMetrics networkHealth={health?.network_health} />
                <Typography variant="h6" sx={{ mt: 4, mb: 2 }}>
                    Disk Usage
                </Typography>
                <VolumeMetrics disks={disks} isLoadingVolumes={isLoadingVolumes} errorVolumes={errorVolumes} />
            </AccordionDetails>
        </Accordion>
    );
}