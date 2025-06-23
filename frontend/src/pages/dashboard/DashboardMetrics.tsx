import { Accordion, AccordionDetails, AccordionSummary, Typography, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Box, CircularProgress, Alert } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { useHealth } from "../../hooks/healthHook";
import { useMemo } from "react";

interface ProcessStatus {
    name: string;
    pid: number | null;
    status: 'Running' | 'Stopped';
    cpu: number | null;
    connections: number | null;
}

export function DashboardMetrics() {
    const { health, isLoading, error } = useHealth();

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
                                    <TableCell align="right">{process.cpu !== null ? process.cpu.toFixed(2) : 'N/A'}</TableCell>
                                    <TableCell align="right">{process.connections ?? 'N/A'}</TableCell>
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