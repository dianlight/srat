import type { StepType } from '@reactour/tour'
import { Box, Divider, Typography } from '@mui/material';
import { TabIDs } from '../../store/locationState';
import { TourEvents, TourEventTypes } from '../../utils/TourEvents';

export const UsersSteps: StepType[] = [
    {
        selector: `[data-tutor="reactour__tab${TabIDs.USERS}__step0"]`,
        content: <Box sx={{ p: 3, maxWidth: '600px', mx: 'auto' }}>
            <Typography variant="h4" component="h1" gutterBottom>
                Manage Samba Users
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Typography variant="body1" component="p">
                This section lets you manage Samba users, including creating, editing, and deleting accounts for file sharing.
            </Typography>
            <Typography variant="body1" sx={{ fontWeight: 'bold' }}>
                Let's walk through the main user management features.
            </Typography>
        </Box>
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.USERS}__step1"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                User List
            </Typography>
            <Typography variant="body1">
                Here you see all configured Samba users. You can search, filter, and sort users for easier management.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.USERS}__step2"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Add New User
            </Typography>
            <Typography variant="body1">
                Click this button to add a new Samba user. You’ll be prompted to enter a username, password, and optional details.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.USERS}__step3"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Edit User
            </Typography>
            <Typography variant="body1">
                Select a user and click the edit icon to update their information or change their password.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.USERS_STEP_3, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.USERS}__step4"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Delete User
            </Typography>
            <Typography variant="body1">
                Remove a user by clicking the delete icon. You’ll be asked to confirm before the user is permanently deleted.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.USERS}__step5"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                User Status & Permissions
            </Typography>
            <Typography variant="body1">
                View each user’s status (enabled/disabled) and their assigned permissions. You can toggle user access as needed.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.USERS}__step6"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Save Changes
            </Typography>
            <Typography variant="body1">
                Don’t forget to save your changes after adding, editing, or deleting users. The save button is enabled when there are unsaved changes.
            </Typography>
        </>,
    },
];