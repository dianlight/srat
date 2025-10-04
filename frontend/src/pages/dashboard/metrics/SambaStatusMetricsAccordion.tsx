import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Typography,
	Box,
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
				<Box sx={{ display: "flex", alignItems: "center", width: "100%", justifyContent: "space-between" }}>
					<Typography variant="h6" component="div">
						Samba Status{` ${sambaStatus?.version ? `(v${sambaStatus.version})` : ""}`}
					</Typography>
					{!expanded && sambaStatus && (
						<Box sx={{ display: "flex", gap: 2 }}>
							<Typography variant="body2" color="text.secondary">
								Sessions: {Object.keys(sambaStatus.sessions || {}).length}
							</Typography>
							<Typography variant="body2" color="text.secondary">
								Tcons: {Object.keys(sambaStatus.tcons || {}).length}
							</Typography>
						</Box>
					)}
				</Box>
			</AccordionSummary>
			<AccordionDetails>
				<SambaStatusMetrics sambaStatus={sambaStatus} />
			</AccordionDetails>
		</Accordion>
	);
}
