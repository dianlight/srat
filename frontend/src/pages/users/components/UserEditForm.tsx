import {
    Box,
    Button,
    CardContent,
    Grid,
    Stack,
    Typography,
} from "@mui/material";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import {
    PasswordElement,
    PasswordRepeatElement,
    TextFieldElement,
} from "react-hook-form-mui";
import type { User } from "../../../store/sratApi";
import type { UsersProps } from "../types";

interface UserEditFormProps {
    userData?: UsersProps;
    onSubmit: (data: UsersProps) => void;
    onCancel?: () => void;
    disabled?: boolean;
}

export function UserEditForm({
    userData,
    onSubmit,
    onCancel,
    disabled = false,
}: UserEditFormProps) {
    const isNewUser = userData?.doCreate === true;
    const isAdmin = userData?.is_admin === true;

    const {
        control,
        handleSubmit,
        reset,
        formState: { errors, isValid, isDirty },
    } = useForm<UsersProps>({
        defaultValues: {
            username: "",
            password: "",
            is_admin: false,
            doCreate: false,
        },
        values: userData,
        mode: "onChange",
    });
    /*
        useEffect(() => {
            reset(userData);
        }, [userData, reset]);
    */
    const handleFormSubmit = (data: UsersProps) => {
        // Trim whitespace from username and password
        const trimmedData: UsersProps = {
            ...data,
            username: data.username?.trim() || "",
            password: data.password?.trim() || "",
        };
        onSubmit(trimmedData);
    };

    return (
        <CardContent>
            <form onSubmit={handleSubmit(handleFormSubmit)} noValidate>
                <Grid container spacing={3}>
                    {/* Username Field */}
                    <Grid size={{ xs: 12, md: 6 }}>
                        <TextFieldElement
                            name="username"
                            autoComplete="username"
                            label="Username"
                            required
                            fullWidth
                            control={control}
                            disabled={disabled}
                            slotProps={{
                                // Admin can rename their username
                                // Non-admin existing users cannot change username
                                // New users can set username
                                input: {
                                    readOnly: !isNewUser && !isAdmin,
                                },
                            }}
                            helperText={
                                isAdmin
                                    ? "Admin username can be changed"
                                    : !isNewUser
                                        ? "Username cannot be changed for existing users"
                                        : undefined
                            }
                            rules={{
                                required: "Username is required",
                                minLength: {
                                    value: 2,
                                    message: "Username must be at least 2 characters",
                                },
                                pattern: {
                                    value: /^[a-zA-Z0-9_-]+$/,
                                    message: "Username can only contain letters, numbers, underscores, and hyphens",
                                },
                            }}
                        />
                    </Grid>

                    {/* Password Fields */}
                    <Grid size={{ xs: 12, md: 6 }}>
                        <Stack spacing={2}>
                            <PasswordElement
                                name="password"
                                autoComplete="new-password"
                                label="Password"
                                required
                                fullWidth
                                control={control}
                                disabled={disabled}
                                rules={{
                                    required: "Password is required",
                                    minLength: {
                                        value: 4,
                                        message: "Password must be at least 4 characters",
                                    },
                                }}
                            />
                            <PasswordRepeatElement
                                passwordFieldName="password"
                                name="password-repeat"
                                autoComplete="new-password"
                                label="Repeat Password"
                                required
                                fullWidth
                                control={control}
                                disabled={disabled}
                            />
                        </Stack>
                    </Grid>

                    {/* User Type Information */}
                    <Grid size={12}>
                        <Box sx={{ bgcolor: "action.hover", p: 2, borderRadius: 1 }}>
                            <Typography variant="body2" color="text.secondary">
                                {isAdmin ? (
                                    <>
                                        <strong>Administrator Account:</strong> You can change the admin username and password.
                                        The admin account cannot be deleted, but there can only be one admin user.
                                    </>
                                ) : isNewUser ? (
                                    <>
                                        <strong>New User:</strong> Create a new regular user account.
                                        Users can be assigned read/write or read-only access to shares.
                                    </>
                                ) : (
                                    <>
                                        <strong>Regular User:</strong> You can update the password for this user.
                                        The username cannot be changed after creation.
                                    </>
                                )}
                            </Typography>
                        </Box>
                    </Grid>

                    {/* Form Actions */}
                    <Grid size={12}>
                        <Stack direction="row" spacing={2} justifyContent="flex-end">
                            {onCancel && (
                                <Button
                                    variant="outlined"
                                    color="secondary"
                                    onClick={onCancel}
                                    disabled={disabled}
                                >
                                    Cancel
                                </Button>
                            )}
                            <Button
                                type="submit"
                                variant="contained"
                                color="primary"
                                disabled={disabled || !isDirty}
                            >
                                {isNewUser ? "Create User" : "Save Changes"}
                            </Button>
                        </Stack>
                    </Grid>
                </Grid>
            </form>
        </CardContent>
    );
}
