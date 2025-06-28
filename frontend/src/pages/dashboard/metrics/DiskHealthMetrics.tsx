
import { Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Typography } from "@mui/material";
import type { DiskHealth } from "../../../store/sratApi";

export function DiskHealthMetrics({ diskHealth }: { diskHealth: DiskHealth }) {
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
                        {diskHealth?.per_disk_io?.map((io) => (
                            <TableRow key={io.device_name}>
                                <TableCell component="th" scope="row">
                                    {io.device_name}
                                </TableCell>
                                <TableCell align="right">{io.read_iops?.toFixed(2)}</TableCell>
                                <TableCell align="right">{io.write_iops?.toFixed(2)}</TableCell>
                                <TableCell align="right">{io.read_latency_ms?.toFixed(2)}</TableCell>
                                <TableCell align="right">{io.write_latency_ms?.toFixed(2)}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        </>
    );
}
