import React, { useState, useEffect } from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Typography,
    Button,
    RadioGroup,
    FormControlLabel,
    Radio,
    Box,
    Alert,
    Link,
    CircularProgress,
} from '@mui/material';
import {
    useGetTelemetryInternetConnectionQuery,
    usePutSettingsMutation,
    useGetSettingsQuery,
    Telemetry_mode,
} from '../store/sratApi';
import type { TelemetryMode } from '../services/telemetryService';
import telemetryService from '../services/telemetryService';

interface TelemetryModalProps {
    open: boolean;
    onClose: () => void;
}

const TelemetryModal: React.FC<TelemetryModalProps> = ({ open, onClose }) => {
    const [selectedMode, setSelectedMode] = useState<Telemetry_mode>(Telemetry_mode.All);
    const [isSubmitting, setIsSubmitting] = useState(false);

    const { data: internetConnection, isLoading: isCheckingConnection } = useGetTelemetryInternetConnectionQuery();
    const { data: settings } = useGetSettingsQuery();
    const [updateSettings] = usePutSettingsMutation();

    // Don't show modal if no internet connection
    useEffect(() => {
        if (!isCheckingConnection && !internetConnection) {
            onClose();
        }
    }, [internetConnection, isCheckingConnection, onClose]);

    const handleSubmit = async () => {
        if (!settings || isSubmitting) return;

        setIsSubmitting(true);
        try {
            // Update settings with selected telemetry mode
            await updateSettings({
                settings: {
                    ...settings,
                    telemetry_mode: selectedMode,
                },
            }).unwrap();

            // Configure telemetry service
            telemetryService.configure(selectedMode as TelemetryMode);

            onClose();
        } catch (error) {
            console.error('Failed to update telemetry settings:', error);
            // Handle error - maybe show toast
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleModeChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setSelectedMode(event.target.value as Telemetry_mode);
    };

    // Show loading if checking connection
    if (isCheckingConnection) {
        return (
            <Dialog open={open} maxWidth="sm" fullWidth>
                <DialogContent sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
                    <CircularProgress />
                </DialogContent>
            </Dialog>
        );
    }

    // Don't show modal if no internet connection
    if (!internetConnection) {
        return null;
    }

    return (
        <Dialog
            open={open}
            onClose={() => { }} // Prevent closing by clicking outside
            maxWidth="md"
            fullWidth
            disableEscapeKeyDown // Prevent closing with Escape key
        >
            <DialogTitle>
                Help Improve SRAT
            </DialogTitle>
            <DialogContent>
                <Typography variant="body1" paragraph>
                    Help us improve SRAT by sharing anonymous usage data and error reports.
                    This helps us identify issues and improve the software for everyone.
                </Typography>

                <Alert severity="info" sx={{ mb: 2 }}>
                    <Typography variant="body2">
                        All data is sent securely to Rollbar servers and is used solely for improving the software.
                        No personal information or file contents are transmitted.
                    </Typography>
                </Alert>

                <Box sx={{ mt: 2 }}>
                    <Typography variant="h6" gutterBottom>
                        Choose your preference:
                    </Typography>

                    <RadioGroup value={selectedMode} onChange={handleModeChange}>
                        <FormControlLabel
                            value={Telemetry_mode.All}
                            control={<Radio />}
                            label={
                                <Box>
                                    <Typography variant="body1" fontWeight="medium">
                                        Send usage data and error reports
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary">
                                        Help us improve SRAT by sharing anonymous usage statistics and error reports
                                    </Typography>
                                </Box>
                            }
                        />
                        <FormControlLabel
                            value={Telemetry_mode.Errors}
                            control={<Radio />}
                            label={
                                <Box>
                                    <Typography variant="body1" fontWeight="medium">
                                        Send only error reports
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary">
                                        Share only error reports to help us fix bugs and improve stability
                                    </Typography>
                                </Box>
                            }
                        />
                        <FormControlLabel
                            value={Telemetry_mode.Disabled}
                            control={<Radio />}
                            label={
                                <Box>
                                    <Typography variant="body1" fontWeight="medium">
                                        Don't send any data
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary">
                                        No data will be sent to external servers
                                    </Typography>
                                </Box>
                            }
                        />
                    </RadioGroup>
                </Box>

                <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
                    You can change this setting at any time in the Settings page.
                    For more information about data collection, visit our{' '}
                    <Link href="https://github.com/dianlight/srat/blob/main/PRIVACY.md" target="_blank">
                        privacy policy
                    </Link>.
                </Typography>
            </DialogContent>
            <DialogActions sx={{ p: 2 }}>
                <Button
                    onClick={handleSubmit}
                    variant="contained"
                    disabled={isSubmitting}
                    fullWidth
                >
                    {isSubmitting ? 'Saving...' : 'Continue'}
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default TelemetryModal;
