import { Accordion, AccordionDetails, AccordionSummary, Typography, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Box, CircularProgress, Alert, useTheme, Grid, Card, CardHeader, CardContent } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { useHealth } from "../../hooks/healthHook";
import { useEffect, useMemo, useRef, useState } from "react";
import { Sparklines, SparklinesBars, SparklinesLine, SparklinesSpots } from 'react-sparklines';
import { useVolume } from "../../hooks/volumeHook";
import { PieChart } from '@mui/x-charts/PieChart';

interface ProcessStatus {
    name: string;
    pid: number | null;
    status: 'Running' | 'Stopped';
    cpu: number | null;
    connections: number | null;
    memory: number | null;
}

interface AddonStatsData {
    blk_read?: number | null;
    blk_write?: number | null;
    cpu_percent?: number | null;
    memory_limit?: number | null;
    memory_percent?: number | null;
    memory_usage?: number | null;
    network_rx?: number | null;
    network_tx?: number | null;
}

function humanizeBytes(bytes: number): string {
    if (bytes <= 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
};

const MAX_HISTORY_LENGTH = 10;

// A simplified version of what's in Volumes.tsx
function decodeEscapeSequence(source: string) {
    if (typeof source !== 'string') return '';
    return source.replace(/\\x([0-9A-Fa-f]{2})/g, function (_match, group1) {
        return String.fromCharCode(parseInt(String(group1), 16));
    });
};

function formatUptime(millis: number): string {
    let seconds = Math.floor((Date.now() - millis) / 1000);
    if (seconds <= 0) return '0 seconds';

    const days = Math.floor(seconds / (24 * 3600));
    seconds %= (24 * 3600);
    const hours = Math.floor(seconds / 3600);
    seconds %= 3600;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = Math.floor(seconds % 60);

    const parts = [];
    if (days > 0) parts.push(`${days}d`);
    if (hours > 0) parts.push(`${hours}h`);
    if (minutes > 0) parts.push(`${minutes}m`);
    if (remainingSeconds > 0 || parts.length === 0) parts.push(`${remainingSeconds}s`);

    return parts.join(' ');
}

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
    }, [health, isLoading, error]);

    const renderProcessMetrics = () => {
        if (isLoading) {
            return (
                <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                    <CircularProgress />
                </Box>
            );
        }

        if (error) {
            return <Alert severity="error">Could not load system metrics.</Alert>;
        }

        return (
            <>
                <Typography variant="body1" sx={{ mb: 2 }}>
                    Overview of Samba-related processes.
                </Typography>
                <TableContainer component={Paper}>
                    <Table aria-label="samba processes table" size="small">
                        <TableHead>
                            <TableRow>
                                <TableCell>Process</TableCell>
                                <TableCell align="right">Status</TableCell>
                                <TableCell align="right">PID</TableCell>
                                <TableCell align="right">CPU (%)</TableCell>
                                <TableCell align="right">Memory (%)</TableCell>
                                <TableCell align="right">Connections</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {processData.map((process) => (
                                <TableRow key={process.name}>
                                    <TableCell component="th" scope="row">
                                        {process.name}
                                    </TableCell>
                                    <TableCell align="right" sx={{ color: process.status === 'Running' ? 'success.main' : 'error.main' }}>
                                        {process.status}
                                    </TableCell>
                                    <TableCell align="right">{process.pid ?? 'N/A'}</TableCell>
                                    <TableCell align="right" sx={{ minWidth: 150 }}>
                                        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end' }}>
                                            <Typography variant="body2" sx={{ mr: 1, minWidth: '45px', textAlign: 'right' }}>
                                                {process.cpu !== null ? `${process.cpu.toFixed(1)}%` : 'N/A'}
                                            </Typography>
                                            <Box sx={{ width: 50, height: 20 }}>
                                                {(cpuHistory[process.name]?.length || 0) > 1 ? (
                                                    <Sparklines data={cpuHistory[process.name]} limit={MAX_HISTORY_LENGTH} width={60} height={20}>
                                                        <SparklinesLine color={theme.palette.primary.main} />
                                                        <SparklinesSpots />
                                                    </Sparklines>
                                                ) : null}
                                            </Box>
                                        </Box>
                                    </TableCell>
                                    <TableCell align="right" sx={{ minWidth: 150 }}>
                                        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end' }}>
                                            <Typography variant="body2" sx={{ mr: 1, minWidth: '70px', textAlign: 'right' }}>
                                                {process.memory !== null ? `${process.memory.toFixed(1)}%` : 'N/A'}
                                            </Typography>
                                            <Box sx={{ width: 50, height: 20 }}>
                                                {(memoryHistory[process.name]?.length || 0) > 1 ? (
                                                    <Sparklines data={memoryHistory[process.name]} limit={MAX_HISTORY_LENGTH} width={60} height={20}>
                                                        <SparklinesLine color={theme.palette.success.main} />
                                                        <SparklinesSpots />
                                                    </Sparklines>
                                                ) : null}
                                            </Box>
                                        </Box>
                                    </TableCell>
                                    <TableCell align="right" sx={{ minWidth: 150 }}>
                                        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end' }}>
                                            <Typography variant="body2" sx={{ mr: 1, minWidth: '45px', textAlign: 'right' }}>
                                                {process.connections ?? 'N/A'}
                                            </Typography>
                                            <Box sx={{ width: 50, height: 20 }}>
                                                {(connectionsHistory[process.name]?.length || 0) > 1 ? (
                                                    <Sparklines data={connectionsHistory[process.name]} limit={MAX_HISTORY_LENGTH} width={60} height={20}>
                                                        <SparklinesBars style={{ fill: "#41c3f9", fillOpacity: ".25" }} />
                                                        <SparklinesLine color={theme.palette.secondary.main} />
                                                    </Sparklines>
                                                ) : null}
                                            </Box>
                                        </Box>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </TableContainer>
            </>
        );
    };

    const renderUptimeMetric = () => {
        if (isLoading) {
            return (
                <Card>
                    <CardHeader title="System Uptime" />
                    <CardContent sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                        <CircularProgress />
                    </CardContent>
                </Card>
            );
        }

        if (error || !health?.startTime) {
            return (
                <Card>
                    <CardHeader title="System Uptime" />
                    <CardContent>
                        <Alert severity="warning">Uptime data not available.</Alert>
                    </CardContent>
                </Card>
            );
        }

        return (
            <Card>
                <CardHeader title="System Uptime" />
                <CardContent>
                    <Typography variant="h4" component="div" align="center">
                        {/* Calculate uptime duration: current time - startTime (last start timestamp) */}
                        {formatUptime(health.startTime)}
                    </Typography>
                </CardContent>
            </Card>
        );
    };

    const renderAddonCpuMetric = () => {
        if (isLoading) {
            return (
                <Card>
                    <CardHeader title="Addon CPU" />
                    <CardContent sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                        <CircularProgress />
                    </CardContent>
                </Card>
            );
        }

        if (error || !health?.addon_stats) {
            return (
                <Card>
                    <CardHeader title="Addon CPU" />
                    <CardContent>
                        <Alert severity="warning">CPU data not available.</Alert>
                    </CardContent>
                </Card>
            );
        }

        return (
            <Card>
                <CardHeader title="Addon CPU" />
                <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Typography variant="h4" component="div">
                            {`${(health.addon_stats.cpu_percent ?? 0).toFixed(1)}%`}
                        </Typography>
                        <Box sx={{ width: '50%', height: 40 }}>
                            {addonCpuHistory.length > 1 && (
                                <Sparklines data={addonCpuHistory} limit={MAX_HISTORY_LENGTH} width={100} height={40}>
                                    <SparklinesLine color={theme.palette.primary.main} />
                                    <SparklinesSpots />
                                </Sparklines>
                            )}
                        </Box>
                    </Box>
                </CardContent>
            </Card>
        );
    };

    const renderAddonMemoryMetric = () => {
        if (isLoading) {
            return (
                <Card>
                    <CardHeader title="Addon Memory" />
                    <CardContent sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                        <CircularProgress />
                    </CardContent>
                </Card>
            );
        }

        if (error || !health?.addon_stats) {
            return (
                <Card>
                    <CardHeader title="Addon Memory" />
                    <CardContent>
                        <Alert severity="warning">Memory data not available.</Alert>
                    </CardContent>
                </Card>
            );
        }

        const { memory_percent, memory_usage, memory_limit } = health.addon_stats;

        return (
            <Card>
                <CardHeader title="Addon Memory" />
                <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                        <Typography variant="h4" component="div">
                            {`${(memory_percent ?? 0).toFixed(1)}%`}
                        </Typography>
                        <Box sx={{ width: '50%', height: 40 }}>
                            {addonMemoryHistory.length > 1 && (
                                <Sparklines data={addonMemoryHistory} limit={MAX_HISTORY_LENGTH} width={100} height={40}>
                                    <SparklinesLine color={theme.palette.success.main} />
                                    <SparklinesSpots />
                                </Sparklines>
                            )}
                        </Box>
                    </Box>
                    <Typography variant="body2" color="text.secondary" align="center">
                        {`${humanizeBytes(memory_usage ?? 0)} / ${humanizeBytes(memory_limit ?? 0)}`}
                    </Typography>
                </CardContent>
            </Card>
        );
    };

    const renderAddonDiskIoMetric = () => {
        if (isLoading) {
            return (
                <Card>
                    <CardHeader title="Disk I/O" />
                    <CardContent sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                        <CircularProgress />
                    </CardContent>
                </Card>
            );
        }

        if (error || !health?.addon_stats) {
            return (
                <Card>
                    <CardHeader title="Disk I/O" />
                    <CardContent>
                        <Alert severity="warning">I/O data not available.</Alert>
                    </CardContent>
                </Card>
            );
        }

        const readRate = addonDiskReadRateHistory.length > 0 ? addonDiskReadRateHistory[addonDiskReadRateHistory.length - 1] : 0;
        const writeRate = addonDiskWriteRateHistory.length > 0 ? addonDiskWriteRateHistory[addonDiskWriteRateHistory.length - 1] : 0;
        const totalHistory = addonDiskReadRateHistory.map((r, i) => r + (addonDiskWriteRateHistory[i] ?? 0));

        return (
            <Card>
                <CardHeader title="Disk I/O" subheader="per second" />
                <CardContent>
                    <Box>
                        <Typography variant="body2">Read: {humanizeBytes(readRate)}/s</Typography>
                        <Typography variant="body2">Write: {humanizeBytes(writeRate)}/s</Typography>
                    </Box>
                    <Box sx={{ height: 40, mt: 1, display: 'flex', justifyContent: 'center' }}>
                        {totalHistory.length > 1 ? (
                            <Sparklines data={totalHistory} limit={MAX_HISTORY_LENGTH} width={100} height={40}>
                                <SparklinesBars style={{ fill: theme.palette.info.main, fillOpacity: ".5" }} />
                            </Sparklines>
                        ) : <Typography variant="caption">gathering data...</Typography>}
                    </Box>
                </CardContent>
            </Card>
        );
    };

    const renderAddonNetworkMetric = () => {
        if (isLoading) {
            return (
                <Card>
                    <CardHeader title="Network I/O" />
                    <CardContent sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                        <CircularProgress />
                    </CardContent>
                </Card>
            );
        }

        if (error || !health?.addon_stats) {
            return (
                <Card>
                    <CardHeader title="Network I/O" />
                    <CardContent>
                        <Alert severity="warning">Network data not available.</Alert>
                    </CardContent>
                </Card>
            );
        }

        const rxRate = addonNetworkRxRateHistory.length > 0 ? addonNetworkRxRateHistory[addonNetworkRxRateHistory.length - 1] : 0;
        const txRate = addonNetworkTxRateHistory.length > 0 ? addonNetworkTxRateHistory[addonNetworkTxRateHistory.length - 1] : 0;
        const totalHistory = addonNetworkRxRateHistory.map((r, i) => r + (addonNetworkTxRateHistory[i] ?? 0));

        return (
            <Card>
                <CardHeader title="Network I/O" subheader="per second" />
                <CardContent>
                    <Box>
                        <Typography variant="body2">Received: {humanizeBytes(rxRate)}/s</Typography>
                        <Typography variant="body2">Sent: {humanizeBytes(txRate)}/s</Typography>
                    </Box>
                    <Box sx={{ height: 40, mt: 1, display: 'flex', justifyContent: 'center' }}>
                        {totalHistory.length > 1 ? (
                            <Sparklines data={totalHistory} limit={MAX_HISTORY_LENGTH} width={100} height={40}>
                                <SparklinesBars style={{ fill: theme.palette.secondary.main, fillOpacity: ".5" }} />
                            </Sparklines>
                        ) : <Typography variant="caption">gathering data...</Typography>}
                    </Box>
                </CardContent>
            </Card>
        );
    };

    const renderVolumeMetrics = () => {
        if (isLoadingVolumes) {
            return (
                <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', mt: 2 }}>
                    <CircularProgress />
                </Box>
            );
        }

        if (errorVolumes) {
            return <Alert severity="error">Could not load disk information.</Alert>;
        }

        const disksWithPartitions = disks?.filter(d => d.partitions && d.partitions.some(p => p.size && p.size > 0)) || [];

        if (disksWithPartitions.length === 0) {
            return <Typography>No partitions with size information found to display.</Typography>;
        }

        return (
            <Grid container spacing={3}>
                {disksWithPartitions.map(disk => {
                    const chartData = (disk.partitions || [])
                        .filter(p => p.size && p.size > 0)
                        .map((p) => ({
                            id: p.id || p.device || 'unknown',
                            value: p.size || 0,
                            label: decodeEscapeSequence(p.name || p.device || 'Unknown'),
                        }));

                    return (
                        <Grid size={{ xs: 12, md: 6, lg: 4 }} key={disk.id}>
                            <Card sx={{ height: '100%' }}>
                                <CardHeader
                                    title={decodeEscapeSequence(disk.id || disk.model || 'Unknown Disk')}
                                    slotProps={
                                        {
                                            title: {
                                                variant: 'subtitle2',
                                                noWrap: true,
                                            },
                                        }
                                    }
                                />
                                <CardContent sx={{ width: '100%', height: 300, display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                                    <PieChart
                                        series={[{
                                            data: chartData,
                                            highlightScope: { fade: 'global', highlight: 'item' },
                                            faded: { innerRadius: 30, additionalRadius: -30, color: 'gray' },
                                            arcLabel: (item) => humanizeBytes(item.value || 0),
                                            arcLabelMinAngle: 25,
                                            valueFormatter: (item) => humanizeBytes(item.value || 0),
                                            innerRadius: 30,
                                            outerRadius: 100,
                                            paddingAngle: 5,
                                            cornerRadius: 15,
                                            cx: 150,
                                            cy: 115, // Adjusted to make space for the legend
                                        }]}
                                        width={300}
                                        height={250} // Set a fixed height for the chart
                                        slotProps={{
                                            legend: {
                                                direction: "horizontal", // Use horizontal legend
                                                position: { vertical: 'bottom', horizontal: 'center' },
                                                sx: {
                                                    fontSize: '0.75rem',
                                                    gap: 1,
                                                },
                                            },
                                        }}
                                    />
                                </CardContent>
                            </Card>
                        </Grid>
                    );
                })}
            </Grid>
        );
    };

    const renderDiskHealthMetrics = () => {
        if (isLoading) {
            return (
                <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                    <CircularProgress />
                </Box>
            );
        }

        if (error || !health?.disk_health) {
            return <Alert severity="error">Could not load disk health metrics.</Alert>;
        }

        return (
            <>
                <Typography variant="h6" sx={{ mt: 4, mb: 2 }}>
                    Disk I/O Health
                </Typography>
                <TableContainer component={Paper}>
                    <Table aria-label="disk health table" size="small">
                        <TableHead>
                            <TableRow>
                                <TableCell>Device</TableCell>
                                <TableCell align="right">Reads IOP/s</TableCell>
                                <TableCell align="right">Writes IOP/s</TableCell>
                                <TableCell align="right">Read Latency (ms)</TableCell>
                                <TableCell align="right">Write Latency (ms)</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {health.disk_health?.per_disk_io?.map((io, i) => {
                                return (
                                    <TableRow key={io.device_name}>
                                        <TableCell component="th" scope="row">
                                            {io.device_name}
                                        </TableCell>
                                        <TableCell align="right">{io.read_iops?.toFixed(2)}</TableCell>
                                        <TableCell align="right">{io.write_iops?.toFixed(2)}</TableCell>
                                        <TableCell align="right">{io.read_latency_ms?.toFixed(2)}</TableCell>
                                        <TableCell align="right">{io.write_latency_ms?.toFixed(2)}</TableCell>
                                    </TableRow>
                                );
                            })}
                        </TableBody>
                    </Table>
                </TableContainer>
            </>
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
                </Grid>
                {renderProcessMetrics()}
                {renderDiskHealthMetrics()}
                <Typography variant="h6" sx={{ mt: 4, mb: 2 }}>
                    Disk Usage
                </Typography>
                {renderVolumeMetrics()}
            </AccordionDetails>
        </Accordion>
    );
}