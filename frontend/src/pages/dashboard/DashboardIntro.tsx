import { Accordion, AccordionDetails, AccordionSummary, Typography } from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';

export function DashboardIntro() {
    return (
        <Accordion defaultExpanded>
            <AccordionSummary
                expandIcon={<ExpandMoreIcon />}
                aria-controls="panel-intro-content"
                id="panel-intro-header"
            >
                <Typography variant="h6">Welcome to SRAT</Typography>
            </AccordionSummary>
            <AccordionDetails>
                <Typography variant="body1">
                    This is your storage management dashboard. Here you can get a quick overview of your system's storage health and perform common actions.
                </Typography>
                <Typography variant="body2" sx={{ mt: 2 }}>
                    <strong>Latest News:</strong> Version X.Y.Z has been released with new performance improvements!
                </Typography>
            </AccordionDetails>
        </Accordion>
    );
}