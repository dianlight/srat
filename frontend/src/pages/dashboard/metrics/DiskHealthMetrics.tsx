
import { Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Typography, Card, CardContent, Grid, LinearProgress } from "@mui/material";
import type { DiskHealth } from "../../../store/sratApi";

function humanizeBytes(bytes: number, decimals = 2) {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];

    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

export function DiskHealthMetrics({ diskHealth }: { diskHealth: DiskHealth }) {
    return (
        <>
            <TableContainer component={Paper}>
                <Table aria-label="disk health table" size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>Description</TableCell>
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
                                    {io.device_description}
                                </TableCell>
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

            <Typography variant="h6" gutterBottom sx={{ mt: 4 }}>
                Disk Partitions
            </Typography>
            <Grid container spacing={2}>
                {Object.entries(diskHealth?.per_partition_info || {}).map(([diskName, partitions]) => (
                    <Grid size={{ xs: 12, sm: 6, md: 4 }} key={diskName}>
                        <Card>
                            <CardContent>
                                <Typography variant="h6" component="div">
                                    {diskHealth?.per_disk_io?.find(io => io.device_name === diskName)?.device_description}
                                </Typography>
                                <Typography variant="body2" color="text.secondary" component="div">
                                    {diskName}
                                </Typography>
                                {[...(partitions || [])]?.sort((a, b) => a.device?.localeCompare(b.device))?.map((partition) => {
                                    const totalSpace = partition.total_space_bytes || 0;
                                    const freeSpace = partition.free_space_bytes || 0;
                                    const usedSpace = totalSpace - freeSpace;
                                    const usagePercentage = totalSpace > 0 ? (usedSpace / totalSpace) * 100 : 0;

                                    return (
                                        <div key={partition.device} style={{ marginTop: '16px' }}>
                                            <Typography variant="subtitle2">
                                                {partition.mount_point || partition.device}
                                            </Typography>
                                            <LinearProgress
                                                variant="determinate"
                                                value={usagePercentage}
                                                sx={{ height: 10, borderRadius: 5 }}
                                            />
                                            <Typography variant="body2" color="text.secondary">
                                                {freeSpace > 0 && `${humanizeBytes(freeSpace)} free of `}{humanizeBytes(totalSpace)}
                                            </Typography>
                                        </div>
                                    );
                                })}
                            </CardContent>
                        </Card>
                    </Grid>
                ))}
            </Grid>
        </>
    );
}
