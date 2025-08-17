import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Typography,
} from "@mui/material";
import type { NetworkStats } from "../../../store/sratApi";
import { NetworkHealthMetrics } from "./NetworkHealthMetrics";
import { TabIDs } from "../../../store/locationState";

interface NetworkHealthMetricsAccordionProps {
	networkHealth: NetworkStats | undefined;
	expanded: boolean;
	onChange: (event: React.SyntheticEvent, isExpanded: boolean) => void;
}

export function NetworkHealthMetricsAccordion({
	networkHealth,
	expanded,
	onChange,
}: NetworkHealthMetricsAccordionProps) {
	return (
		<Accordion
			expanded={expanded}
			onChange={onChange}
			id="network-health-details"
			data-tutor={`reactour__tab${TabIDs.DASHBOARD}__step7`}
		>
			<AccordionSummary
				expandIcon={<ExpandMoreIcon />}
				aria-controls="panel-network-health-content"
				id="panel-network-health-header"
			>
				<Typography variant="h6">Network Health</Typography>
			</AccordionSummary>
			<AccordionDetails>
				<NetworkHealthMetrics networkHealth={networkHealth} />
			</AccordionDetails>
		</Accordion>
	);
}
