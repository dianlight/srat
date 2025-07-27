import { Box, Grid, Stack } from "@mui/material";
import { useState } from "react";
import { useGithubNews } from "../../hooks/githubNewsHook";
import { DashboardActions } from "./DashboardActions";
import { DashboardIntro } from "./DashboardIntro";
import { DashboardMetrics } from "./DashboardMetrics";

export function Dashboard() {
	const { news, isLoading: isLoadingNews, error: errorNews } = useGithubNews();
	const [isIntroCollapsed, setIsIntroCollapsed] = useState(true);

	const handleToggleIntroCollapse = () => {
		setIsIntroCollapsed((prev) => !prev);
	};

	return (
		<Grid container spacing={{ xs: 2, sm: 3 }} sx={{ p: { xs: 1, sm: 2 }, pt: { xs: 2, sm: 3 } }}>
			<Box
				sx={{
					display: { xs: "none", md: "flex" },
					width: { md: isIntroCollapsed ? '8.33%' : '33.33%' }
				}}
			>
				<DashboardIntro
					isCollapsed={isIntroCollapsed}
					onToggleCollapse={handleToggleIntroCollapse}
					news={news}
					isLoading={isLoadingNews}
					error={errorNews}
				/>
			</Box>
			<Box
				sx={{
					width: { xs: '100%', md: isIntroCollapsed ? '91.67%' : '66.67%' }
				}}
			>
				<Stack spacing={{ xs: 2, sm: 3 }}>
					<DashboardActions />
					<DashboardMetrics />
				</Stack>
			</Box>
		</Grid>
	);
}
