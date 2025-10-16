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
import { SparkLineChart } from "@mui/x-charts/SparkLineChart";
import { useEffect, useMemo, useState } from "react";
import type { NetworkStats } from "../../../store/sratApi";
import { filesize } from "filesize";

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
	const sortedPerNicIO = useMemo(
		() =>
			networkHealth?.perNicIO
				? [...networkHealth.perNicIO].sort((a, b) =>
					(a.deviceName || "").localeCompare(b.deviceName || ""),
				)
				: [],
		[networkHealth?.perNicIO],
	);

	return (
		<TableContainer component={Paper}>
			<Table aria-label="network health table" size="small">
				<TableHead>
					<TableRow>
						<TableCell>Device</TableCell>
						<TableCell align="right">IP</TableCell>
						<TableCell align="right">Netmask</TableCell>
						<TableCell align="right">Inbound Traffic (B/s)</TableCell>
						<TableCell align="right">Outbound Traffic (B/s)</TableCell>
					</TableRow>
				</TableHead>
				<TableBody>
					{sortedPerNicIO.map((nic) => (
						<TableRow key={nic.deviceName}>
							<TableCell component="th" scope="row">
								{nic.deviceName}
							</TableCell>
							<TableCell align="right">
								{nic.ip || "-"}
							</TableCell>
							<TableCell align="right">
								{nic.netmask || "-"}
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
										{filesize(nic.inboundTraffic)}/s
									</Typography>
									<Box sx={{ width: 50, height: 20 }}>
										{(networkTrafficHistory[nic.deviceName]?.inbound?.length || 0) > 1 ? (
											<SparkLineChart
												data={networkTrafficHistory[nic.deviceName]?.inbound ?? []}
												width={60}
												height={20}
												color={theme.palette.primary.main}
												showTooltip
											/>
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
										{filesize(nic.outboundTraffic)}/s
									</Typography>
									<Box sx={{ width: 50, height: 20 }}>
										{(networkTrafficHistory[nic.deviceName]?.outbound?.length || 0) > 1 ? (
											<SparkLineChart
												data={networkTrafficHistory[nic.deviceName]?.outbound ?? []}
												width={60}
												height={20}
												color={theme.palette.primary.main}
												showTooltip
											/>
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
