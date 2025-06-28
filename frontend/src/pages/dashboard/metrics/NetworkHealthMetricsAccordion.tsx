import { Accordion, AccordionDetails, AccordionSummary, Typography } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { NetworkHealthMetrics } from "./NetworkHealthMetrics";
import type { NetworkStats } from "../../../store/sratApi";


interface NetworkHealthMetricsAccordionProps {
    networkHealth: NetworkStats;
    expanded: boolean;
    onChange: (event: React.SyntheticEvent, isExpanded: boolean) => void;
}

export function NetworkHealthMetricsAccordion({ networkHealth, expanded, onChange }: NetworkHealthMetricsAccordionProps) {
    return (
        <Accordion expanded={expanded} onChange={onChange}>
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
