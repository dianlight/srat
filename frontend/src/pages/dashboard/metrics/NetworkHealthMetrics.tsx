
import { Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper, Typography } from "@mui/material";
import { humanizeBytes } from "./utils";
import type { NetworkStats } from "../../../store/sratApi";


export function NetworkHealthMetrics({ networkHealth }: { networkHealth: NetworkStats | undefined }) {
    return (
        <>
            <TableContainer component={Paper}>
                <Table aria-label="network health table" size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>Device</TableCell>
                            <TableCell align="right">Inbound Traffic (B/s)</TableCell>
                            <TableCell align="right">Outbound Traffic (B/s)</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {networkHealth?.perNicIO?.map((nic) => (
                            <TableRow key={nic.deviceName}>
                                <TableCell component="th" scope="row">{nic.deviceName}</TableCell>
                                <TableCell align="right">{humanizeBytes(nic.inboundTraffic)}/s</TableCell>
                                <TableCell align="right">{humanizeBytes(nic.outboundTraffic)}/s</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        </>
    );
}
