import { Link, Stack, Tooltip, useMediaQuery, useTheme } from "@mui/material";
import Container from "@mui/material/Container";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import JsonTable from "ts-react-json-table";
import pkg from "../../package.json";
import { getGitCommitHash } from "../macro/getGitCommitHash.ts" with {
	type: "macro",
};
import { type HealthPing, usePutRestartMutation } from "../store/sratApi.ts";
//import { apiContext } from "../Contexts.ts";

export function Footer(props: { healthData: HealthPing }) {
	const theme = useTheme();
	const isSmallScreen = useMediaQuery(theme.breakpoints.down("sm"));
	const [restart, { isLoading }] = usePutRestartMutation();

	//const samba = useSWR<DtoSambaProcessStatus>('/samba/status', () => apiContext.samba.statusList().then(res => res.data));

	const handleRestart = () => {
		if (!isLoading) {
			restart()
				.unwrap()
				.then(() => {
					console.log("Server restarted successfully");
				})
				.catch((error) => {
					console.error("Failed to restart the server:", error);
				});
		}
	};

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
					<Tooltip title={props.healthData.build_version} arrow placement="top">
						<Typography variant="caption">
							<Link href={`${pkg.repository.url}/commit/${getGitCommitHash()}`}>
								Version {pkg.version}
							</Link>
						</Typography>
					</Tooltip>

					<Typography variant="caption">
						© 2024-2025 Copyright {pkg.author.name}
					</Typography>

					{isSmallScreen || (
						<Tooltip
							title={
								<JsonTable
									rows={Object.values(props.healthData.samba_process_status)}
								/>
							}
							arrow
						>
							<Typography variant="caption">
								smbd {props.healthData.samba_process_status?.smbd?.pid || "off"}{" "}
								| nmbd{" "}
								{props.healthData.samba_process_status?.nmbd?.pid || "off"} |
								wsdd2{" "}
								{props.healthData.samba_process_status?.wsdd2?.pid || "off"} |
								avahi{" "}
								{props.healthData.samba_process_status?.avahi?.pid || "off"}
							</Typography>
						</Tooltip>
					)}
					{/*
                    <Tooltip title="Restart the server" arrow>
                        <Typography onClick={() => handleRestart()} variant="caption">[R]</Typography>
                    </Tooltip>
                    */}
				</Stack>
			</Container>
		</Paper>
	);
}
