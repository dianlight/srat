import React, { useState, useEffect } from "react";
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Typography,
    Button,
    Box,
    Alert,
    TextField,
    Stack,
} from "@mui/material";
import {
    usePutApiSettingsMutation,
    usePutApiUseradminMutation,
    useGetApiSettingsQuery,
    useGetApiUsersQuery,
    type Settings,
    type User,
} from "../store/sratApi";

interface BaseConfigModalProps {
    open: boolean;
    onClose: () => void;
}

const BaseConfigModal: React.FC<BaseConfigModalProps> = ({ open, onClose }) => {
    const [newPassword, setNewPassword] = useState("");
    const [confirmPassword, setConfirmPassword] = useState("");
    const [hostname, setHostname] = useState("");
    const [workgroup, setWorkgroup] = useState("");
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [passwordError, setPasswordError] = useState("");

    const { data: settings } = useGetApiSettingsQuery();
    const { data: users } = useGetApiUsersQuery();
    const [updateSettings] = usePutApiSettingsMutation();
    const [updateAdminUser] = usePutApiUseradminMutation();

    // Type guard to ensure settings is a Settings object and not an error
    const isValidSettings = (data: unknown): data is Settings => {
        return data !== null && typeof data === "object" && "hostname" in data;
    };

    // Type guard to ensure users is an array and not an error
    const isValidUsers = (data: unknown): data is User[] => {
        return Array.isArray(data);
    };

    useEffect(() => {
        if (isValidSettings(settings)) {
            setHostname(settings.hostname || "");
            setWorkgroup(settings.workgroup || "");
        }
    }, [settings]);

    const handleSubmit = async () => {
        // Validate passwords match
        if (newPassword !== confirmPassword) {
            setPasswordError("Passwords do not match");
            return;
        }

        // Validate password is not empty and not the default
        if (!newPassword || newPassword === "changeme!") {
            setPasswordError(
                "Password cannot be empty or the default password"
            );
            return;
        }

        // Validate password length
        if (newPassword.length < 6) {
            setPasswordError("Password must be at least 6 characters long");
            return;
        }

        if (!isValidUsers(users) || !isValidSettings(settings) || isSubmitting)
            return;

        const adminUser = users.find((u) => u.is_admin);
        if (!adminUser) return;

        setIsSubmitting(true);
        setPasswordError("");
        try {
            // Update admin user password
            await updateAdminUser({
                user: {
                    ...adminUser,
                    password: newPassword,
                },
            }).unwrap();

            // Update settings with general config fields
            await updateSettings({
                settings: {
                    ...settings,
                    hostname: hostname || undefined,
                    workgroup: workgroup || undefined,
                } as Settings,
            }).unwrap();

            onClose();
        } catch (error) {
            console.error("Failed to update settings:", error);
            setPasswordError("Failed to save changes. Please try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <Dialog
            open={open}
            onClose={() => { }} // Prevent closing by clicking outside
            maxWidth="md"
            fullWidth
            disableEscapeKeyDown // Prevent closing with Escape key
        >
            <DialogTitle>
                Secure Your System
            </DialogTitle>
            <DialogContent>
                <Typography variant="body1" paragraph sx={{ mt: 2 }}>
                    Welcome to SRAT (Samba Rest Administration Tool)! Your
                    system is using the default administrator password. For
                    security, you must change it now and configure basic system
                    settings to proceed.
                </Typography>

                <Alert severity="warning" sx={{ mb: 3 }}>
                    <Typography variant="body2">
                        The current default password is{" "}
                        <strong>changeme!</strong>. You must change this
                        immediately for security reasons.
                    </Typography>
                </Alert>

                <Stack spacing={2}>
                    <Box>
                        <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>
                            Change Administrator Password
                        </Typography>
                        <Typography
                            variant="body2"
                            color="text.secondary"
                            paragraph
                        >
                            Create a strong new password for your administrator
                            account.
                        </Typography>
                    </Box>

                    <TextField
                        type="password"
                        label="New Administrator Password"
                        value={newPassword}
                        onChange={(e) => {
                            setNewPassword(e.target.value);
                            setPasswordError("");
                        }}
                        fullWidth
                        placeholder="Enter new password"
                        helperText="Must be at least 6 characters"
                    />

                    <TextField
                        type="password"
                        label="Confirm Password"
                        value={confirmPassword}
                        onChange={(e) => {
                            setConfirmPassword(e.target.value);
                            setPasswordError("");
                        }}
                        fullWidth
                        placeholder="Confirm new password"
                        error={
                            passwordError.length > 0 &&
                            newPassword !== confirmPassword
                        }
                    />

                    {passwordError && (
                        <Alert severity="error">
                            <Typography variant="body2">
                                {passwordError}
                            </Typography>
                        </Alert>
                    )}

                    <Box>
                        <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>
                            General Configuration
                        </Typography>
                        <Typography
                            variant="body2"
                            color="text.secondary"
                            paragraph
                        >
                            Configure basic system settings for your Samba
                            server.
                        </Typography>
                    </Box>

                    <TextField
                        label="Hostname"
                        value={hostname}
                        onChange={(e) => setHostname(e.target.value)}
                        fullWidth
                        placeholder="e.g., samba-nas"
                        helperText="The name of your Samba server on the network"
                    />

                    <TextField
                        label="Workgroup"
                        value={workgroup}
                        onChange={(e) => setWorkgroup(e.target.value)}
                        fullWidth
                        placeholder="e.g., WORKGROUP"
                        helperText="The workgroup name for your Samba server"
                    />
                </Stack>

                <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{ mt: 3 }}
                >
                    You can change these settings later in the Settings page.
                </Typography>
            </DialogContent>
            <DialogActions sx={{ p: 2 }}>
                <Button
                    onClick={handleSubmit}
                    variant="contained"
                    disabled={
                        isSubmitting ||
                        !newPassword ||
                        !confirmPassword ||
                        newPassword !== confirmPassword
                    }
                    fullWidth
                >
                    {isSubmitting ? "Saving..." : "Secure System"}
                </Button>
            </DialogActions>
        </Dialog>
    );
};

export default BaseConfigModal;

