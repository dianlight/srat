import type { StepType } from '@reactour/tour'
import { TabIDs } from '../../store/locationState';
import { Box, Divider, Typography } from '@mui/material';
import { TourEvents, TourEventTypes } from '../../utils/TourEvents';
export const DashboardSteps: StepType[] = [
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step0"]`,
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

            <Typography variant="body1" >
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

            <Typography variant="body1" >
                This is the welcome and news section. Here you can find the latest updates and announcements.
            </Typography>
        </>,
        position: "center",
        action: (elem) => {
            TourEvents.emit(TourEventTypes.DASHBOARD_STEP_2, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step3"]`,
        mutationObservables: [`[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step3"]`],
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Actionable Items
            </Typography>

            <Typography variant="body1" >
                This is the actionable items section. Here you can find tasks and suggestions based on your hardware and software configurations.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.DASHBOARD_STEP_3, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step4"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Monitor Your Metrics
            </Typography>

            <Typography variant="body1" >
                This is the metrics overview section. Here you can see key performance indicators and other important data at a glance.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.DASHBOARD_STEP_4, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step5"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Process Metrics
            </Typography>

            <Typography variant="body1" paragraph>
                Here you can view detailed metrics for important system processes:
            </Typography>
            <Typography variant="body2" paragraph>
                <strong>smbd</strong>: Handles file sharing and network access for SMB/CIFS clients. It enables Windows and other devices to access shared folders and files on your system.
            </Typography>
            <Typography variant="body2" paragraph>
                <strong>nmbd</strong>: Manages NetBIOS name resolution and browsing. It helps devices discover your server on the local network and supports legacy Windows networking.
            </Typography>
            <Typography variant="body2" paragraph>
                <strong>wsdd2</strong>: Provides Web Service Discovery, allowing Windows 10+ clients to find your shares automatically using modern protocols.
            </Typography>
            <Typography variant="body2" paragraph>
                <strong>avahi</strong>: Implements mDNS/DNS-SD (Bonjour/ZeroConf), making your server discoverable by macOS, Linux, and other devices supporting network service discovery.
            </Typography>
            <Typography variant="body1" >
                Monitoring these processes helps ensure your network shares and discovery features are working correctly.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.DASHBOARD_STEP_5, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step6"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Disk Health
            </Typography>

            <Typography variant="body1" paragraph>
                Here you can monitor your disk health, including key metrics such as IOPS (input/output operations per second), latency, and temperature (if available). These indicators help you assess disk performance and detect potential issues early.
            </Typography>
            <Typography variant="body1" paragraph>
                You can also view partition occupation, which shows how much space is used and available on each disk partition. Keeping an eye on these metrics helps ensure your system runs smoothly and prevents unexpected storage problems.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.DASHBOARD_STEP_6, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step7"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Network Health
            </Typography>

            <Typography variant="body1" paragraph>
                Here you can monitor key network health metrics such as IOPS (input/output operations per second) and latency. IOPS indicates how many network operations are being processed, helping you understand the throughput and responsiveness of your network connections.
            </Typography>
            <Typography variant="body1" paragraph>
                Latency measures the time it takes for data to travel across the network. Lower latency means faster communication between devices, which is important for smooth file sharing and remote access.
            </Typography>
            <Typography variant="body1" >
                Keeping an eye on these metrics helps you detect network bottlenecks and maintain reliable connectivity for all your devices.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.DASHBOARD_STEP_7, elem);
        }
    },
    {
        selector: `[data-tutor="reactour__tab${TabIDs.DASHBOARD}__step8"]`,
        content: <>
            <Typography variant="h6" component="h2" gutterBottom>
                Samba Status
            </Typography>

            <Typography variant="body1" >
                Here you can view the status of the Samba services running on your system. Monitoring these services helps ensure that file sharing and network access are functioning correctly.
            </Typography>
        </>,
        action: (elem) => {
            TourEvents.emit(TourEventTypes.DASHBOARD_STEP_8, elem);
        }
    },
]