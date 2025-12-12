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
import { SafeSparkLineChart as SparkLineChart } from "../../../components/charts/SafeSparkLineChart";
import type { ProcessStatus } from "./types";

const MAX_HISTORY_LENGTH = 10;

interface ProcessMetricsProps {
	processData: ProcessStatus[];
	cpuHistory: Record<string, number[]>;
	memoryHistory: Record<string, number[]>;
	connectionsHistory: Record<string, number[]>;
}

export function ProcessMetrics({
	processData,
	cpuHistory,
	memoryHistory,
	connectionsHistory,
}: ProcessMetricsProps) {
	const theme = useTheme();

	// Separate main processes and subprocesses
	// Subprocesses have negative PIDs where the absolute value is the parent PID
	const mainProcesses = processData.filter((p) => p.pid === null || p.pid >= 0);
	const subprocesses = processData.filter((p) => p.pid !== null && p.pid < 0);

	// Create a map of parent PID to subprocesses
	const subprocessMap = new Map<number, ProcessStatus[]>();
	for (const subprocess of subprocesses) {
		const parentPid = Math.abs(subprocess.pid!);
		if (!subprocessMap.has(parentPid)) {
			subprocessMap.set(parentPid, []);
		}
		subprocessMap.get(parentPid)!.push(subprocess);
	}

	const renderProcess = (process: ProcessStatus, isSubprocess = false, uniqueId?: string) => {
		const pidDisplay = (isSubprocess && process.pid !== null && process.pid <= 0)
			? "sub"
			: process.pid !== null && process.pid >= 0
				? process.pid
				: "N/A";

		const cpuDisplay =
			process.cpu !== null && process.cpu > 0
				? `${process.cpu.toFixed(1)}%`
				: isSubprocess
					? "N/A"
					: process.cpu !== null
						? `${process.cpu.toFixed(1)}%`
						: "N/A";

		const memoryDisplay =
			process.memory !== null && process.memory > 0
				? `${process.memory.toFixed(1)}%`
				: isSubprocess
					? "N/A"
					: process.memory !== null
						? `${process.memory.toFixed(1)}%`
						: "N/A";

		const showCpuChart =
			!isSubprocess && (cpuHistory[process.name]?.length || 0) > 1;
		const showMemoryChart =
			!isSubprocess && (memoryHistory[process.name]?.length || 0) > 1;

		const tableRowKey = uniqueId || process.name;
		return (
			<TableRow key={tableRowKey}>
				<TableCell component="th" scope="row">
					<Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
						{isSubprocess && (
							<Box
								sx={{
									width: 20,
									height: 20,
									display: "flex",
									alignItems: "center",
									justifyContent: "center",
								}}
							>
								<Box
									sx={{
										width: 2,
										height: 12,
										bgcolor: "text.disabled",
										mr: 0.5,
									}}
								/>
								<Box
									sx={{
										width: 8,
										height: 2,
										bgcolor: "text.disabled",
									}}
								/>
							</Box>
						)}
						<Typography
							variant="body2"
							sx={{
								fontStyle: isSubprocess ? "italic" : "normal",
								color: isSubprocess ? "text.secondary" : "text.primary",
							}}
						>
							{process.name}
						</Typography>
					</Box>
				</TableCell>
				<TableCell
					align="right"
					sx={{
						color:
							process.status === "Running" ? "success.main" : "error.main",
					}}
				>
					{process.status}
				</TableCell>
				<TableCell align="right">{pidDisplay}</TableCell>
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
							{cpuDisplay}
						</Typography>
						<Box sx={{ width: 50, height: 20 }}>
							{showCpuChart ? (
								<SparkLineChart
									data={cpuHistory[process.name] ?? []}
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
							{memoryDisplay}
						</Typography>
						<Box sx={{ width: 50, height: 20 }}>
							{showMemoryChart ? (
								<SparkLineChart
									data={memoryHistory[process.name] ?? []}
									width={60}
									height={20}
									color={theme.palette.success.main}
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
							sx={{ mr: 1, minWidth: "45px", textAlign: "right" }}
						>
							{process.connections ?? "N/A"}
						</Typography>
						<Box sx={{ width: 50, height: 20 }}>
							{(connectionsHistory[process.name]?.length || 0) > 1 ? (
								<SparkLineChart
									data={connectionsHistory[process.name] ?? []}
									width={60}
									height={20}
									plotType="bar"
									color={theme.palette.secondary.main}
									showTooltip
								/>
							) : null}
						</Box>
					</Box>
				</TableCell>
			</TableRow>
		);
	};

	return (
		<>
			<Typography variant="body1" sx={{ mb: 2 }}>
				Overview of Samba-related processes.
			</Typography>
			<TableContainer component={Paper}>
				<Table aria-label="samba processes table" size="small">
					<TableHead>
						<TableRow>
							<TableCell>Process</TableCell>
							<TableCell align="right">Status</TableCell>
							<TableCell align="right">PID</TableCell>
							<TableCell align="right">CPU (%)</TableCell>
							<TableCell align="right">Memory (%)</TableCell>
							<TableCell align="right">Connections</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{mainProcesses.flatMap((process, processIndex) => {
							const rows = [
								renderProcess(process, false, `process-${processIndex}`),
							];
							
							if (
								process.pid !== null &&
								process.pid > 0 &&
								subprocessMap.has(process.pid)
							) {
								subprocessMap.get(process.pid)!.forEach((subprocess, subIndex) => {
									rows.push(renderProcess(subprocess, true, `process-${processIndex}-sub-${subIndex}`));
								});
							}
							
							if (process.child_processes) {
								process.child_processes.forEach((child, childIndex) => {
									rows.push(renderProcess(child, true, `process-${processIndex}-child-${childIndex}`));
								});
							}
							
							return rows;
						})}
					</TableBody>
				</Table>
			</TableContainer>
		</>
	);
}

// Force a full reload on HMR updates to avoid @mui/x-charts internal hook mismatch during hot swapping
if (import.meta && (import.meta as any).hot) {
	(import.meta as any).hot.accept(() => {
		window.location.reload();
	});
}
