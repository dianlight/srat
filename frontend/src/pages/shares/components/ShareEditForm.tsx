import ModeEditIcon from "@mui/icons-material/ModeEdit";
import PlaylistAddIcon from "@mui/icons-material/PlaylistAdd";
import {
    Box,
    Button,
    Card,
    CardContent,
    CardHeader,
    Chip,
    Grid,
    IconButton,
    InputAdornment,
    Stack,
    Tooltip,
    Typography,
} from "@mui/material";
import { MuiChipsInput } from "mui-chips-input";
import { Fragment, useEffect, useMemo, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import {
    AutocompleteElement,
    CheckboxElement,
    SelectElement,
    SwitchElement,
    TextFieldElement,
} from "react-hook-form-mui";
import { useVolume } from "../../../hooks/volumeHook";
import default_json from "../../../json/default_config.json";
import {
    type MountPointData,
    type SharedResource,
    type User,
    Time_machine_support,
    Usage,
    useGetApiUsersQuery,
} from "../../../store/sratApi";
import type { ShareEditProps } from "../types";
import {
    casingCycleOrder,
    getCasingIcon,
    getPathBaseName,
    isValidVetoFileEntry,
    sanitizeAndUppercaseShareName,
    toCamelCase,
    toKebabCase,
} from "../utils";
import { filesize } from "filesize";

interface ShareEditFormProps {
    shareData?: ShareEditProps;
    shares?: Record<string, SharedResource> | SharedResource[];
    onSubmit: (data: ShareEditProps) => void;
    onDelete?: (shareName: string, shareData: SharedResource) => void;
    disabled?: boolean;
    showActions?: boolean; // Control whether to show form actions (buttons)
    variant?: 'card' | 'plain'; // Control the wrapper (card vs plain content)
    testOverrides?: {
        useGetApiUsersQuery?: typeof useGetApiUsersQuery;
        useVolume?: typeof useVolume;
    };
}

export function ShareEditForm({
    shareData,
    shares,
    onSubmit,
    onDelete,
    disabled = false,
    showActions = true,
    variant = 'card',
    testOverrides,
}: ShareEditFormProps) {
    const useUsersQuery = testOverrides?.useGetApiUsersQuery ?? useGetApiUsersQuery;
    const useVolumeHook = testOverrides?.useVolume ?? useVolume;
    const {
        data: users,
        isLoading: usLoading,
        error: usError,
    } = useUsersQuery();
    const { disks: volumes, isLoading: vlLoading, error: vlError } = useVolumeHook();
    const [editName, setEditName] = useState(shareData?.org_name === undefined);
    const [activeCasingIndex, setActiveCasingIndex] = useState(0);

    const adminUser = useMemo(() => {
        return Array.isArray(users)
            ? users.find((u) => u.is_admin)
            : undefined;
    }, [users])

    const {
        control,
        handleSubmit,
        watch,
        formState: { errors },
        reset,
        setValue,
        getValues,
    } = useForm<ShareEditProps>({
        defaultValues: {
            org_name: shareData?.org_name,
            name: shareData?.name || "",
            mount_point_data: shareData?.mount_point_data,
            users:
                shareData?.mount_point_data?.is_write_supported ?
                    ((shareData.org_name === undefined) &&
                        (!shareData?.users ||
                            shareData.users.length === 0) &&
                        adminUser
                        ? [adminUser]
                        : shareData.users || []) : [],
            ro_users: shareData?.mount_point_data?.is_write_supported ?
                (shareData?.ro_users || []) : ((shareData?.org_name === undefined) &&
                    (!shareData?.ro_users ||
                        shareData.ro_users.length === 0) &&
                    adminUser
                    ? [adminUser]
                    : shareData?.ro_users || []),
            timemachine: shareData?.mount_point_data?.time_machine_support === Time_machine_support.Unsupported ? false : (shareData?.timemachine || false),
            recycle_bin_enabled: (shareData?.recycle_bin_enabled || false),
            guest_ok: shareData?.guest_ok || false,
            timemachine_max_size: shareData?.timemachine_max_size ||
                (shareData?.mount_point_data?.disk_size ? filesize(shareData?.mount_point_data?.disk_size, {
                    round: 0,

                }) : "MAX"),
            usage: shareData?.usage || Usage.None,
            veto_files: shareData?.veto_files || [],
            disabled: shareData?.disabled,
        }
    });
    const isDisabled = watch("disabled") || disabled;
    const [availablePartitions, setAvailablePartition] = useState<MountPointData[]>([]);

    useEffect(() => {
        if (volumes) {
            const newAvailablePartitions = volumes
                ?.flatMap((disk) => disk.partitions)
                ?.filter(Boolean)
                .filter(
                    (partition) =>
                        !(partition?.system && partition?.host_mount_point_data && partition?.host_mount_point_data.length > 0)
                )
                .filter((partition) => partition?.mount_point_data)
                .flatMap(
                    (partition) => partition?.mount_point_data,
                )
                .filter(
                    (mp) => mp?.path !== "",
                ) as MountPointData[] || [];
            setAvailablePartition(newAvailablePartitions);
        }
    }, [volumes]);


    //setEditName(shareData?.org_name === undefined);

    // Effect to auto-populate share name if empty when a volume is selected
    const selectedMountPointData = watch("mount_point_data");
    const currentShareName = watch("name");

    useEffect(() => {
        if (
            (!currentShareName || currentShareName.trim() === "") &&
            selectedMountPointData &&
            selectedMountPointData.path
        ) {
            const baseName = getPathBaseName(selectedMountPointData.path);
            if (baseName) {
                const suggestedName = sanitizeAndUppercaseShareName(baseName);
                if (currentShareName !== suggestedName) {
                    setValue("name", suggestedName, {
                        shouldValidate: true,
                        shouldDirty: true,
                    });
                    setActiveCasingIndex(0);
                }
            }
        }
    }, [selectedMountPointData, currentShareName, setValue]);

    const handleCycleCasing = () => {
        const currentName = watch("name");
        if (typeof currentName !== "string") return;

        const styleToApply = casingCycleOrder[activeCasingIndex];
        let transformedName = currentName;

        switch (styleToApply) {
            case "UPPERCASE":
                transformedName = currentName.toUpperCase();
                break;
            case "lowercase":
                transformedName = currentName.toLowerCase();
                break;
            case "camelCase":
                transformedName = toCamelCase(currentName);
                break;
            case "kebab-case":
                transformedName = toKebabCase(currentName);
                break;
        }
        setValue("name", transformedName, {
            shouldValidate: true,
            shouldDirty: true,
        });
        setActiveCasingIndex(
            (prevIndex) => (prevIndex + 1) % casingCycleOrder.length,
        );
    };

    // activeCasingIndex is guaranteed to be a valid index due to modulo operation
    const nextCasingStyleName = casingCycleOrder[activeCasingIndex]!;
    const cycleCasingTooltipTitle = `Cycle casing (Next: ${nextCasingStyleName.charAt(0).toUpperCase() + nextCasingStyleName.slice(1)})`;
    const CasingIconToDisplay = getCasingIcon(nextCasingStyleName);

    const renderFormContent = () => (
        <>
            <form
                id="editshareform"
                onSubmit={handleSubmit(onSubmit)}
                noValidate
            >
                <Grid container spacing={2}>
                    {watch("usage") !== Usage.Internal && (
                        <Grid size={{ xs: 12, md: 8 }}>
                            {availablePartitions.length > 0 && (
                                <AutocompleteElement
                                    label="Volume"
                                    name="mount_point_data"
                                    options={availablePartitions}
                                    control={control}
                                    required
                                    loading={vlLoading}
                                    autocompleteProps={{
                                        disabled: isDisabled,
                                        size: "small",
                                        renderValue: (value: MountPointData) => {
                                            return <Typography variant="body2">
                                                {value.disk_label || value.device_id} <sup>{value.is_write_supported ? "" : (<Typography variant="caption" color="error">Read-Only</Typography>)}</sup>
                                            </Typography>;
                                        },
                                        getOptionLabel: (option) =>
                                            (option as MountPointData)?.disk_label || "",
                                        getOptionKey: (option) =>
                                            (option as MountPointData)?.path_hash || "",
                                        renderOption: (props, option) => (
                                            <li {...props}>
                                                <Typography variant="body2">
                                                    {option.disk_label || option.device_id} <sup>{option.is_write_supported ? "" : (<Typography variant="caption" color="error">Read-Only</Typography>)}</sup>
                                                </Typography>
                                            </li>
                                        ),
                                        isOptionEqualToValue(option, value) {
                                            if (!value || !option) return false;
                                            return option.path_hash === value?.path_hash;
                                        },
                                        getOptionDisabled: (option) => {
                                            if (!shares || !option.path_hash) {
                                                return false;
                                            }

                                            const currentEditingShareName = shareData?.org_name;

                                            for (const existingShare of Object.values(shares)) {
                                                if (
                                                    existingShare.mount_point_data?.path_hash ===
                                                    option.path_hash
                                                ) {
                                                    if (
                                                        currentEditingShareName &&
                                                        existingShare.name === currentEditingShareName
                                                    ) {
                                                        return false;
                                                    }
                                                    return true;
                                                }
                                            }
                                            return false;
                                        },
                                    }}
                                />
                            )}
                        </Grid>
                    )}
                    {watch("usage") !== Usage.Internal && (
                        <Grid size={{ xs: 12, md: 4 }}>
                            <SelectElement
                                sx={{ display: "flex" }}
                                size="small"
                                label="Usage"
                                name="usage"
                                disabled={isDisabled}
                                options={Object.values(Usage)
                                    .filter(
                                        (usage) =>
                                            usage !== Usage.Internal,
                                    )
                                    .map((usage) => {
                                        return { id: usage, label: usage.charAt(0).toUpperCase() + usage.slice(1) };
                                    })}
                                required
                                control={control}
                            />
                        </Grid>
                    )}

                    <Grid size={{ xs: 12 }}>
                        <Controller
                            name="veto_files"
                            control={control}
                            defaultValue={[]}
                            rules={{
                                validate: (chips: string[] | undefined) => {
                                    if (
                                        !chips ||
                                        chips == null ||
                                        chips.length === 0
                                    )
                                        return true;
                                    for (const chip of chips) {
                                        if (!isValidVetoFileEntry(chip)) {
                                            return `Invalid entry: "${chip}". Veto file entries cannot be empty, contain '/' or null characters.`;
                                        }
                                    }
                                    return true;
                                },
                            }}
                            render={({ field, fieldState: { error } }) => (
                                <MuiChipsInput
                                    {...field}
                                    disabled={isDisabled}
                                    size="small"
                                    fullWidth
                                    hideClearAll
                                    label="Veto Files"
                                    validate={(chipValue) =>
                                        typeof chipValue === "string" &&
                                        isValidVetoFileEntry(chipValue)
                                    }
                                    error={!!error}
                                    helperText={
                                        error
                                            ? error.message
                                            : "List of files/patterns to hide (e.g., ._* Thumbs.db). Entries cannot contain '/'."
                                    }
                                    renderChip={(Component, key, props) => {
                                        const isDefault =
                                            default_json.veto_files?.includes(
                                                props.label as string,
                                            );
                                        return (
                                            <Component
                                                key={key}
                                                {...props}
                                                sx={{
                                                    color: isDefault
                                                        ? "text.secondary"
                                                        : "text.primary",
                                                }}
                                                size="small"
                                            />
                                        );
                                    }}
                                    slotProps={{
                                        input: {
                                            endAdornment: (
                                                <InputAdornment
                                                    position="end"
                                                    sx={{ pr: 1 }}
                                                >
                                                    <Tooltip title="Add suggested default Veto files">
                                                        <span>
                                                            <IconButton
                                                                disabled={isDisabled}
                                                                aria-label="add suggested default veto files"
                                                                onClick={() => {
                                                                    const currentVetoFiles:
                                                                        string[] =
                                                                        getValues(
                                                                            "veto_files",
                                                                        ) || [];
                                                                    const defaultVetoFiles:
                                                                        string[] =
                                                                        default_json.veto_files ||
                                                                        [];
                                                                    const newVetoFilesToAdd =
                                                                        defaultVetoFiles.filter(
                                                                            (defaultFile) =>
                                                                                !currentVetoFiles.includes(
                                                                                    defaultFile,
                                                                                ),
                                                                        );
                                                                    setValue(
                                                                        "veto_files",
                                                                        [
                                                                            ...currentVetoFiles,
                                                                            ...newVetoFilesToAdd,
                                                                        ],
                                                                        {
                                                                            shouldDirty: true,
                                                                            shouldValidate: true,
                                                                        },
                                                                    );
                                                                }}
                                                                edge="end"
                                                                size="small"
                                                            >
                                                                <PlaylistAddIcon />
                                                            </IconButton>
                                                        </span>
                                                    </Tooltip>
                                                </InputAdornment>
                                            ),
                                        },
                                    }}
                                />
                            )}
                        />
                    </Grid>

                    {watch("mount_point_data")?.is_write_supported && (
                        <Grid size={{ xs: 12, sm: 6 }}>
                            <Tooltip
                                title={`Time Machine is ${watch("mount_point_data")?.time_machine_support} for the current volume!`}
                            >
                                <span>
                                    <SwitchElement
                                        switchProps={{
                                            size: "small",
                                            color: watch("mount_point_data")?.time_machine_support !== Time_machine_support.Supported ? "error" : "primary"
                                        }}
                                        label="Support Timemachine Backups"
                                        slotProps={{
                                            typography: {
                                                fontSize: "0.875rem",
                                                color: watch("mount_point_data")?.time_machine_support !== Time_machine_support.Supported ? "error" : "default"
                                            },
                                        }}
                                        name="timemachine"
                                        disabled={isDisabled || watch("mount_point_data")?.time_machine_support === Time_machine_support.Unsupported}
                                        control={control}
                                    />
                                </span>
                            </Tooltip>
                        </Grid>
                    )}

                    {watch("timemachine") && (
                        <Grid size={{ xs: 12, sm: 6 }}>
                            <TextFieldElement
                                size="small"
                                label="Time Machine Max Size (e.g., 100G, 5T, MAX)"
                                name="timemachine_max_size"
                                sx={{ display: "flex" }}
                                disabled={isDisabled}
                                control={control}
                                rules={{
                                    pattern: {
                                        value: /^(MAX|\d+\s*[KMGTP]B{0,1}?)$/i,
                                        message: "Invalid format. Use MAX or a number followed by K, M, G, T, P (e.g., 100G, 5T).",
                                    },
                                }}
                            />
                        </Grid>
                    )}

                    {watch("mount_point_data")?.is_write_supported && (
                        <Grid size={{ xs: 12, sm: 6 }}>
                            <SwitchElement
                                switchProps={{
                                    size: "small",
                                }}
                                slotProps={{
                                    typography: {
                                        fontSize: "0.875rem",
                                    },
                                }}
                                label="Support Recycle Bin"
                                name="recycle_bin_enabled"
                                disabled={isDisabled}
                                control={control}
                            />
                        </Grid>
                    )}

                    <Grid size={{ xs: 12, sm: 6 }}>
                        <SwitchElement
                            switchProps={{
                                size: "small",
                            }}
                            slotProps={{
                                typography: {
                                    fontSize: "0.875rem",
                                },
                            }}
                            label="Guest Access"
                            name="guest_ok"
                            disabled={isDisabled}
                            control={control}
                        />
                    </Grid>

                    {!watch("guest_ok") && (
                        <Grid size={{ xs: 12, sm: 6 }}>
                            {!usLoading && ((users as User[]) || []).length > 0 && (
                                <AutocompleteElement
                                    multiple
                                    name="users"
                                    label="Read and Write users"
                                    options={usLoading ? [] : (users as User[]) || []}
                                    control={control}
                                    loading={usLoading}
                                    autocompleteProps={{
                                        disabled: isDisabled || watch("mount_point_data")?.is_write_supported === false,
                                        size: "small",
                                        limitTags: 3,
                                        getOptionKey: (option) =>
                                            (option as User).username || "",
                                        getOptionLabel: (option) =>
                                            (option as User).username || "",
                                        renderOption: (props, option) => (
                                            <li {...props} key={props.key}>
                                                <Typography
                                                    variant="body2"
                                                    color={option.is_admin ? "warning" : "default"}
                                                >
                                                    {option.username}
                                                </Typography>
                                            </li>
                                        ),
                                        getOptionDisabled: (option) => {
                                            if (
                                                watch("ro_users")?.find(
                                                    (user) =>
                                                        user.username === option.username,
                                                )
                                            ) {
                                                return true;
                                            }
                                            return false;
                                        },
                                        isOptionEqualToValue(option, value) {
                                            return option.username === value.username;
                                        },
                                        renderValue: (values, getItemProps) =>
                                            values.map((option, index) => {
                                                const { key, ...itemProps } = getItemProps({
                                                    index,
                                                });
                                                return (
                                                    <Chip
                                                        color={
                                                            (option as User).is_admin
                                                                ? "warning"
                                                                : "default"
                                                        }
                                                        key={key}
                                                        variant="outlined"
                                                        label={
                                                            (option as User)?.username || "unknown"
                                                        }
                                                        size="small"
                                                        {...itemProps}
                                                    />
                                                );
                                            }),
                                    }}
                                    textFieldProps={{
                                        InputLabelProps: { shrink: true },
                                    }}
                                />
                            )}
                        </Grid>
                    )}

                    {!watch("guest_ok") && (
                        <Grid size={{ xs: 12, sm: 6 }}>
                            {!usLoading && ((users as User[]) || []).length > 0 && (
                                <AutocompleteElement
                                    multiple
                                    name="ro_users"
                                    label="Read Only users"
                                    options={usLoading ? [] : (users as User[]) || []}
                                    control={control}
                                    loading={usLoading}
                                    autocompleteProps={{
                                        disabled: isDisabled,
                                        size: "small",
                                        limitTags: 3,
                                        getOptionKey: (option) =>
                                            (option as User).username || "",
                                        getOptionLabel: (option) =>
                                            (option as User).username || "",
                                        renderOption: (props, option) => (
                                            <li {...props} key={props.key + "@ro"}>
                                                <Typography
                                                    variant="body2"
                                                    color={option.is_admin ? "warning" : "default"}
                                                >
                                                    {option.username}
                                                </Typography>
                                            </li>
                                        ),
                                        getOptionDisabled: (option) => {
                                            if (
                                                watch("users")?.find(
                                                    (user) =>
                                                        user.username === option.username,
                                                )
                                            ) {
                                                return true;
                                            }
                                            return false;
                                        },
                                        isOptionEqualToValue(option, value) {
                                            return option.username === value.username;
                                        },
                                        renderValue: (values, getItemProps) =>
                                            values.map((option, index) => {
                                                const { key, ...itemProps } = getItemProps({
                                                    index,
                                                });
                                                return (
                                                    <Chip
                                                        color={
                                                            (option as User).is_admin
                                                                ? "warning"
                                                                : "default"
                                                        }
                                                        key={key + "@ro"}
                                                        variant="outlined"
                                                        label={
                                                            (option as User)?.username || "unknown"
                                                        }
                                                        size="small"
                                                        {...itemProps}
                                                    />
                                                );
                                            }),
                                    }}
                                    textFieldProps={{
                                        InputLabelProps: { shrink: true },
                                    }}
                                />
                            )}
                        </Grid>
                    )}
                </Grid>

                {showActions && (
                    <Box sx={{ mt: 3, display: "flex", gap: 1, justifyContent: "flex-end" }}>
                        {shareData?.org_name && onDelete && (
                            <Button
                                onClick={() => {
                                    if (shareData?.org_name && onDelete) {
                                        onDelete(shareData.org_name, shareData);
                                    }
                                }}
                                color="error"
                                variant="outlined"
                                size="small"
                            >
                                Delete
                            </Button>
                        )}
                        <Button
                            type="submit"
                            variant="contained"
                            color="primary"
                            size="small"
                            disabled={isDisabled}
                        >
                            {shareData?.org_name === undefined ? "Create" : "Apply"}
                        </Button>
                    </Box>
                )}
            </form>
        </>
    );

    const renderNameHeader = () => (
        <Stack direction="row" spacing={2} alignItems="center" sx={{ flex: "auto" }}>
            {!(editName || shareData?.org_name === undefined) && (
                <Box sx={{ display: "flex", alignItems: "center", flexGrow: "inherit" }}>
                    <Tooltip title={shareData?.usage === Usage.Internal ? "Cannot edit name of internal shares" : "Edit share name"}>
                        <span>
                            <IconButton
                                onClick={() => setEditName(true)}
                                size="small"
                                disabled={shareData?.usage === Usage.Internal}
                            >
                                <ModeEditIcon fontSize="small" />
                            </IconButton>
                        </span>
                    </Tooltip>
                    <Typography variant="h6">{shareData?.name}</Typography>
                </Box>
            )}
            {(editName || shareData?.org_name === undefined) && (
                <TextFieldElement
                    sx={{ display: "flex", flexGrow: "inherit" }}
                    name="name"
                    label="Share Name"
                    required
                    size="small"
                    disabled={isDisabled}
                    rules={{
                        required: "Share name is required",
                        pattern: {
                            value: /^[a-zA-Z0-9_]+$/,
                            message:
                                "Share name can only contain letters, numbers, and underscores (_)",
                        },
                        maxLength: {
                            value: 80,
                            message: "Share name cannot exceed 80 characters",
                        },
                    }}
                    control={control}
                    slotProps={{
                        input: {
                            endAdornment: (
                                <InputAdornment position="end">
                                    <Tooltip title={cycleCasingTooltipTitle}>
                                        <IconButton
                                            aria-label="cycle share name casing"
                                            onClick={handleCycleCasing}
                                            edge="end"
                                            size="small"
                                        >
                                            <CasingIconToDisplay />
                                        </IconButton>
                                    </Tooltip>
                                </InputAdornment>
                            ),
                        },
                    }}
                />
            )}

            {shareData?.org_name !== undefined && (
                <SwitchElement
                    switchProps={{
                        size: "small",
                        color: isDisabled ? "text.secondary" : "text.primary",
                    }}
                    slotProps={{
                        typography: {
                            fontSize: "0.875rem",
                        },
                    }}
                    control={control}
                    name="disabled"
                    label={isDisabled ? "Disabled" : "Enabled"}
                    sx={{ mr: 0 }}
                />
            )}
        </Stack>
    );

    if (variant === 'plain') {
        return (
            <>
                {renderNameHeader()}
                <br />
                {renderFormContent()}
            </>
        );
    }

    return (
        <Card>
            <CardHeader title={renderNameHeader()} />
            <CardContent>
                {renderFormContent()}
            </CardContent>
        </Card>
    );
}