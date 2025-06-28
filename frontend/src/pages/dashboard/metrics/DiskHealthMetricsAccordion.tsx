import { Accordion, AccordionDetails, AccordionSummary, Typography } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { DiskHealthMetrics } from "./DiskHealthMetrics";
import type { DiskHealth } from "../../../store/sratApi";

interface DiskHealthMetricsAccordionProps {
    diskHealth: DiskHealth;
}

export function DiskHealthMetricsAccordion({ diskHealth }: DiskHealthMetricsAccordionProps) {
    return (
        <Accordion defaultExpanded>
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
