import { Accordion, AccordionDetails, AccordionSummary, Typography, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Box, CircularProgress, Alert } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { useHealth } from "../../hooks/healthHook";
import { useMemo } from "react";

interface ProcessStatus {
    name: string;
    pid: number | null;
    status: 'Running' | 'Stopped';
}

export function DashboardMetrics() {
    const { health, isLoading, error } = useHealth();

    const processData = useMemo((): ProcessStatus[] => {
        if (!health?.samba_process_status) {
            return [];
        }
        return Object.entries(health.samba_process_status).map(([name, details]) => ({
            name,
            pid: details?.pid || null,
            status: details?.pid ? 'Running' : 'Stopped',
        }));
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