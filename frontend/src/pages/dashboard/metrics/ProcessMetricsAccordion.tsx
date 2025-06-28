import { Accordion, AccordionDetails, AccordionSummary, Typography } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { ProcessMetrics } from "./ProcessMetrics";
import type { ProcessStatus } from "./types";

interface ProcessMetricsAccordionProps {
    processData: ProcessStatus[];
    cpuHistory: Record<string, number[]>;
    memoryHistory: Record<string, number[]>;
    connectionsHistory: Record<string, number[]>;
}

export function ProcessMetricsAccordion({
    processData,
    cpuHistory,
    memoryHistory,
    connectionsHistory,
}: ProcessMetricsAccordionProps) {
    return (
        <Accordion defaultExpanded>
            <AccordionSummary
                expandIcon={<ExpandMoreIcon />}
                aria-controls="panel-process-metrics-content"
                id="panel-process-metrics-header"
            >
                <Typography variant="h6">Process Metrics</Typography>
            </AccordionSummary>
            <AccordionDetails>
                <ProcessMetrics
                    processData={processData}
                    cpuHistory={cpuHistory}
                    memoryHistory={memoryHistory}
                    connectionsHistory={connectionsHistory}
                />
            </AccordionDetails>
        </Accordion>
    );
}
