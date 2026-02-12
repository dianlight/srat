import { Link, Stack, Tooltip, useMediaQuery, useTheme } from "@mui/material";
import Container from "@mui/material/Container";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import pkg from "../../package.json";
import { getCompileYear } from "../macro/CompileYear.ts" with { type: "macro"
};
import { getCurrentEnv } from "../macro/Environment.ts" with { type: "macro"
};
import { getGitCommitHash } from "../macro/GitCommitHash.ts" with { type: "macro"
};
import { useGetServerEventsQuery } from "../store/sseApi.ts";

export function Footer() {
	const theme = useTheme();
	const isSmallScreen = useMediaQuery(theme.breakpoints.down("sm"));
	const { data: evdata, isLoading } = useGetServerEventsQuery();

	return (
		<Paper
			sx={{
				marginTop: "auto",
				width: "100%",
				//position: 'fixed',
				bottom: 0,
			}}
			component="footer"
			square
			variant="outlined"
		>
			<Container maxWidth="lg">
				<Stack
					direction={isSmallScreen ? "column" : "row"}
					spacing={isSmallScreen ? 0.5 : 2}
					sx={{
						flexGrow: 1,
						justifyContent: "center",
						display: "flex",
						alignItems: "center",
						my: isSmallScreen ? 0.5 : 1,
					}}
				>
					<Tooltip title={isLoading ? "Loading..." : `${evdata?.hello?.build_version} (${getCurrentEnv()})`} arrow placement="top">
						<Typography variant="caption">
							<Link href={`${pkg.repository.url}/commit/${getGitCommitHash()}`}>
								Version {pkg.version}
							</Link>
						</Typography>
					</Tooltip>

					<Typography variant="caption">
						© 2024-{getCompileYear()} Copyright {pkg.author.name}
					</Typography>

					{isSmallScreen || (isLoading && <div>Loading...</div>) ? null : (
						<Tooltip
							title={
								Object.entries(evdata?.heartbeat?.samba_process_status || {}).map(
									([id, status], _index) => (
										<div key={id}>
											<strong>{id}</strong>: PID {status?.pid || "N/A"} - {status?.is_running ? "Running" : "Stopped"}
										</div>
									),
								)
							}
							arrow
						>
							<Typography variant="caption">
								{Object.entries(evdata?.heartbeat?.samba_process_status || {}).map(
									([id, status], index) => (
										<span key={id}>
											{index > 0 && " | "}
											{id} {status?.pid || "off"}
										</span>
									),
								)}
							</Typography>
						</Tooltip>
					)}
				</Stack>
			</Container>
		</Paper>
	);
}
