import {
	Box,
	Paper,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Typography,
	useTheme,
} from "@mui/material";
import {
	Sparklines,
	SparklinesLine,
	SparklinesSpots,
} from "react-sparklines";
import { useEffect, useState } from "react";
import type { NetworkStats } from "../../../store/sratApi";
import { humanizeBytes } from "./utils";

const MAX_HISTORY_LENGTH = 10;

export function NetworkHealthMetrics({
	networkHealth,
}: {
	networkHealth: NetworkStats | undefined;
}) {
	const theme = useTheme();

	const [networkTrafficHistory, setNetworkTrafficHistory] = useState<
		Record<string, { inbound: number[]; outbound: number[] }>
	>({});

	useEffect(() => {
		if (!networkHealth?.perNicIO) {
			return;
		}

		setNetworkTrafficHistory((prevHistory) => {
			const newHistory = { ...prevHistory };
			networkHealth.perNicIO?.forEach((nic) => {
				const deviceName = nic.deviceName;
				if (!newHistory[deviceName]) {
					newHistory[deviceName] = { inbound: [], outbound: [] };
				}

				const updateHistory = (historyArray: number[], newValue: number) => {
					const updated = [...historyArray, newValue];
					if (updated.length > MAX_HISTORY_LENGTH) {
						updated.shift();
					}
					return updated;
				};

				newHistory[deviceName].inbound = updateHistory(
					newHistory[deviceName].inbound,
					nic.inboundTraffic ?? 0,
				);
				newHistory[deviceName].outbound = updateHistory(
					newHistory[deviceName].outbound,
					nic.outboundTraffic ?? 0,
				);
			});
			return newHistory;
		});
	}, [networkHealth]);
	return (
		<TableContainer component={Paper}>
			<Table aria-label="network health table" size="small">
				<TableHead>
					<TableRow>
						<TableCell>Device</TableCell>
						<TableCell align="right">Inbound Traffic (B/s)</TableCell>
						<TableCell align="right">Outbound Traffic (B/s)</TableCell>
					</TableRow>
				</TableHead>
				<TableBody>
					{networkHealth?.perNicIO?.map((nic) => (
						<TableRow key={nic.deviceName}>
							<TableCell component="th" scope="row">
								{nic.deviceName}
							</TableCell>
							<TableCell align="right" sx={{ minWidth: 150 }}>
								<Box
									sx={{
										display: "flex",
										alignItems: "center",
										justifyContent: "flex-end",
									}}
								>
									<Typography
										variant="body2"
										sx={{ mr: 1, minWidth: "70px", textAlign: "right" }}
									>
										{humanizeBytes(nic.inboundTraffic)}/s
									</Typography>
									<Box sx={{ width: 50, height: 20 }}>
										{(networkTrafficHistory[nic.deviceName]?.inbound?.length || 0) > 1 ? (
											<Sparklines
												data={networkTrafficHistory[nic.deviceName].inbound}
												limit={MAX_HISTORY_LENGTH}
												width={60}
												height={20}
												min={0}
											>
												<SparklinesLine color={theme.palette.primary.main} />
												<SparklinesSpots />
											</Sparklines>
										) : null}
									</Box>
								</Box>
							</TableCell>
							<TableCell align="right" sx={{ minWidth: 150 }}>
								<Box
									sx={{
										display: "flex",
										alignItems: "center",
										justifyContent: "flex-end",
									}}
								>
									<Typography
										variant="body2"
										sx={{ mr: 1, minWidth: "70px", textAlign: "right" }}
									>
										{humanizeBytes(nic.outboundTraffic)}/s
									</Typography>
									<Box sx={{ width: 50, height: 20 }}>
										{(networkTrafficHistory[nic.deviceName]?.outbound?.length || 0) > 1 ? (
											<Sparklines
												data={networkTrafficHistory[nic.deviceName].outbound}
												limit={MAX_HISTORY_LENGTH}
												width={60}
												height={20}
												min={0}
											>
												<SparklinesLine color={theme.palette.primary.main} />
												<SparklinesSpots />
											</Sparklines>
										) : null}
									</Box>
								</Box>
							</TableCell>
						</TableRow>
					))}
				</TableBody>
			</Table>
		</TableContainer>
	);
}
