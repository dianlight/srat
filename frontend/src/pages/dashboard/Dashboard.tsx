import { Grid, Stack } from "@mui/material";
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
		<Grid container spacing={3} sx={{ p: 2, pt: 3 }}>
			<Grid
				size={{ xs: isIntroCollapsed ? 2 : 12, md: isIntroCollapsed ? 1 : 4 }}
				sx={{ display: { xs: "none", md: "flex" } }}
			>
				<DashboardIntro
					isCollapsed={isIntroCollapsed}
					onToggleCollapse={handleToggleIntroCollapse}
					news={news}
					isLoading={isLoadingNews}
					error={errorNews}
				/>
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
