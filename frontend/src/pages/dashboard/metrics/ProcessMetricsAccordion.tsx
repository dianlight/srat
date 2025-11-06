import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Typography,
	Box,
} from "@mui/material";
import { ProcessMetrics } from "./ProcessMetrics";
import type { ProcessStatus } from "./types";
import { TabIDs } from "../../../store/locationState";

interface ProcessMetricsAccordionProps {
	processData: ProcessStatus[];
	cpuHistory: Record<string, number[]>;
	memoryHistory: Record<string, number[]>;
	connectionsHistory: Record<string, number[]>;
	expanded: boolean;
	onChange: (event: React.SyntheticEvent, isExpanded: boolean) => void;
}

export function ProcessMetricsAccordion({
	processData,
	cpuHistory,
	memoryHistory,
	connectionsHistory,
	expanded,
	onChange,
}: ProcessMetricsAccordionProps) {
	// Aggregate metrics for collapsed view
	// Exclude subprocesses (negative PIDs) from aggregates
	const aggregate = (() => {
		if (!processData || processData.length === 0) {
			return { cpu: 0, memory: 0, connections: 0 };
		}
		let cpu = 0;
		let memory = 0;
		let connections = 0;
		for (const p of processData) {
			// Skip subprocesses (negative PIDs)
			if (p.pid !== null && p.pid < 0) {
				continue;
			}
			if (typeof p.cpu === "number" && !isNaN(p.cpu)) cpu += p.cpu;
			if (typeof p.memory === "number" && !isNaN(p.memory)) memory += p.memory;
			if (typeof p.connections === "number" && !isNaN(p.connections))
				connections += p.connections;
		}
		return { cpu, memory, connections };
	})();

	return (
		<Accordion
			expanded={expanded}
			onChange={onChange}
			id="process-metrics-details"
			data-tutor={`reactour__tab${TabIDs.DASHBOARD}__step5`}
		>
			<AccordionSummary
				expandIcon={<ExpandMoreIcon />}
				aria-controls="panel-process-metrics-content"
				id="panel-process-metrics-header"
			>
				<Box sx={{ display: "flex", alignItems: "center", width: "100%", justifyContent: "space-between" }}>
					<Typography variant="h6">Process Metrics</Typography>
					{!expanded && (
						<Box sx={{ display: "flex", gap: 2 }}>
							<Typography variant="body2" color="text.secondary">
								CPU: {aggregate.cpu.toFixed(1)}%
							</Typography>
							<Typography variant="body2" color="text.secondary">
								Mem: {aggregate.memory.toFixed(1)}%
							</Typography>
							<Typography variant="body2" color="text.secondary">
								Conns: {aggregate.connections}
							</Typography>
						</Box>
					)}
				</Box>
			</AccordionSummary>
			<AccordionDetails>
				<ProcessMetrics
					processData={processData}
					cpuHistory={cpuHistory}
					memoryHistory={memoryHistory}
					connectionsHistory={connectionsHistory}
				/>
			</AccordionDetails>
		</Accordion>
	);
}
