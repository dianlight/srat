import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Typography,
} from "@mui/material";
import type { SambaStatus } from "../../../store/sratApi";
import { SambaStatusMetrics } from "./SambaStatusMetrics";
import { TabIDs } from "../../../store/locationState";

interface SambaStatusMetricsAccordionProps {
	sambaStatus: SambaStatus | undefined;
	expanded: boolean;
	onChange: (event: React.SyntheticEvent, isExpanded: boolean) => void;
}

export function SambaStatusMetricsAccordion({
	sambaStatus,
	expanded,
	onChange,
}: SambaStatusMetricsAccordionProps) {
	return (
		<Accordion
			data-tutor={`reactour__tab${TabIDs.DASHBOARD}__step8`}

			expanded={expanded}
			onChange={onChange}
			id="samba-status-details"
		>
			<AccordionSummary
				expandIcon={<ExpandMoreIcon />}
				aria-controls="panel-samba-metrics-content"
				id="panel-samba-metrics-header"
			>
				<Typography variant="h6">Samba Status</Typography>
			</AccordionSummary>
			<AccordionDetails>
				<SambaStatusMetrics sambaStatus={sambaStatus} />
			</AccordionDetails>
		</Accordion>
	);
}
