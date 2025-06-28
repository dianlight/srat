
import { Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Box, Typography, useTheme } from "@mui/material";
import { Sparklines, SparklinesBars, SparklinesLine, SparklinesSpots } from 'react-sparklines';
import { type ProcessStatus } from "./types";

const MAX_HISTORY_LENGTH = 10;

interface ProcessMetricsProps {
    processData: ProcessStatus[];
    cpuHistory: Record<string, number[]>;
    memoryHistory: Record<string, number[]>;
    connectionsHistory: Record<string, number[]>;
}

export function ProcessMetrics({ processData, cpuHistory, memoryHistory, connectionsHistory }: ProcessMetricsProps) {
    const theme = useTheme();

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
                                                <Sparklines data={cpuHistory[process.name]} limit={MAX_HISTORY_LENGTH} width={60} height={20} min={0} max={100}>
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
                                                <Sparklines data={memoryHistory[process.name]} limit={MAX_HISTORY_LENGTH} width={60} height={20} min={0} max={100}>
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
                                                <Sparklines data={connectionsHistory[process.name]} limit={MAX_HISTORY_LENGTH} width={60} height={20} min={0}>
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
}
