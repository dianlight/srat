
import { Box, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography } from "@mui/material";
import { type SambaStatus } from "../../../store/sratApi";

export function SambaStatusMetrics({ sambaStatus }: { sambaStatus: SambaStatus | undefined }) {

    if (!sambaStatus) {
        return <Typography>Samba status data not available.</Typography>;
    }

    const sessions = Object.entries(sambaStatus.sessions || {});
    const tcons = Object.entries(sambaStatus.tcons || {});

    return (
        <>
            <Typography variant="h6" gutterBottom>Samba Sessions</Typography>
            {sessions.length > 0 ? (
                <TableContainer component={Paper}>
                    <Table aria-label="samba sessions table" size="small">
                        <TableHead>
                            <TableRow>
                                <TableCell>Session ID</TableCell>
                                <TableCell align="right">Channels</TableCell>
                                <TableCell>Hostname</TableCell>
                                <TableCell>Remote Machine</TableCell>
                                <TableCell>Username</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {sessions.map(([key, session]) => (
                                <TableRow key={key}>
                                    <TableCell component="th" scope="row">
                                        {session.session_id}
                                    </TableCell>
                                    <TableCell align="right">{Object.keys(session.channels || {}).length}</TableCell>
                                    <TableCell>{session.hostname}</TableCell>
                                    <TableCell>{session.remote_machine}</TableCell>
                                    <TableCell>{session.username}</TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </TableContainer>
            ) : (
                <Typography>No active Samba sessions.</Typography>
            )}

            <Box mt={4}>
                <Typography variant="h6" gutterBottom>Samba Tcons</Typography>
                {tcons.length > 0 ? (
                    <TableContainer component={Paper}>
                        <Table aria-label="samba tcons table" size="small">
                            <TableHead>
                                <TableRow>
                                    <TableCell>Tcon ID</TableCell>
                                    <TableCell>Device</TableCell>
                                    <TableCell>Machine</TableCell>
                                    <TableCell>Service</TableCell>
                                    <TableCell>Share</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {tcons.map(([key, tcon]) => (
                                    <TableRow key={key}>
                                        <TableCell component="th" scope="row">
                                            {tcon.tcon_id}
                                        </TableCell>
                                        <TableCell>{tcon.device}</TableCell>
                                        <TableCell>{tcon.machine}</TableCell>
                                        <TableCell>{tcon.service}</TableCell>
                                        <TableCell>{tcon.share}</TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                ) : (
                    <Typography>No active Samba Tcons.</Typography>
                )}
            </Box>
        </>
    );
}
