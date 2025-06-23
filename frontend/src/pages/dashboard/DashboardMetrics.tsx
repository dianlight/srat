import { Accordion, AccordionDetails, AccordionSummary, Typography, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Box, CircularProgress, Alert, useTheme } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { useHealth } from "../../hooks/healthHook";
import { useEffect, useMemo, useState } from "react";
import { Sparklines, SparklinesBars, SparklinesLine, SparklinesSpots } from 'react-sparklines';

interface ProcessStatus {
    name: string;
    pid: number | null;
    status: 'Running' | 'Stopped';
    cpu: number | null;
    connections: number | null;
}

const MAX_HISTORY_LENGTH = 10;

export function DashboardMetrics() {
    const { health, isLoading, error } = useHealth();
    const theme = useTheme();
    const [connectionsHistory, setConnectionsHistory] = useState<Record<string, number[]>>({});
    const [cpuHistory, setCpuHistory] = useState<Record<string, number[]>>({});

    const processData = useMemo((): ProcessStatus[] => {
        if (!health?.samba_process_status) {
            return [];
        }
        // The 'details' object from the health endpoint is expected to have cpu_percent and memory_usage.
        return Object.entries(health.samba_process_status).map(([name, details]) => {
            const typedDetails = details as any; // Cast to access properties not yet in the generated type
            return {
                name,
                pid: typedDetails?.pid || null,
                status: typedDetails?.pid ? 'Running' : 'Stopped',
                cpu: typedDetails?.cpu_percent ?? null,
                connections: typedDetails?.connections ?? null,
            };
        });
    }, [health]);

    useEffect(() => {
        // Don't update history if the initial load is happening or there's an error
        if (isLoading || error || !health?.samba_process_status) {
            return;
        }

        setCpuHistory(prevHistory => {
            const newHistory = { ...prevHistory };
            for (const [name, details] of Object.entries(health.samba_process_status)) {
                const typedDetails = details as any;
                const cpu = typedDetails?.cpu_percent ?? 0; // Default to 0 if null
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
            for (const [name, details] of Object.entries(health.samba_process_status)) {
                const typedDetails = details as any;
                const connections = typedDetails?.connections ?? 0; // Default to 0 if null
                const history = newHistory[name] ? [...newHistory[name]] : [];
                history.push(connections);
                if (history.length > MAX_HISTORY_LENGTH) {
                    history.shift(); // Remove the oldest entry
                }
                newHistory[name] = history;
            }
            return newHistory;
        });
    }, [health, isLoading, error]);

    const renderContent = () => {
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
                                            <Box sx={{ width: 60, height: 20 }}>
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
                                            <Typography variant="body2" sx={{ mr: 1, minWidth: '45px', textAlign: 'right' }}>
                                                {process.connections ?? 'N/A'}
                                            </Typography>
                                            <Box sx={{ width: 60, height: 20 }}>
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
                <Typography variant="body1" sx={{ mt: 3 }}>
                    More graphs and metrics about volumes and partitions will be displayed here.
                </Typography>
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
                {renderContent()}
            </AccordionDetails>
        </Accordion>
    );
}