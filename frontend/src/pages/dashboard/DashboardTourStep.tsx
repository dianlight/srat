import type { StepType } from '@reactour/tour'
import { TabIDs } from '../../store/locationState';
import { Box, Divider, Typography } from '@mui/material';
export const DashboardSteps: StepType[] = [
    {
        selector: '[data-tutor="reactour__step1"]',
        content: <Box sx={{ p: 3, maxWidth: '600px', mx: 'auto' }}>
            <Typography variant="h4" component="h1" gutterBottom>
                Welcome to Your New Dashboard!
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Typography variant="body1" paragraph>
                Welcome to this tutorial! We're excited to give you a tour of your new dashboard and show you how to make the most of its features.
            </Typography>
            <Typography variant="body1" paragraph>
                In the following steps, we will walk you through the specific components and functionalities you'll find. We'll cover everything you need to know to get started and use the dashboard effectively.
            </Typography>
            <Typography variant="body1" sx={{ fontWeight: 'bold' }}>
                Let's begin!
            </Typography>
        </Box>
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step1"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Navigate with the Tab Menu
            </Typography>

            <Typography variant="body1" color="text.secondary">
                This is the tab menu, your primary navigation tool. Use it to get to the dashboard where you can monitor your most important metrics at a glance.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step2"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Welcome and News Section
            </Typography>

            <Typography variant="body1" color="text.secondary">
                This is the welcome and news section. Here you can find the latest updates and announcements.
            </Typography>
        </>,
        position: "center",
        action: (elem) => {
            console.log("Step 2 action triggered", elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step3"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Actionable Items
            </Typography>

            <Typography variant="body1" color="text.secondary">
                This is the actionable items section. Here you can find tasks and suggestions based on your hardware and software configurations.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step4"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Monitor Your Metrics
            </Typography>

            <Typography variant="body1" color="text.secondary">
                This is the metrics overview section. Here you can see key performance indicators and other important data at a glance.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step5"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Step 5
            </Typography>

            <Typography variant="body1" color="text.secondary">
                This is the fifth step.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step6"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Step 6
            </Typography>

            <Typography variant="body1" color="text.secondary">
                This is the sixth step.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step7"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Step 7
            </Typography>

            <Typography variant="body1" color="text.secondary">
                This is the seventh step.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step8"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Step 8
            </Typography>

            <Typography variant="body1" color="text.secondary">
                This is the eighth step.
            </Typography>
        </>,
    },
]