import { Grid, Stack } from "@mui/material";
import { DashboardIntro } from "./DashboardIntro";
import { DashboardActions } from "./DashboardActions";
import { DashboardMetrics } from "./DashboardMetrics";

export function Dashboard() {
    return (
        <Grid container spacing={3} sx={{ p: 2, pt: 3 }}>
            <Grid size={{ xs: 12, md: 4 }} >
                <DashboardIntro />
            </Grid>
            <Grid size={{ xs: 12, md: 8 }}>
                <Stack spacing={3}>
                    <DashboardActions />
                    <DashboardMetrics />
                </Stack>
            </Grid>
        </Grid>
    );
}