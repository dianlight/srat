import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import pkg from '../../package.json';
import { getGitCommitHash } from '../macro/getGitCommitHash.ts' with { type: 'macro' };
import Stack from "@mui/material/Stack";
import { AppBar, Link, Toolbar, Tooltip } from "@mui/material";
import { useContext } from "react";
import { usePutRestartMutation, type HealthPing } from "../store/sratApi.ts";
//import { apiContext } from "../Contexts.ts";


export function Footer(props: { healthData: HealthPing }) {

    const [restart, { isLoading }] = usePutRestartMutation();

    //const samba = useSWR<DtoSambaProcessStatus>('/samba/status', () => apiContext.samba.statusList().then(res => res.data));

    const handleRestart = () => {
        if (!isLoading) {
            restart().unwrap().then(() => {
                console.log("Server restarted successfully");
            }).catch((error) => {
                console.error("Failed to restart the server:", error);
            });
        }
    };

    return (
        <Paper sx={{
            marginTop: 'auto',
            width: '100%',
            //position: 'fixed',
            bottom: 0,
        }} component="footer" square variant="outlined">
            <Container maxWidth="lg">

                <Stack
                    direction="row"
                    spacing={2}
                    sx={{
                        flexGrow: 1,
                        justifyContent: "center",
                        display: "flex",
                        my: 1
                    }}
                >
                    <Typography variant="caption">
                        <Link href={pkg.repository.url + "/commit/" + getGitCommitHash()}>Version {pkg.version}</Link>
                    </Typography>

                    <Typography variant="caption">
                        © 2024-2025 Copyright {pkg.author.name}
                    </Typography>

                    <Tooltip title={JSON.stringify(props.healthData.samba_process_status, null, 2)} arrow>
                        <Typography variant="caption">
                            smbd pid {props.healthData.samba_process_status?.pid || "unknown"}
                        </Typography>
                    </Tooltip>

                    <Tooltip title="Restart the server" arrow>
                        <Typography onClick={() => handleRestart()} variant="caption">[R]</Typography>
                    </Tooltip>

                </Stack>
            </Container>
        </Paper>
    );
}