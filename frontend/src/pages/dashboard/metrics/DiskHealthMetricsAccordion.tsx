import { Accordion, AccordionDetails, AccordionSummary, Typography } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { DiskHealthMetrics } from "./DiskHealthMetrics";
import type { DiskHealth } from "../../../store/sratApi";

interface DiskHealthMetricsAccordionProps {
    diskHealth: DiskHealth;
    expanded: boolean;
    onChange: (event: React.SyntheticEvent, isExpanded: boolean) => void;
}

export function DiskHealthMetricsAccordion({ diskHealth, expanded, onChange }: DiskHealthMetricsAccordionProps) {
    return (
        <Accordion expanded={expanded} onChange={onChange}>
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
