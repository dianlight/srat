import type { StepType } from '@reactour/tour'
import { Box, Divider, Typography } from '@mui/material';
import { TabIDs } from '../../store/locationState';
import { TourEvents, TourEventTypes } from '../../utils/TourEvents';

export const SettingsSteps: StepType[] = [
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SETTINGS}__step0"]`,
        content: <Box sx={{ p: 3, maxWidth: '600px', mx: 'auto' }}>
            <Typography variant="h4" component="h1" gutterBottom>
                Configure System Settings
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Typography variant="body1" component="p">
                This section allows you to configure essential system settings for your Samba server, including network configuration, security settings, and system preferences.
            </Typography>
            <Typography variant="body1" sx={{ fontWeight: 'bold' }}>
                Let's explore the key configuration options.
            </Typography>
        </Box>
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SETTINGS}__step1"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Navigate with the Tab Menu
            </Typography>

            <Typography variant="body1" >
                This is the tab menu, your primary navigation tool. Use it to access the settings section where you can configure your system preferences.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SETTINGS}__step2"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Update Channel & Telemetry
            </Typography>
            <Typography variant="body1" >
                Configure your update channel preferences and telemetry settings. The update channel determines which version updates you receive, while telemetry helps improve the software.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SETTINGS}__step3"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                System Identity
            </Typography>
            <Typography variant="body1" >
                Set your system's hostname and workgroup. The hostname identifies your server on the network, while the workgroup determines which Windows workgroup it belongs to.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.SETTINGS_STEP_3, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SETTINGS}__step4"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Fetch System Hostname
            </Typography>
            <Typography variant="body1" >
                Use this button to automatically fetch the current system hostname. This is helpful when you want to use the actual system hostname instead of manually typing it.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SETTINGS}__step5"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Network Access Control
            </Typography>
            <Typography variant="body1" >
                Configure which IP addresses or networks can access your Samba shares. You can add individual IP addresses or use CIDR notation for entire networks (e.g., 192.168.1.0/24).
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.SETTINGS_STEP_5, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SETTINGS}__step6"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Add Default Allow Hosts
            </Typography>
            <Typography variant="body1" >
                Click this button to quickly add commonly used IP address ranges to your allow hosts list. This saves time when setting up typical network configurations.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SETTINGS}__step7"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Compatibility & Performance
            </Typography>
            <Typography variant="body1" >
                Enable compatibility mode for older clients or multi-channel mode for improved performance with modern clients that support multiple network connections.
            </Typography>
        </>,
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SETTINGS}__step8"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Network Interface Configuration
            </Typography>
            <Typography variant="body1" >
                Choose specific network interfaces for Samba to use, or enable "Bind All Interfaces" to automatically use all available network interfaces.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.SETTINGS_STEP_8, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.SETTINGS}__step9"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Save Your Changes
            </Typography>
            <Typography variant="body1" >
                Use these buttons to reset your changes or apply them to the system. The buttons are only enabled when you have unsaved changes.
            </Typography>
        </>,
    },
];