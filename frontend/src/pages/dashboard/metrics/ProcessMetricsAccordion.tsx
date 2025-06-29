import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Typography,
} from "@mui/material";
import { ProcessMetrics } from "./ProcessMetrics";
import type { ProcessStatus } from "./types";

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
	return (
		<Accordion
			expanded={expanded}
			onChange={onChange}
			id="process-metrics-details"
		>
			<AccordionSummary
				expandIcon={<ExpandMoreIcon />}
				aria-controls="panel-process-metrics-content"
				id="panel-process-metrics-header"
			>
				<Typography variant="h6">Process Metrics</Typography>
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
