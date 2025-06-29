import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Typography,
} from "@mui/material";
import type { DiskHealth } from "../../../store/sratApi";
import { DiskHealthMetrics } from "./DiskHealthMetrics";

interface DiskHealthMetricsAccordionProps {
	diskHealth: DiskHealth | undefined;
	expanded: boolean;
	onChange: (event: React.SyntheticEvent, isExpanded: boolean) => void;
}

export function DiskHealthMetricsAccordion({
	diskHealth,
	expanded,
	onChange,
}: DiskHealthMetricsAccordionProps) {
	return (
		<Accordion expanded={expanded} onChange={onChange} id="disk-health-details">
			<AccordionSummary
				expandIcon={<ExpandMoreIcon />}
				aria-controls="panel-disk-health-content"
				id="panel-disk-health-header"
			>
				<Typography variant="h6">Disk Health</Typography>
			</AccordionSummary>
			<AccordionDetails>
				<DiskHealthMetrics diskHealth={diskHealth} />
			</AccordionDetails>
		</Accordion>
	);
}
