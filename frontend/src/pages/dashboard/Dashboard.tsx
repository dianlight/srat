import { Grid, Stack } from "@mui/material";
import { DashboardIntro } from "./DashboardIntro";
import { DashboardActions } from "./DashboardActions";
import { DashboardMetrics } from "./DashboardMetrics";
import { useEffect, useRef, useState } from "react";
import { useGithubNews } from "../../hooks/githubNewsHook";

export function Dashboard() {
    const { news, isLoading: isLoadingNews, error: errorNews } = useGithubNews();
    const [isIntroCollapsed, setIsIntroCollapsed] = useState(true);
    const initialCheckDone = useRef(false);

    useEffect(() => {
        // Once news has loaded, if there are news items, expand the intro panel.
        // This should only happen on the initial load.
        if (!isLoadingNews && !initialCheckDone.current) {
            if (news.length > 0) {
                setIsIntroCollapsed(false);
            }
            initialCheckDone.current = true;
        }
    }, [news, isLoadingNews]);

    const handleToggleIntroCollapse = () => {
        setIsIntroCollapsed(prev => !prev);
    };

    return (
        <Grid container spacing={3} sx={{ p: 2, pt: 3 }}>
            <Grid
                size={{ xs: isIntroCollapsed ? 2 : 12, md: isIntroCollapsed ? 1 : 4 }}
                sx={{ display: { xs: 'none', md: 'flex' } }}
            >
                <DashboardIntro
                    isCollapsed={isIntroCollapsed}
                    onToggleCollapse={handleToggleIntroCollapse}
                    news={news}
                    isLoading={isLoadingNews}
                    error={errorNews} />
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