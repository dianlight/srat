import { Grid, Stack } from "@mui/material";
import { DashboardIntro } from "./DashboardIntro";
import { DashboardActions } from "./DashboardActions";
import { DashboardMetrics } from "./DashboardMetrics";
import { useState } from "react";

export function Dashboard() {
    const [isIntroCollapsed, setIsIntroCollapsed] = useState(false);

    const handleToggleIntroCollapse = () => {
        setIsIntroCollapsed(prev => !prev);
    };

    return (
        <Grid container spacing={3} sx={{ p: 2, pt: 3 }}>
            <Grid size={{ xs: 12, md: isIntroCollapsed ? 1 : 4 }}>
                <DashboardIntro isCollapsed={isIntroCollapsed} onToggleCollapse={handleToggleIntroCollapse} />
            </Grid>
            <Grid size={{ xs: 12, md: isIntroCollapsed ? 11 : 8 }}>
                <Stack spacing={3}>
                    <DashboardActions />
                    <DashboardMetrics />
                </Stack>
            </Grid>
        </Grid>
    );
}