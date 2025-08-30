import { Link, Stack, Tooltip, useMediaQuery, useTheme } from "@mui/material";
import Container from "@mui/material/Container";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import JsonTable from "ts-react-json-table";
import pkg from "../../package.json";
import { getGitCommitHash } from "../macro/getGitCommitHash.ts" with {
	type: "macro",
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
					<Tooltip title={isLoading ? "Loading..." : evdata?.hello.build_version} arrow placement="top">
						<Typography variant="caption">
							<Link href={`${pkg.repository.url}/commit/${getGitCommitHash()}`}>
								Version {pkg.version}
							</Link>
						</Typography>
					</Tooltip>

					<Typography variant="caption">
						© 2024-2025 Copyright {pkg.author.name}
					</Typography>

					{isSmallScreen || (isLoading && <div>Loading...</div>) ? null : (
						<Tooltip
							title={
								<JsonTable
									rows={Object.values(evdata?.heartbeat?.samba_process_status || {})}
								/>
							}
							arrow
						>
							<Typography variant="caption">
								smbd {evdata?.heartbeat?.samba_process_status?.smbd?.pid || "off"}{" "}
								| nmbd{" "}
								{evdata?.heartbeat?.samba_process_status?.nmbd?.pid || "off"} |
								wsdd2{" "}
								{evdata?.heartbeat?.samba_process_status?.wsdd2?.pid || "off"}
							</Typography>
						</Tooltip>
					)}
				</Stack>
			</Container>
		</Paper>
	);
}
