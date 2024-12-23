import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import pkg from '../../package.json';
import { getGitCommitHash } from '../macro/getGitCommitHash.ts' with { type: 'macro' };


export function Footer() {
    return (
        <Paper sx={{
            marginTop: 'auto',
            width: '100%',
            //position: 'fixed',
            bottom: 0,
        }} component="footer" square variant="outlined">
            <Container maxWidth="lg">

                <Box
                    sx={{
                        flexGrow: 1,
                        justifyContent: "center",
                        display: "flex",
                        my: 1
                    }}
                >
                    <Typography variant="caption" color="initial">
                        <a className="col s8" href={pkg.repository.url + "/commit/" + getGitCommitHash()}>Version {pkg.version} [Git Hash {getGitCommitHash()}]</a>
                    </Typography>
                </Box>

                <Box
                    sx={{
                        flexGrow: 1,
                        justifyContent: "center",
                        display: "flex",
                        mb: 2,
                    }}
                >
                    <Typography variant="caption" color="initial">
                        © 2024 Copyright {pkg.author.name}
                    </Typography>
                </Box>
            </Container>
        </Paper>
    );
}