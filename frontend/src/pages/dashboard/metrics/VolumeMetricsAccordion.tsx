import { Accordion, AccordionDetails, AccordionSummary, Typography } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { VolumeMetrics } from "./VolumeMetrics";
import type { Disk } from "../../../store/sratApi";

interface VolumeMetricsAccordionProps {
    disks: Disk[];
    isLoadingVolumes: boolean;
    errorVolumes: Error | null | undefined | {};
}

export function VolumeMetricsAccordion({
    disks,
    isLoadingVolumes,
    errorVolumes,
}: VolumeMetricsAccordionProps) {
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
                <VolumeMetrics disks={disks} isLoadingVolumes={isLoadingVolumes} errorVolumes={errorVolumes} />
            </AccordionDetails>
        </Accordion>
    );
}
