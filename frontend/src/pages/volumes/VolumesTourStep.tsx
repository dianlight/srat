
import type { StepType } from '@reactour/tour'
import { Box, Divider, Typography } from '@mui/material';
import { TabIDs } from '../../store/locationState';
import { TourEvents, TourEventTypes } from '../../utils/TourEvents';

export const VolumesSteps: StepType[] = [
    {
        selector: `[data-tutor="reactour__tab${TabIDs.VOLUMES}__step0"]`,
        content: <Box sx={{ p: 3, maxWidth: '600px', mx: 'auto' }}>
            <Typography variant="h4" component="h1" gutterBottom>
                Manage Your Volumes
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Typography variant="body1" component="p">
                This section lets you view, mount, unmount, and manage your storage volumes.
            </Typography>
            <Typography variant="body1" sx={{ fontWeight: 'bold' }}>
                Let's explore the key actions.
            </Typography>
        </Box>
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.VOLUMES}__step1"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Navigate with the Tab Menu
            </Typography>

            <Typography variant="body1">
                This is the tab menu, your primary navigation tool. Use it to get to the volumes section where you can manage your storage.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.VOLUMES}__step2"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Hide System Partitions
            </Typography>
            <Typography variant="body1">
                Use this switch to hide system partitions and focus on the volumes that are relevant to you.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.VOLUMES}__step3"]`,
        mutationObservables: [`[data-tutor="reactour__tab${TabIDs.VOLUMES}__step3"]`],
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Disk Information
            </Typography>
            <Typography variant="body1">
                Disks are presented in this accordion view. Click on a disk to expand it and see its partitions.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.VOLUMES_STEP_3, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.VOLUMES}__step4"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Partition Information
            </Typography>
            <Typography variant="body1">
                Here you can see the partitions for the selected disk.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.VOLUMES_STEP_4, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.VOLUMES}__step5"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Partition Actions
            </Typography>
            <Typography variant="body1">
                Each partition provides quick actions like mount, unmount, view settings, and create a share.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.VOLUMES_STEP_5, elem);
        }
    },
];
