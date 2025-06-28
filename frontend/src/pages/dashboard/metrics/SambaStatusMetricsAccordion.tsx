import { Accordion, AccordionDetails, AccordionSummary, Typography } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { SambaStatusMetrics } from "./SambaStatusMetrics";
import type { SambaStatus } from "../../../store/sratApi";

interface SambaStatusMetricsAccordionProps {
    sambaStatus: SambaStatus;
    expanded: boolean;
    onChange: (event: React.SyntheticEvent, isExpanded: boolean) => void;
}

export function SambaStatusMetricsAccordion({ sambaStatus, expanded, onChange }: SambaStatusMetricsAccordionProps) {
    return (
        <Accordion expanded={expanded} onChange={onChange}>
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
