import { Accordion, AccordionDetails, AccordionSummary, Typography } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { SambaStatusMetrics } from "./SambaStatusMetrics";
import type { SambaStatus } from "../../../store/sratApi";

interface SambaStatusMetricsAccordionProps {
    sambaStatus: SambaStatus;
}

export function SambaStatusMetricsAccordion({ sambaStatus }: SambaStatusMetricsAccordionProps) {
    return (
        <Accordion defaultExpanded>
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
