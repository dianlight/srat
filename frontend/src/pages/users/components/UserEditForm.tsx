import {
    Box,
    Button,
    CardContent,
    Grid,
    Stack,
    Typography,
} from "@mui/material";
import { useForm } from "react-hook-form";
import {
    PasswordElement,
    TextFieldElement
} from "react-hook-form-mui";
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
        formState: { isDirty },
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
                                required={isNewUser}
                                fullWidth
                                control={control}
                                disabled={disabled}
                                rules={{
                                    required: isNewUser ? "Password is required" : false,
                                    validate: (value) => {
                                        const password = value?.trim() || "";
                                        if (!password) {
                                            return true;
                                        }

                                        return password.length >= 4
                                            || "Password must be at least 4 characters";
                                    },
                                }}
                            />
                            <PasswordElement
                                name="password-repeat"
                                autoComplete="new-password"
                                label="Repeat Password"
                                required={isNewUser}
                                fullWidth
                                control={control}
                                disabled={disabled}
                                rules={{
                                    required: isNewUser ? "Repeat Password is required" : false,
                                    validate: (value, formValues) => {
                                        const password = formValues.password?.trim() || "";
                                        const repeatPassword = value?.trim() || "";

                                        if (!password && !repeatPassword) {
                                            return true;
                                        }

                                        return password === repeatPassword
                                            || "Passwords do not match";
                                    },
                                }}
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
