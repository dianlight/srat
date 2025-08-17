import type { StepType } from '@reactour/tour'
import { Box, Divider, Typography } from '@mui/material';
import { TabIDs } from '../../store/locationState';
import { TourEvents, TourEventTypes } from '../../utils/TourEvents';

export const SharesSteps: StepType[] = [
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SHARES}__step0"]`,
        content: <Box sx={{ p: 3, maxWidth: '600px', mx: 'auto' }}>
            <Typography variant="h4" component="h1" gutterBottom>
                Manage Your Shares
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Typography variant="body1" component="p">
                This section lets you view, create, edit, enable/disable and delete Samba shares.
            </Typography>
            <Typography variant="body1" sx={{ fontWeight: 'bold' }}>
                Let's explore the key actions.
            </Typography>
        </Box>
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SHARES}__step1"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Navigate with the Tab Menu
            </Typography>

            <Typography variant="body1" >
                This is the tab menu, your primary navigation tool. Use it to get to the shares section where you can manage your Samba shares.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SHARES}__step2"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Create New Share
            </Typography>
            <Typography variant="body1" >
                Use this button to create a new share. It's enabled when an unused mounted path is available.
            </Typography>
        </>,
        //        action: (elem) => {
        //            TourEvents.emit(TourEventTypes.SHARES_STEP_2, elem);
        //        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SHARES}__step3"]`,
        mutationObservables: [`[data-tutor="reactour__tab${TabIDs.SHARES}__step3"]`],
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Share Groups
            </Typography>
            <Typography variant="body1" >
                Shares are grouped by usage. Expand a group to see its shares.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.SHARES_STEP_3, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SHARES}__step4"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Share Actions
            </Typography>
            <Typography variant="body1" >
                Each share provides quick actions like settings, view mount settings, delete, enable/disable.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.SHARES_STEP_4, elem);
        }
    },
];
