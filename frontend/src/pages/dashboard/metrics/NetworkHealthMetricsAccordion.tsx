import { Accordion, AccordionDetails, AccordionSummary, Typography } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { NetworkHealthMetrics } from "./NetworkHealthMetrics";
import type { NetworkStats } from "../../../store/sratApi";


interface NetworkHealthMetricsAccordionProps {
    networkHealth: NetworkStats;
}

export function NetworkHealthMetricsAccordion({ networkHealth }: NetworkHealthMetricsAccordionProps) {
    return (
        <Accordion defaultExpanded>
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
