import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Box,
	Checkbox,
	FormControlLabel,
	Grid,
	IconButton,
	Menu,
	MenuItem,
	Typography,
} from "@mui/material";
import { useEffect, useRef, useState } from "react";
import type { HealthPing } from "../../../store/sratApi";
import { MetricCard } from "./MetricCard";
import type { AddonStatsData } from "./types";
import humanizeDuration from "humanize-duration";
import { TabIDs } from "../../../store/locationState";
import { filesize } from "filesize";

const MAX_HISTORY_LENGTH = 10;

interface SystemMetricsAccordionProps {
	health: HealthPing | null;
	isLoading: boolean;
	error: Error | null | undefined | {};
	expandedAccordion: string | false;
	onAccordionChange: (
		accordionId: string,
	) => (event: React.SyntheticEvent, isExpanded: boolean) => void;
	onDetailClick: (metricId: string) => void;
}

export function SystemMetricsAccordion({
	health,
	isLoading,
	error,
	expandedAccordion,
	onAccordionChange,
	onDetailClick,
}: SystemMetricsAccordionProps) {
	const [addonCpuHistory, setAddonCpuHistory] = useState<number[]>([]);
	const [addonMemoryHistory, setAddonMemoryHistory] = useState<number[]>([]);
	const [addonDiskReadRateHistory, setAddonDiskReadRateHistory] = useState<
		number[]
	>([]);
	const [addonDiskWriteRateHistory, setAddonDiskWriteRateHistory] = useState<
		number[]
	>([]);
	const [addonNetworkRxRateHistory, setAddonNetworkRxRateHistory] = useState<
		number[]
	>([]);
	const [addonNetworkTxRateHistory, setAddonNetworkTxRateHistory] = useState<
		number[]
	>([]);
	const prevAddonStatsRef = useRef<AddonStatsData | null>(null);

	const [diskIopsHistory, setDiskIopsHistory] = useState<number[]>([]);
	const [networkTrafficHistory, setNetworkTrafficHistory] = useState<number[]>(
		[],
	);
	const [sambaSessionsHistory, setSambaSessionsHistory] = useState<number[]>(
		[],
	);

	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
	const [metricVisibility, setMetricVisibility] = useState<
		Record<string, {
			visible: boolean;
			title: string;
			enabled: boolean;
		}>
	>(() => {
		try {
			const storedVisibility = localStorage.getItem("metricVisibility2");
			return storedVisibility
				? JSON.parse(storedVisibility)
				: {
					uptime: {
						visible: true,
						title: "Uptime",
						enabled: true,
					},
					addonCpu: {
						visible: true,
						title: "Addon CPU",
						enabled: true,
					},
					addonMemory: {
						visible: true,
						title: "Addon Memory",
						enabled: true,
					},
					addonDiskIo: {
						visible: false,
						title: "Addon Disk I/O",
						enabled: false,
					},
					addonNetwork: {
						visible: false,
						title: "Addon Network",
						enabled: false,
					},
					globalDiskIo: {
						visible: true,
						title: "Global Disk I/O",
						enabled: true,
					},
					globalNetworkIo: {
						visible: true,
						title: "Global Network I/O",
						enabled: true,
					},
					sambaSessions: {
						visible: true,
						title: "Samba Sessions",
						enabled: true,
					},
				};
		} catch (e) {
			console.error("Failed to parse metric visibility from localStorage", e);
			return {
				uptime: {
					visible: true,
					title: "Uptime",
					enabled: true,
				},
				addonCpu: {
					visible: true,
					title: "Addon CPU",
					enabled: true,
				},
				addonMemory: {
					visible: true,
					title: "Addon Memory",
					enabled: true,
				},
				addonDiskIo: {
					visible: false,
					title: "Addon Disk I/O",
					enabled: false,
				},
				addonNetwork: {
					visible: false,
					title: "Addon Network",
					enabled: false,
				},
				globalDiskIo: {
					visible: true,
					title: "Global Disk I/O",
					enabled: true,
				},
				globalNetworkIo: {
					visible: true,
					title: "Global Network I/O",
					enabled: true,
				},
				sambaSessions: {
					visible: true,
					title: "Samba Sessions",
					enabled: true,
				}
			}
		}
	});

	useEffect(() => {
		try {
			localStorage.setItem(
				"metricVisibility2",
				JSON.stringify(metricVisibility),
			);
		} catch (e) {
			console.error("Failed to save metric visibility to localStorage", e);
		}
	}, [metricVisibility]);

	const handleMenuClick = (event: React.MouseEvent<HTMLElement>) => {
		event.stopPropagation(); // Prevent accordion from toggling
		setAnchorEl(event.currentTarget);
	};

	const handleMenuClose = (event: React.MouseEvent<HTMLElement>) => {
		event.stopPropagation(); // Prevent accordion from toggling
		setAnchorEl(null);
	};

	const handleToggleMetric = (metricName: string) => {
		//console.debug("Toggling metric visibility:", metricName, metricVisibility[metricName]);
		setMetricVisibility((prev) => ({
			...prev,
			[metricName]: { ...prev[metricName], visible: !prev[metricName].visible },
		}));
	};

	useEffect(() => {
		if (isLoading || error || !health) {
			return;
		}

		if (health.addon_stats) {
			const { addon_stats } = health;
			const intervalInSeconds = 5;

			setAddonCpuHistory((prev) => {
				const newHistory = [...prev, addon_stats.cpu_percent ?? 0];
				if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
				return newHistory;
			});
			setAddonMemoryHistory((prev) => {
				const newHistory = [...prev, addon_stats.memory_percent ?? 0];
				if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
				return newHistory;
			});

			if (prevAddonStatsRef.current) {
				const prevStats = prevAddonStatsRef.current;

				const calculateRate = (
					current?: number | null,
					prev?: number | null,
				) => {
					const delta = (current ?? 0) - (prev ?? 0);
					return delta >= 0 ? delta / intervalInSeconds : 0;
				};

				const updateRateHistory = (
					setter: React.Dispatch<React.SetStateAction<number[]>>,
					current?: number | null,
					prev?: number | null,
				) => {
					setter((h) => {
						const rate = calculateRate(current, prev);
						const newHistory = [...h, rate];
						if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
						return newHistory;
					});
				};

				updateRateHistory(
					setAddonDiskReadRateHistory,
					addon_stats.blk_read,
					prevStats.blk_read,
				);
				updateRateHistory(
					setAddonDiskWriteRateHistory,
					addon_stats.blk_write,
					prevStats.blk_write,
				);
				updateRateHistory(
					setAddonNetworkRxRateHistory,
					addon_stats.network_rx,
					prevStats.network_rx,
				);
				updateRateHistory(
					setAddonNetworkTxRateHistory,
					addon_stats.network_tx,
					prevStats.network_tx,
				);
			}
			prevAddonStatsRef.current = addon_stats;
		}

		if (health.disk_health?.global) {
			setDiskIopsHistory((prev) => {
				const newHistory = [
					...prev,
					health.disk_health?.global.total_iops ?? 0,
				];
				if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
				return newHistory;
			});
		}

		if (health.network_health?.global) {
			const totalTraffic =
				(health.network_health.global.totalInboundTraffic ?? 0) +
				(health.network_health.global.totalOutboundTraffic ?? 0);
			setNetworkTrafficHistory((prev) => {
				const newHistory = [...prev, totalTraffic];
				if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
				return newHistory;
			});
		}

		if (health.samba_status?.sessions) {
			const sessionCount = Object.keys(health.samba_status.sessions).length;
			setSambaSessionsHistory((prev) => {
				const newHistory = [...prev, sessionCount];
				if (newHistory.length > MAX_HISTORY_LENGTH) newHistory.shift();
				return newHistory;
			});
		}
	}, [health, isLoading, error]);

	const renderUptimeMetric = () => {
		if (!metricVisibility.uptime.visible) return null;
		const uptimeSeconds = health?.uptime ?? 0;
		const fiveMinutesInSeconds = 5 * 60;
		const uptimeValue = health?.uptime
			? humanizeDuration(health.uptime, {
				round: true,
				largest: uptimeSeconds > fiveMinutesInSeconds ? 2 : undefined,
				units: uptimeSeconds > fiveMinutesInSeconds ? ['y', 'mo', 'd', 'h', 'm'] : undefined
			})
			: "N/A";
		return (
			<MetricCard
				title="Server Uptime"
				value={uptimeValue}
				isLoading={isLoading}
				error={!!error || !health?.uptime}
			/>
		);
	};

	const renderAddonCpuMetric = () => {
		if (!metricVisibility.addonCpu.visible) return null;
		return (
			<MetricCard
				title="Addon CPU"
				value={`${(health?.addon_stats?.cpu_percent ?? 0).toFixed(1)}%`}
				history={addonCpuHistory}
				isLoading={isLoading}
				error={!!error || !health?.addon_stats}
				detailMetricId="processMetrics"
				onDetailClick={onDetailClick}
			/>
		);
	};

	const renderAddonMemoryMetric = () => {
		if (!metricVisibility.addonMemory.visible) return null;
		const { memory_percent, memory_usage, memory_limit } =
			health?.addon_stats || {};
		return (
			<MetricCard
				title="Addon Memory"
				value={`${(memory_percent ?? 0).toFixed(1)}%`}
				history={addonMemoryHistory}
				isLoading={isLoading}
				error={!!error || !health?.addon_stats}
				detailMetricId="processMetrics"
				onDetailClick={onDetailClick}
			>
				<Typography variant="body2" color="text.secondary" align="center">
					{`${filesize(memory_usage ?? 0)} / ${filesize(memory_limit ?? 0)}`}
				</Typography>
			</MetricCard>
		);
	};

	const renderAddonDiskIoMetric = () => {
		if (!metricVisibility.addonDiskIo.visible) return null;
		const readRate =
			addonDiskReadRateHistory.length > 0
				? addonDiskReadRateHistory[addonDiskReadRateHistory.length - 1]
				: 0;
		const writeRate =
			addonDiskWriteRateHistory.length > 0
				? addonDiskWriteRateHistory[addonDiskWriteRateHistory.length - 1]
				: 0;
		const totalHistory = addonDiskReadRateHistory.map(
			(r, i) => r + (addonDiskWriteRateHistory[i] ?? 0),
		);

		return (
			<MetricCard
				title="Addon Disk I/O"
				subheader="per second"
				value={`${filesize(readRate + writeRate)}/s`}
				history={totalHistory}
				isLoading={isLoading}
				error={!!error || !health?.addon_stats}
				historyType="bar"
				detailMetricId="diskHealthMetrics"
				onDetailClick={onDetailClick}
			>
				<Typography variant="body2">
					Read: {filesize(readRate)}/s
				</Typography>
				<Typography variant="body2">
					Write: {filesize(writeRate)}/s
				</Typography>
			</MetricCard>
		);
	};

	const renderAddonNetworkMetric = () => {
		if (!metricVisibility.addonNetwork.visible) return null;
		const rxRate =
			addonNetworkRxRateHistory.length > 0
				? addonNetworkRxRateHistory[addonNetworkRxRateHistory.length - 1]
				: 0;
		const txRate =
			addonNetworkTxRateHistory.length > 0
				? addonNetworkTxRateHistory[addonNetworkTxRateHistory.length - 1]
				: 0;
		const totalHistory = addonNetworkRxRateHistory.map(
			(r, i) => r + (addonNetworkTxRateHistory[i] ?? 0),
		);

		return (
			<MetricCard
				title="Addon Network I/O"
				subheader="per second"
				value={`${filesize(rxRate + txRate)}/s`}
				history={totalHistory}
				isLoading={isLoading}
				error={!!error || !health?.addon_stats}
				historyType="bar"
				detailMetricId="networkHealthMetrics"
				onDetailClick={onDetailClick}
			>
				<Typography variant="body2">
					Received: {filesize(rxRate)}/s
				</Typography>
				<Typography variant="body2">Sent: {filesize(txRate)}/s</Typography>
			</MetricCard>
		);
	};

	const renderGlobalDiskIoMetric = () => {
		if (!metricVisibility.globalDiskIo.visible) return null;
		const { total_read_latency_ms, total_write_latency_ms } =
			health?.disk_health?.global || {};
		return (
			<MetricCard
				title="Global Disk I/O"
				subheader="IOPS"
				value={`${(health?.disk_health?.global?.total_iops ?? 0).toFixed(1)}`}
				history={diskIopsHistory}
				isLoading={isLoading}
				error={!!error || !health?.disk_health?.global}
				detailMetricId="diskHealthMetrics"
				onDetailClick={onDetailClick}
			>
				<Typography variant="body2" color="text.secondary" align="center">
					Latency (r/w): {(total_read_latency_ms ?? 0).toFixed(2)}ms /{" "}
					{(total_write_latency_ms ?? 0).toFixed(2)}ms
				</Typography>
			</MetricCard>
		);
	};

	const renderGlobalNetworkIoMetric = () => {
		if (!metricVisibility.globalNetworkIo.visible) return null;
		const { totalInboundTraffic, totalOutboundTraffic } =
			health?.network_health?.global || {};
		const totalTraffic =
			(totalInboundTraffic ?? 0) + (totalOutboundTraffic ?? 0);

		return (
			<MetricCard
				title="Global Network I/O"
				subheader="per second"
				value={`${filesize(totalTraffic)}/s`}
				history={networkTrafficHistory}
				isLoading={isLoading}
				error={!!error || !health?.network_health?.global}
				detailMetricId="networkHealthMetrics"
				onDetailClick={onDetailClick}
			>
				<Typography variant="body2" color="text.secondary" align="center">
					In: {filesize(totalInboundTraffic ?? 0)}/s | Out:{" "}
					{filesize(totalOutboundTraffic ?? 0)}/s
				</Typography>
			</MetricCard>
		);
	};

	const renderSambaSessionsMetric = () => {
		if (!metricVisibility.sambaSessions.visible) return null;
		const sessionCount = Object.keys(
			health?.samba_status?.sessions || {},
		).length;
		return (
			<MetricCard
				title="Samba Sessions"
				value={sessionCount.toString()}
				history={sambaSessionsHistory}
				isLoading={isLoading}
				error={!!error || !health?.samba_status}
				detailMetricId="sambaStatusMetrics"
				onDetailClick={onDetailClick}
			/>
		);
	};

	return (
		<Accordion
			data-tutor={`reactour__tab${TabIDs.DASHBOARD}__step4`}
			expanded={expandedAccordion === "system-metrics-details"}
			onChange={onAccordionChange("system-metrics-details")}
		>
			<AccordionSummary
				expandIcon={<ExpandMoreIcon />}
				aria-controls="panel-system-metrics-content"
				id="panel-system-metrics-header"
			>
				<Box
					sx={{
						display: "flex",
						alignItems: "center",
						justifyContent: "space-between",
						width: "100%",
					}}
				>
					<Typography variant="h6">System Metrics</Typography>
					<IconButton
						component="div"
						role="button"
						aria-label="show metrics menu"
						aria-controls="metrics-menu"
						aria-haspopup="true"
						onClick={handleMenuClick}
						color="inherit"
					>
						<MoreVertIcon />
					</IconButton>
					<Menu
						id="metrics-menu"
						anchorEl={anchorEl}
						keepMounted
						open={Boolean(anchorEl)}
						onClose={handleMenuClose}
					>
						{Object.entries(metricVisibility).map(([key, data]) => (
							<MenuItem
								key={key}
								onClick={(e) => {
									e.stopPropagation();
									handleToggleMetric(key);
								}}
							>
								<FormControlLabel
									disabled={!data.enabled}
									control={
										<Checkbox
											checked={data.visible}
											name={key} color="primary"
										/>
									}
									label={data.title || key.replace(/([A-Z])/g, " $1").trim()}
								/>
							</MenuItem>
						))}
					</Menu>
				</Box>
			</AccordionSummary>
			<AccordionDetails>
				<Grid container spacing={3} sx={{ mb: 4 }}>
					{metricVisibility.uptime.visible && (
						<Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
							{renderUptimeMetric()}
						</Grid>
					)}
					{metricVisibility.addonCpu.visible && (
						<Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
							{renderAddonCpuMetric()}
						</Grid>
					)}
					{metricVisibility.addonMemory.visible && (
						<Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
							{renderAddonMemoryMetric()}
						</Grid>
					)}
					{metricVisibility.addonDiskIo.visible && (
						<Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
							{renderAddonDiskIoMetric()}
						</Grid>
					)}
					{metricVisibility.addonNetwork.visible && (
						<Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
							{renderAddonNetworkMetric()}
						</Grid>
					)}
					{metricVisibility.globalDiskIo.visible && (
						<Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
							{renderGlobalDiskIoMetric()}
						</Grid>
					)}
					{metricVisibility.globalNetworkIo.visible && (
						<Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
							{renderGlobalNetworkIoMetric()}
						</Grid>
					)}
					{metricVisibility.sambaSessions.visible && (
						<Grid size={{ xs: 12, sm: 6, md: 4, lg: 4 }}>
							{renderSambaSessionsMetric()}
						</Grid>
					)}
				</Grid>
			</AccordionDetails>
		</Accordion>
	);
}
