import { Accordion, AccordionDetails, AccordionSummary, Typography } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { VolumeMetrics } from "./VolumeMetrics";
import { useGetHealthQuery } from "../../../store/sratApi";
import type { Disk, DiskHealth, HealthPing } from "../../../store/sratApi";

interface DiskHealthMetricsAccordionProps {
    diskHealth: DiskHealth;
}

export function VolumeMetricsAccordion({ diskHealth }: DiskHealthMetricsAccordionProps) {
    return (
        <Accordion defaultExpanded>
            <AccordionSummary
                expandIcon={<ExpandMoreIcon />}
                aria-controls="panel-volume-metrics-content"
                id="panel-volume-metrics-header"
            >
                <Typography variant="h6">Disk Usage</Typography>
            </AccordionSummary>
            <AccordionDetails>
                <VolumeMetrics diskHealth={diskHealth} />
            </AccordionDetails>
        </Accordion>
    );
}
