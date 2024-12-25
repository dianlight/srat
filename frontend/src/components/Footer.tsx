import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import pkg from '../../package.json';
import { getGitCommitHash } from '../macro/getGitCommitHash.ts' with { type: 'macro' };
import Stack from "@mui/material/Stack";
import { AppBar, Link, Toolbar, Tooltip } from "@mui/material";
import type { MainHealth, MainSambaProcessStatus } from "../srat.ts";
import useSWR from "swr";
import { useContext } from "react";
import { apiContext } from "../Contexts.ts";


export function Footer(props: { healthData: MainHealth }) {
    const api = useContext(apiContext);

    const samba = useSWR<MainSambaProcessStatus>('/samba/status', () => api.samba.statusList().then(res => res.data));


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
                        © 2024 Copyright {pkg.author.name}
                    </Typography>

                    <Tooltip title={JSON.stringify(samba, null, 2)} onOpen={() => samba.mutate()} arrow>
                        <Typography variant="caption">
                            smbd pid {props.healthData.samba_pid || "unknown"}
                        </Typography>
                    </Tooltip>

                </Stack>
            </Container>
        </Paper>
    );
}