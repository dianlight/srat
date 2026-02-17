import { Memory as SmartIcon, Warning as WarningIcon } from "@mui/icons-material";
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
    Tooltip,
    Typography,
    useTheme,
} from "@mui/material";
import { filesize } from "filesize";
import { useEffect, useState } from "react";
import { SafeSparkLineChart as SparkLineChart } from "../../../components/charts/SafeSparkLineChart";
import { PreviewDialog } from "../../../components/PreviewDialog";
import type { DiskHealth, DiskIoStats, PerPartitionInfo } from "../../../store/sratApi";

const MAX_HISTORY_LENGTH = 10;
type DiskIoHistoryKey =
	| "read_iops"
	| "write_iops"
	| "read_latency_ms"
	| "write_latency_ms"
	| "temperature";

export function DiskHealthMetrics({
	diskHealth,
}: {
	diskHealth: DiskHealth | undefined;
}) {
	const theme = useTheme();
	const [selectedIoStats, setSelectedIoStats] = useState<DiskIoStats | PerPartitionInfo | null>(null);

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
					io.smart_data?.temperature?.value ?? 0,
				);
			});
			return newHistory;
		});
	}, [diskHealth]);
	const isDiskIoStats = (obj: any): obj is DiskIoStats =>
		!!obj && typeof obj === "object" && "device_name" in obj && "device_description" in obj;

	const sortedDiskIo = [...(diskHealth?.per_disk_io ?? [])].sort((a, b) =>
		(a.device_name || "").localeCompare(b.device_name || ""),
	);

	const sortedPartitionEntries = Object.entries(diskHealth?.per_partition_info || {}).sort(
		([a], [b]) => (a || "").localeCompare(b || ""),
	);

	const renderMetricCell = ({
		deviceName,
		valueLabel,
		historyKey,
	}: {
		deviceName: string;
		valueLabel?: string;
		historyKey: DiskIoHistoryKey;
	}) => {
		const historyData = diskIoHistory[deviceName]?.[historyKey] ?? [];

		return (
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
						{valueLabel}
					</Typography>
					<Box sx={{ width: 50, height: 20 }}>
						{historyData.length > 1 ? (
							<SparkLineChart
								data={historyData}
								width={60}
								height={20}
								color={theme.palette.primary.main}
								showTooltip
							/>
						) : null}
					</Box>
				</Box>
			</TableCell>
		);
	};

	return (
		<>
			<PreviewDialog
				objectToDisplay={selectedIoStats}
				onClose={() => setSelectedIoStats(null)}
				open={!!selectedIoStats}
				title={
					isDiskIoStats(selectedIoStats)
						? `Detailed I/O Stats - ${selectedIoStats.device_description} (${selectedIoStats.device_name})`
						: `Detailed Partition Stats - ${selectedIoStats?.name ?? selectedIoStats?.device ?? ""}`
				}
			/>
			<TableContainer component={Paper}>
				<Table aria-label="disk health table" size="small">
					<TableHead>
						<TableRow>
							<TableCell>Description</TableCell>
							<TableCell>Device</TableCell>
							{(diskHealth as any)?.hdidle_running && (
								<TableCell align="center">Spin Status</TableCell>
							)}
							<TableCell align="right">Reads IOP/s</TableCell>
							<TableCell align="right">Writes IOP/s</TableCell>
							<TableCell align="right">Read Latency (ms)</TableCell>
							<TableCell align="right">Write Latency (ms)</TableCell>
							<TableCell align="right">Temperature (°C / Max °C)</TableCell>
							<TableCell align="right">Power On Hours</TableCell>
							<TableCell align="right">Power Cycles</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{sortedDiskIo.map((io) => {
							// Look up SMART health from per_disk_info using device_description as key
							const diskInfo = (diskHealth as any)?.per_disk_info?.[io.device_description];
							const smartHealth = diskInfo?.smart_health;
							const isSmartHealthOk = !smartHealth || smartHealth.passed;
							const smartHealthTooltip = smartHealth && !smartHealth.passed
								? `SMART Health: ${smartHealth.overall_status}${smartHealth.failing_attributes?.length ? `\nFailing attributes: ${smartHealth.failing_attributes.join(", ")}` : ""}`
								: "";

							return (
								<TableRow key={io.device_name}>
									<TableCell component="th" scope="row" sx={{ cursor: "pointer" }} onClick={() => setSelectedIoStats(io)}>
										<Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
											{!isSmartHealthOk && diskInfo?.smart_info?.supported && (
												<Tooltip title={smartHealthTooltip} arrow>
													<WarningIcon
														color="warning"
														fontSize="small"
														sx={{ verticalAlign: "middle" }}
													/>
												</Tooltip>
											)}
											{io.device_description}
										</Box>
									</TableCell>
									<TableCell component="th" scope="row">
										<Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
											{diskInfo?.smart_info?.supported && (
												<Tooltip 
													title={
														<Box>
															<Typography variant="body2" sx={{ fontWeight: 'bold', mb: 0.5 }}>
																SMART Enabled
															</Typography>
															{diskInfo.smart_info.disk_type && (
																<Typography variant="caption" display="block">
																	Type: {diskInfo.smart_info.disk_type}
																</Typography>
															)}
															{diskInfo.smart_info.rotation_rate !== undefined && diskInfo.smart_info.rotation_rate > 0 && (
																<Typography variant="caption" display="block">
																	RPM: {diskInfo.smart_info.rotation_rate}
																</Typography>
															)}
															{diskInfo.smart_info.rotation_rate === 0 && (
																<Typography variant="caption" display="block">
																	Type: SSD
																</Typography>
															)}
														</Box>
													}
													arrow
												>
													<SmartIcon
														color="info"
														fontSize="small"
														sx={{ verticalAlign: "middle" }}
													/>
												</Tooltip>
											)}
											{io.device_name}
										</Box>
									</TableCell>
									{(diskHealth as any)?.hdidle_running && (
										<TableCell align="center">
											{diskInfo?.hdidle_status ? (
												<Tooltip 
													title={diskInfo.hdidle_status.spun_down 
														? `Spun down${diskInfo.hdidle_status.spin_down_at ? ` at ${new Date(diskInfo.hdidle_status.spin_down_at).toLocaleTimeString()}` : ''}`
														: `Active${diskInfo.hdidle_status.spin_up_at ? ` since ${new Date(diskInfo.hdidle_status.spin_up_at).toLocaleTimeString()}` : ''}`
													}
													arrow
												>
													<Typography 
														variant="body2" 
														sx={{ 
															color: diskInfo.hdidle_status.spun_down 
																? theme.palette.info.main 
																: theme.palette.success.main,
															fontWeight: 'medium'
														}}
													>
														{diskInfo.hdidle_status.spun_down ? "⏸" : "▶"}
													</Typography>
												</Tooltip>
											) : (
												<Typography variant="body2" color="text.secondary">
													N/A
												</Typography>
											)}
										</TableCell>
									)}
									{renderMetricCell({
										deviceName: io.device_name,
										valueLabel: io.read_iops?.toFixed(2),
										historyKey: "read_iops",
									})}
									{renderMetricCell({
										deviceName: io.device_name,
										valueLabel: io.write_iops?.toFixed(2),
										historyKey: "write_iops",
									})}
									{renderMetricCell({
										deviceName: io.device_name,
										valueLabel: io.read_latency_ms?.toFixed(2),
										historyKey: "read_latency_ms",
									})}
									{renderMetricCell({
										deviceName: io.device_name,
										valueLabel: io.write_latency_ms?.toFixed(2),
										historyKey: "write_latency_ms",
									})}
									{renderMetricCell({
										deviceName: io.device_name,
										valueLabel: `${io.smart_data?.temperature?.value ? `${io.smart_data.temperature.value}°C` : "N/A"} / ${io.smart_data?.temperature?.max ? `${io.smart_data.temperature.max}°C` : "N/A"}`,
										historyKey: "temperature",
									})}
									<TableCell align="right">
										<Typography variant="body2">
											{io.smart_data?.power_on_hours?.value?.toLocaleString() ?? "N/A"}
										</Typography>
									</TableCell>
									<TableCell align="right">
										<Typography variant="body2">
											{io.smart_data?.power_cycle_count?.value?.toLocaleString() ?? "N/A"}
										</Typography>
									</TableCell>
								</TableRow>
							);
						})}
					</TableBody>
				</Table>
			</TableContainer>

			<Typography variant="h6" gutterBottom sx={{ mt: 4 }}>
				Disk Partitions
			</Typography>
			<Grid container spacing={2}>
				{sortedPartitionEntries.map(
					([diskName, partitions]) => (
						<Grid size={{ xs: 12, sm: 6, md: 4 }} key={diskName}>
							<Card>
								<CardContent>
									<Typography variant="h6" component="div" >
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
													<Typography variant="subtitle2" sx={{ cursor: "pointer" }} onClick={() => setSelectedIoStats(partition)}>
														{partition.name || partition.device}
													</Typography>
													<LinearProgress
														variant="determinate"
														value={usagePercentage}
														sx={{ height: 10, borderRadius: 5 }}
														color={freeSpace > 0 ? (usagePercentage > 90 ? "error" : "primary") : "inherit"}
													/>
													<Typography variant="body2" color="text.secondary">
														{freeSpace > 0 &&
															`${filesize(freeSpace)} free of `}
														{filesize(totalSpace)}
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

// Force a full reload on HMR updates to avoid @mui/x-charts internal hook mismatch during hot swapping
if (import.meta && (import.meta as any).hot) {
	(import.meta as any).hot.accept(() => {
		window.location.reload();
	});
}
