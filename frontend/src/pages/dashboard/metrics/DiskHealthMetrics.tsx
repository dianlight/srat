import {
	Box,
	Card,
	CardContent,
	Grid,
	LinearProgress,
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
import { useEffect, useRef, useState } from "react";
import type { DiskHealth, DiskIoStats } from "../../../store/sratApi";

const MAX_HISTORY_LENGTH = 10;

function humanizeBytes(bytes: number, decimals = 2) {
	if (bytes === 0) return "0 Bytes";

	const k = 1024;
	const dm = decimals < 0 ? 0 : decimals;
	const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

	const i = Math.floor(Math.log(bytes) / Math.log(k));

	return `${parseFloat((bytes / k ** i).toFixed(dm))} ${sizes[i]}`;
}

export function DiskHealthMetrics({
	diskHealth,
}: {
	diskHealth: DiskHealth | undefined;
}) {
	const theme = useTheme();

	const [diskIoHistory, setDiskIoHistory] = useState<Record<string, {
		read_iops: number[];
		write_iops: number[];
		read_latency_ms: number[];
		write_latency_ms: number[];
		temperature: number[];
	}>>({});

	useEffect(() => {
		if (!diskHealth?.per_disk_io) {
			return;
		}

		setDiskIoHistory((prevHistory) => {
			const newHistory = { ...prevHistory };
			diskHealth.per_disk_io?.forEach((io) => {
				const deviceName = io.device_name;
				if (!newHistory[deviceName]) {
					newHistory[deviceName] = {
						read_iops: [],
						write_iops: [],
						read_latency_ms: [],
						write_latency_ms: [],
						temperature: [],
					};
				}

				const updateHistory = (historyArray: number[], newValue: number) => {
					const updated = [...historyArray, newValue];
					if (updated.length > MAX_HISTORY_LENGTH) {
						updated.shift();
					}
					return updated;
				};

				newHistory[deviceName].read_iops = updateHistory(
					newHistory[deviceName].read_iops,
					io.read_iops ?? 0,
				);
				newHistory[deviceName].write_iops = updateHistory(
					newHistory[deviceName].write_iops,
					io.write_iops ?? 0,
				);
				newHistory[deviceName].read_latency_ms = updateHistory(
					newHistory[deviceName].read_latency_ms,
					io.read_latency_ms ?? 0,
				);
				newHistory[deviceName].write_latency_ms = updateHistory(
					newHistory[deviceName].write_latency_ms,
					io.write_latency_ms ?? 0,
				);
				newHistory[deviceName].temperature = updateHistory(
					newHistory[deviceName].temperature,
					io.smart_data?.temperature ?? 0,
				);
			});
			return newHistory;
		});
	}, [diskHealth]);
	return (
		<>
			<TableContainer component={Paper}>
				<Table aria-label="disk health table" size="small">
					<TableHead>
						<TableRow>
							<TableCell>Description</TableCell>
							<TableCell>Device</TableCell>
							<TableCell align="right">Reads IOP/s</TableCell>
							<TableCell align="right">Writes IOP/s</TableCell>
							<TableCell align="right">Read Latency (ms)</TableCell>
							<TableCell align="right">Write Latency (ms)</TableCell>
							<TableCell align="right">Temperature (°C)</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{diskHealth?.per_disk_io?.map((io) => (
							<TableRow key={io.device_name}>
								<TableCell component="th" scope="row">
									{io.device_description}
								</TableCell>
								<TableCell component="th" scope="row">
									{io.device_name}
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
											sx={{ mr: 1, minWidth: "45px", textAlign: "right" }}
										>
											{io.read_iops?.toFixed(2)}
										</Typography>
										<Box sx={{ width: 50, height: 20 }}>
											{(diskIoHistory[io.device_name]?.read_iops?.length || 0) > 1 ? (
												<Sparklines
													data={diskIoHistory[io.device_name].read_iops}
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
											sx={{ mr: 1, minWidth: "45px", textAlign: "right" }}
										>
											{io.write_iops?.toFixed(2)}
										</Typography>
										<Box sx={{ width: 50, height: 20 }}>
											{(diskIoHistory[io.device_name]?.write_iops?.length || 0) > 1 ? (
												<Sparklines
													data={diskIoHistory[io.device_name].write_iops}
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
											sx={{ mr: 1, minWidth: "45px", textAlign: "right" }}
										>
											{io.read_latency_ms?.toFixed(2)}
										</Typography>
										<Box sx={{ width: 50, height: 20 }}>
											{(diskIoHistory[io.device_name]?.read_latency_ms?.length || 0) > 1 ? (
												<Sparklines
													data={diskIoHistory[io.device_name].read_latency_ms}
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
											sx={{ mr: 1, minWidth: "45px", textAlign: "right" }}
										>
											{io.write_latency_ms?.toFixed(2)}
										</Typography>
										<Box sx={{ width: 50, height: 20 }}>
											{(diskIoHistory[io.device_name]?.write_latency_ms?.length || 0) > 1 ? (
												<Sparklines
													data={diskIoHistory[io.device_name].write_latency_ms}
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
											sx={{ mr: 1, minWidth: "45px", textAlign: "right" }}
										>
											{io.smart_data?.temperature ? `${io.smart_data.temperature}°C` : "N/A"}
										</Typography>
										<Box sx={{ width: 50, height: 20 }}>
											{(diskIoHistory[io.device_name]?.temperature?.length || 0) > 1 ? (
												<Sparklines
													data={diskIoHistory[io.device_name].temperature}
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

			<Typography variant="h6" gutterBottom sx={{ mt: 4 }}>
				Disk Partitions
			</Typography>
			<Grid container spacing={2}>
				{Object.entries(diskHealth?.per_partition_info || {}).map(
					([diskName, partitions]) => (
						<Grid size={{ xs: 12, sm: 6, md: 4 }} key={diskName}>
							<Card>
								<CardContent>
									<Typography variant="h6" component="div">
										{
											diskHealth?.per_disk_io?.find(
												(io) => io.device_name === diskName,
											)?.device_description
										}
									</Typography>
									<Typography
										variant="body2"
										color="text.secondary"
										component="div"
									>
										{diskName}
									</Typography>
									{[...(partitions || [])]
										?.sort((a, b) =>
											(a.device || "").localeCompare(b.device || ""),
										)
										?.map((partition) => {
											const totalSpace = partition.total_space_bytes || 0;
											const freeSpace = partition.free_space_bytes || 0;
											const usedSpace = totalSpace - freeSpace;
											const usagePercentage =
												totalSpace > 0 ? (usedSpace / totalSpace) * 100 : 0;

											return (
												<div
													key={partition.device}
													style={{ marginTop: "16px" }}
												>
													<Typography variant="subtitle2">
														{partition.mount_point || partition.device}
													</Typography>
													<LinearProgress
														variant="determinate"
														value={usagePercentage}
														sx={{ height: 10, borderRadius: 5 }}
													/>
													<Typography variant="body2" color="text.secondary">
														{freeSpace > 0 &&
															`${humanizeBytes(freeSpace)} free of `}
														{humanizeBytes(totalSpace)}
													</Typography>
												</div>
											);
										})}
								</CardContent>
							</Card>
						</Grid>
					),
				)}
			</Grid>
		</>
	);
}
