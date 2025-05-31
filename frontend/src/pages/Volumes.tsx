import { Fragment, useState, useEffect, useRef } from "react";
import { InView } from "react-intersection-observer";
import { PreviewDialog } from "../components/PreviewDialog";
import List from "@mui/material/List"; // Import Collapse and Chip
import { ListItemButton, ListItem, IconButton, ListItemAvatar, Avatar, ListItemText, Divider, Stack, Typography, Tooltip, Dialog, Button, DialogActions, DialogContent, DialogContentText, DialogTitle, Grid, Collapse, Chip, Switch, FormControlLabel, Autocomplete, TextField } from "@mui/material"; // Import Collapse and Chip, Switch, FormControlLabel
import ShareIcon from '@mui/icons-material/Share';
import AddIcon from '@mui/icons-material/Add';
import StorageIcon from '@mui/icons-material/Storage';
import CreditScoreIcon from '@mui/icons-material/CreditScore';
import ExpandLess from '@mui/icons-material/ExpandLess'; // Import expand icons
import ExpandMore from '@mui/icons-material/ExpandMore'; // Import expand icons
import { useConfirm } from "material-ui-confirm";
import { filesize } from "filesize";
import { faPlug, faPlugCircleXmark, faPlugCircleMinus } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeSvgIcon } from "../components/FontAwesomeSvgIcon";
import { AutocompleteElement, useForm, TextFieldElement, MultiSelectElement, Controller, useFieldArray, CheckboxElement } from "react-hook-form-mui"; // Import TextFieldElement
import { toast } from "react-toastify";
import { useVolume } from "../hooks/volumeHook";
import { useReadOnly } from "../hooks/readonlyHook";
import { useDeleteVolumeByMountPathHashMountMutation, useGetFilesystemsQuery, usePostVolumeByMountPathHashMountMutation, type Partition, type Disk, type MountPointData, Type, type FilesystemType, type MountFlag } from "../store/sratApi";
// Add EjectIcon to your imports
import EjectIcon from '@mui/icons-material/Eject';
import UsbIcon from '@mui/icons-material/Usb';
import SdStorageIcon from '@mui/icons-material/SdStorage';
// ... other icon imports ...
import ComputerIcon from '@mui/icons-material/Computer';
import SettingsSuggestIcon from '@mui/icons-material/SettingsSuggest';
import MD5 from "crypto-js/md5";

// --- Helper functions (decodeEscapeSequence, onSubmitMountVolume, etc.) remain the same ---
function decodeEscapeSequence(source: string) {
    // Basic check to avoid errors if source is not a string
    if (typeof source !== 'string') return '';
    return source.replace(/\\x([0-9A-Fa-f]{2})/g, function (_match, group1) {
        // Ensure group1 is treated as a string before parseInt
        return String.fromCharCode(parseInt(String(group1), 16));
    });
};


export function Volumes() {
    const read_only = useReadOnly();
    const [showPreview, setShowPreview] = useState<boolean>(false);
    const [showMount, setShowMount] = useState<boolean>(false);

    const [hideSystemPartitions, setHideSystemPartitions] = useState<boolean>(true); // Default to hide system partitions
    // Assuming useVolume returns an array of disk objects, where each disk has a 'partitions' array
    // If the structure is different (e.g., a flat list of partitions with parent info), the grouping logic needs adjustment
    const { disks, isLoading, error } = useVolume();
    const [selected, setSelected] = useState<Partition | Disk | undefined>(undefined); // Can hold a disk or partition
    const confirm = useConfirm();
    const [mountVolume, mountVolumeResult] = usePostVolumeByMountPathHashMountMutation();
    const [umountVolume, umountVolumeResult] = useDeleteVolumeByMountPathHashMountMutation();
    const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({}); // State for collapse, key is disk identifier
    const initialAutoOpenDone = useRef(false);

    useEffect(() => {
        // This effect runs when disks data is loaded to open the first visible disk "at start".
        // It should only run once for the initial auto-open.
        if (isLoading || !Array.isArray(disks) || disks.length === 0 || initialAutoOpenDone.current) {
            //console.log(isLoading, !Array.isArray(disks), initialAutoOpenDone.current)
            return;
        }

        //console.log("Rendering!", disks.length)

        // At this point, isLoading is false, disks is an array, and initial auto-open hasn't happened.
        if (disks.length > 0) {
            let firstVisibleDiskIdentifier: string | null = null;
            for (let i = 0; i < disks.length; i++) {
                const disk = disks[i];
                const diskIdentifier = disk.id || `disk-${i}`; // Must match identifier logic in render

                // Determine if this disk would be visible based on current hideSystemPartitions state
                const filteredPartitions = disk.partitions?.filter(p => !(hideSystemPartitions && p.system)) || [];
                const hasActualPartitions = disk.partitions && disk.partitions.length > 0;
                const allPartitionsAreHiddenByToggle = hasActualPartitions && filteredPartitions.length === 0 && hideSystemPartitions;

                //console.log(filteredPartitions.length, hideSystemPartitions, allPartitionsAreHiddenByToggle, diskIdentifier)

                if (!allPartitionsAreHiddenByToggle) {
                    firstVisibleDiskIdentifier = diskIdentifier;
                    break; // Found the first disk that will be rendered
                }
            }

            if (firstVisibleDiskIdentifier) {
                console.log("Opening first visible disk:", firstVisibleDiskIdentifier);
                setOpenGroups({ [firstVisibleDiskIdentifier]: true });
            }
        }
        initialAutoOpenDone.current = true; // Mark initial auto-open as done
    }, [disks, isLoading, hideSystemPartitions]); // hideSystemPartitions is included as its initial state affects the first visible disk


    // Toggle collapse state for a group (disk)
    const handleToggleGroup = (diskIdentifier: string) => {
        setOpenGroups(prev => ({ ...prev, [diskIdentifier]: !prev[diskIdentifier] }));
    };


    function onSubmitMountVolume(data?: MountPointData) {
        console.log("Mount Request Data:", data);
        // Type guard to check if selected is a Partition
        const isPartition = (item: any): item is Partition => item && !(item as Disk).partitions && item.name;

        if (!selected || !isPartition(selected) || !data || !data.path) {
            toast.error("Cannot mount: Invalid selection or missing data.");
            console.error("Mount validation failed:", { selected, isPartition: isPartition(selected), data });
            return;
        }

        // Ensure device is included in submitData if required by API
        const submitData: MountPointData = {
            ...data,
            device: selected.device
        };
        //console.log("Submitting Mount Data:", submitData);

        mountVolume({ mountPathHash: data.path_hash || "", mountPointData: submitData }).unwrap().then((res) => {
            toast.info(`Volume ${(res as MountPointData).path || selected.name} mounted successfully.`);
            setSelected(undefined); // Clear selection after successful mount
            setShowMount(false); // Close the mount dialog
        }).catch(err => {
            console.error("Mount Error:", err);
            const errorData = err?.data || {};
            const errorMsg = errorData?.detail || errorData?.message || err?.status || 'Unknown mount error';
            const errorCode = errorData?.status || 'Error';
            toast.error(`${errorCode}: ${errorMsg}`, { data: { error: errorData || err } });
        })
    }

    function handleCreateShare(partition: Partition) {
        // TODO: Implement navigation or action to create a share
        // const mountPath = partition.mount_point_data?.[0]?.path;
        // if (mountPath) navigate(`/shares/create?path=${mountPath}`);
        console.log("Create share for:", partition);
        toast.info("Create share functionality not yet implemented.");
    }

    function handleGoToShare(partition: Partition) {
        // TODO: Implement navigation or action to go to an existing share
        // const mountPath = partition.mount_point_data?.[0]?.path;
        // if (mountPath) navigate(`/shares?path=${mountPath}`); // Example navigation
        console.log("Go to share for:", partition);
        toast.info("Go to share functionality not yet implemented.");
    }

    function onSubmitUmountVolume(partition: Partition, force = false) {
        console.log("Umount Request", partition, "Force:", force);
        // Ensure mount_point_data exists and has at least one entry with a path
        const mountData = partition.mount_point_data?.[0];
        if (!mountData?.path) {
            toast.error("Cannot unmount: Missing mount point path.");
            console.error("Missing mount path for partition:", partition);
            return;
        }

        // Use partition label or name for confirmation dialog
        const displayName = decodeEscapeSequence(partition.name || 'this volume');

        confirm({
            title: `Unmount ${displayName}?`,
            description: `Do you really want to ${force ? "forcefully " : ""}unmount the Volume ${displayName} (${partition.device}) mounted at ${mountData.path}?`,
            confirmationText: force ? "Force Unmount" : "Unmount",
            cancellationText: "Cancel",
            confirmationButtonProps: { color: force ? "error" : "primary" },
            acknowledgement: `Please confirm this action carefully. Unmounting may lead to data loss or corruption if the volume is in use. ${force ? 'NOTE:Configured shares will be disabled!' : ''}`,
        })
            .then((crseult) => { // Only proceed if confirmed
                if (crseult.reason === "confirm") {
                    console.log(`Proceeding with ${force ? 'forced ' : ''}unmount for:`, mountData.path);
                    umountVolume({
                        mountPathHash: mountData.path_hash || "", // Use the extracted path
                        force: force,
                        lazy: true, // Consider if lazy unmount is always desired
                    }).unwrap().then(() => {
                        toast.info(`Volume ${displayName} unmounted successfully.`);
                        // Optionally clear selection if the unmounted item was selected
                        if (selected?.id === partition.id) {
                            setSelected(undefined);
                        }

                    }).catch(err => {
                        console.error("Unmount Error:", err);
                        const errorData = err?.data || {};
                        const errorMsg = errorData?.message || err?.status || 'Unknown error';
                        toast.error(`Error unmounting ${displayName}: ${errorMsg}`, { data: { error: err } });
                    })
                }
            })
    }


    // Handle loading and error states
    if (isLoading) {
        return <Typography>Loading volumes...</Typography>;
    }

    if (error) {
        // Provide a more user-friendly error message
        console.error("Error loading volumes:", error);
        return <Typography color="error">Error loading volume information. Please try again later.</Typography>;
    }

    // Ensure disks is an array before mapping
    const validDisks = Array.isArray(disks) ? disks : [];

    // Helper function to render disk icon
    const renderDiskIcon = (disk: Disk) => {
        switch (disk.connection_bus?.toLowerCase()) {
            case 'usb': return <UsbIcon />;
            case 'sdio': case 'mmc': return <SdStorageIcon />;
        }
        if (disk.removable) {
            return <EjectIcon />;
        }
        // Add more specific icons based on bus or type if needed
        // e.g., if (disk.type === 'nvme') return <MemoryIcon />;
        return <ComputerIcon />;
    };

    // Helper function to render partition icon
    const renderPartitionIcon = (partition: Partition) => {
        const isToMountAtStartup = partition.mount_point_data?.[0]?.is_to_mount_at_startup === true;
        const iconColorProp = isToMountAtStartup ? { color: "primary" as const } : {};

        if (partition.name === 'hassos-data') {
            return <CreditScoreIcon fontSize="small" {...iconColorProp} />;
        }
        if (partition.system) {
            return <SettingsSuggestIcon fontSize="small" {...iconColorProp} />;
        }
        return <StorageIcon fontSize="small" {...iconColorProp} />;
    };


    return (
        <InView>
            {/* Pass selected (could be disk or partition) to mount dialog */}
            <VolumeMountDialog
                // Type guard to ensure we only pass Partitions to the mount dialog
                objectToEdit={selected && !(selected as Disk).partitions && (selected as Partition).name ? selected as Partition : undefined}
                open={showMount}
                onClose={(data) => {
                    if (data) {
                        onSubmitMountVolume(data);
                    } else {
                        setSelected(undefined);
                        setShowMount(false);
                    }
                }} />
            {/* PreviewDialog can show details for both disks and partitions */}
            <PreviewDialog
                // Improved title logic using type guards
                title={
                    selected
                        ? (selected as Disk).model // If it has a model, it's likely a Disk
                            ? `Disk: ${(selected as Disk).model}`
                            : `Partition: ${decodeEscapeSequence((selected as Partition).name || 'Unknown')}` // Otherwise, assume Partition
                        : "Details"
                }
                objectToDisplay={selected}
                open={showPreview}
                onClose={() => {
                    setSelected(undefined);
                    setShowPreview(false);
                }} />
            <br />
            <Stack direction="row" justifyContent="flex-start" sx={{ pl: 2, mb: 1 }}>
                <FormControlLabel
                    control={
                        <Switch
                            checked={hideSystemPartitions}
                            onChange={(e) => setHideSystemPartitions(e.target.checked)}
                            name="hideSystemPartitions"
                            size="small"
                        />
                    }
                    label={<Typography variant="body2">Hide system partitions</Typography>}
                />
            </Stack>
            <List dense={true}>
                <Divider />
                {/* Iterate over disks */}
                {validDisks.map((disk, diskIdx) => {
                    const diskIdentifier = disk.id || `disk-${diskIdx}`;
                    const isGroupOpen = !!openGroups[diskIdentifier];

                    const filteredPartitions = disk.partitions?.filter(partition => !(hideSystemPartitions && partition.system)) || [];

                    // Determine if the disk itself should be hidden
                    // A disk is hidden if:
                    // 1. The "hideSystemPartitions" toggle is on.
                    // 2. The disk actually has partitions.
                    // 3. All of its partitions are system partitions (meaning filteredPartitions would be empty).
                    const hasActualPartitions = disk.partitions && disk.partitions.length > 0;
                    const allPartitionsAreHiddenByToggle = hasActualPartitions && filteredPartitions.length === 0 && hideSystemPartitions;

                    if (allPartitionsAreHiddenByToggle) {
                        return null; // Don't render this disk if all its partitions are hidden by the toggle
                    }

                    return (
                        <Fragment key={diskIdentifier}>
                            {/* Header Row for the Physical Disk */}
                            <ListItemButton
                                onClick={() => { setSelected(disk); setShowPreview(true); }}
                                sx={{ pl: 0, alignItems: 'flex-start' }} // Align items top for multi-line secondary
                            >
                                <ListItemAvatar sx={{ pt: 1 }}> {/* Add padding top to align avatar with first line */}
                                    <Avatar>
                                        {renderDiskIcon(disk)}
                                    </Avatar>
                                </ListItemAvatar>
                                <ListItemText
                                    primary={`Disk: ${disk.model?.toUpperCase() || `Disk ${diskIdx + 1}`}`}
                                    disableTypography
                                    secondary={
                                        <Stack spacing={0.5} sx={{ pt: 0.5 }}> {/* Stack for secondary info */}
                                            <Typography variant="caption" component="div"> {/* Wrap partition count */}
                                                {`${disk.partitions?.length || 0} partition(s)`}
                                            </Typography>
                                            {/* Conditionally display other disk details */}
                                            <Stack direction="row" spacing={1} flexWrap="wrap" alignItems="center">
                                                {disk.size != null && <Chip label={`Size: ${filesize(disk.size, { round: 1 })}`} size="small" variant="outlined" />}
                                                {disk.vendor && <Chip label={`Vendor: ${disk.vendor}`} size="small" variant="outlined" />}
                                                {disk.serial && <Chip label={`Serial: ${disk.serial}`} size="small" variant="outlined" />}
                                                {disk.connection_bus && <Chip label={`Bus: ${disk.connection_bus}`} size="small" variant="outlined" />}
                                                {disk.revision && <Chip label={`Rev: ${disk.revision}`} size="small" variant="outlined" />}
                                            </Stack>
                                        </Stack>
                                    }
                                />
                                {/* Conditionally render IconButton only if there are partitions */}
                                {disk.partitions && disk.partitions.length > 0 && (
                                    <IconButton
                                        edge="end"
                                        size="small"
                                        onClick={(e) => { e.stopPropagation(); handleToggleGroup(diskIdentifier); }}
                                        aria-label={isGroupOpen ? 'Collapse partitions' : 'Expand partitions'}
                                        aria-expanded={isGroupOpen}
                                        sx={{ mt: 0.5 }} // Adjust margin top to align better
                                    >
                                        {isGroupOpen ? <ExpandLess /> : <ExpandMore />}
                                    </IconButton>
                                )}
                            </ListItemButton>
                            {/* Collapsible Section for Partitions */}
                            {disk.partitions && disk.partitions.length > 0 && (
                                <Collapse in={isGroupOpen} timeout="auto" unmountOnExit>
                                    <List component="div" disablePadding dense={true} sx={{ pl: 4 }} >
                                        {filteredPartitions.map((partition, partIdx) => {
                                            const partitionIdentifier = partition.id || `${diskIdentifier}-part-${partIdx}`;
                                            const isMounted = partition.mount_point_data
                                                && partition.mount_point_data.length > 0
                                                && partition.mount_point_data.some(mpd => mpd.is_mounted);
                                            const hasShares = partition.mount_point_data
                                                && partition.mount_point_data.length > 0
                                                && partition.mount_point_data.some(mpd => {
                                                    return mpd.shares &&
                                                        mpd.shares.length > 0 &&
                                                        mpd.shares.some(share => !share.disabled)
                                                });

                                            const firstMountPath = partition.mount_point_data?.[0]?.path;
                                            const showShareActions = isMounted && firstMountPath?.startsWith("/mnt/");
                                            const partitionNameDecoded = decodeEscapeSequence(partition.name || "Unknown Partition");

                                            return (
                                                <Fragment key={partitionIdentifier}>
                                                    <ListItemButton
                                                        sx={{ pl: 1, alignItems: 'flex-start' }} // Align items top
                                                        onClick={() => { setSelected(partition); setShowPreview(true); }}
                                                    >
                                                        <ListItem
                                                            disablePadding
                                                            secondaryAction={!read_only && !partition.system && ( // Only show actions if not read-only and not system partition
                                                                (<Stack direction="row" spacing={0} alignItems="center" sx={{ pr: 1 }}> {/* Reduced spacing */}
                                                                    {!isMounted && (
                                                                        <Tooltip title="Mount Partition">
                                                                            <IconButton onClick={(e) => { e.stopPropagation(); setSelected(partition); setShowMount(true); }} edge="end" aria-label="mount" size="small">
                                                                                <FontAwesomeSvgIcon icon={faPlug} />
                                                                            </IconButton>
                                                                        </Tooltip>
                                                                    )}
                                                                    {isMounted && (
                                                                        <>
                                                                            {!hasShares && (
                                                                                <Tooltip title="Unmount Partition">
                                                                                    <IconButton onClick={(e) => { e.stopPropagation(); onSubmitUmountVolume(partition, false); }} edge="end" aria-label="unmount" size="small">
                                                                                        <FontAwesomeSvgIcon icon={faPlugCircleMinus} />
                                                                                    </IconButton>
                                                                                </Tooltip>
                                                                            )
                                                                            }
                                                                            <Tooltip title="Force Unmount Partition">
                                                                                <IconButton onClick={(e) => { e.stopPropagation(); onSubmitUmountVolume(partition, true); }} edge="end" aria-label="force unmount" size="small">
                                                                                    <FontAwesomeSvgIcon icon={faPlugCircleXmark} />
                                                                                </IconButton>
                                                                            </Tooltip>
                                                                            {(showShareActions && !hasShares) ? (
                                                                                <Tooltip title="Create Share">
                                                                                    <IconButton onClick={(e) => { e.stopPropagation(); handleCreateShare(partition); }} edge="end" aria-label="create share" size="small">
                                                                                        <AddIcon fontSize="small" />
                                                                                    </IconButton>
                                                                                </Tooltip>
                                                                            ) : showShareActions && (
                                                                                <Tooltip title="Go to Share">
                                                                                    <IconButton onClick={(e) => { e.stopPropagation(); handleGoToShare(partition); }} edge="end" aria-label="go to share" size="small">
                                                                                        <ShareIcon fontSize="small" />
                                                                                    </IconButton>
                                                                                </Tooltip>
                                                                            )}
                                                                        </>
                                                                    )}
                                                                </Stack>)
                                                            )}
                                                        >
                                                            <ListItemAvatar sx={{ minWidth: 'auto', pr: 1.5, pt: 0.5 }}> {/* Align avatar */}
                                                                <Avatar sx={{ width: 32, height: 32 }}>
                                                                    {renderPartitionIcon(partition)}
                                                                </Avatar>
                                                            </ListItemAvatar>
                                                            <ListItemText
                                                                primary={partitionNameDecoded}
                                                                disableTypography
                                                                secondary={
                                                                    <Stack spacing={1} direction="row" flexWrap="wrap" alignItems="center" sx={{ pt: 0.5 }}>
                                                                        {partition.size != null && <Chip label={`Size: ${filesize(partition.size, { round: 0 })}`} size="small" variant="outlined" />}
                                                                        {partition.mount_point_data?.[0]?.fstype && <Chip label={`Type: ${partition.mount_point_data[0].fstype}`} size="small" variant="outlined" />}
                                                                        {isMounted && <Chip label={`Mount: ${partition.mount_point_data?.map((mpd) => mpd.path).join(" ")}`} size="small" variant="outlined" />}
                                                                        {partition.host_mount_point_data && <Chip label={`Mount: ${partition.host_mount_point_data?.map((mpd) => mpd.path).join(" ")}`} size="small" variant="outlined" />}
                                                                        {partition.id && <Chip label={`UUID: ${partition.id}`} size="small" variant="outlined" />}
                                                                        {partition.device && <Chip label={`Dev: ${partition.device}`} size="small" variant="outlined" />}
                                                                    </Stack>
                                                                }
                                                            />
                                                        </ListItem>
                                                    </ListItemButton>
                                                    {partIdx < filteredPartitions.length - 1 && (
                                                        <Divider variant="inset" component="li" sx={{ ml: 4 }} />
                                                    )}
                                                </Fragment>
                                            );
                                        })}
                                        {isGroupOpen && disk.partitions && disk.partitions.length > 0 && filteredPartitions.length === 0 && hideSystemPartitions && (
                                            <ListItem dense sx={{ pl: 1 }}>
                                                <ListItemText
                                                    secondary="System partitions are hidden."
                                                    slotProps={{
                                                        secondary: { variant: 'caption', fontStyle: 'italic' }
                                                    }}
                                                />
                                            </ListItem>
                                        )}
                                    </List>

                                </Collapse>
                            )}
                            <Divider />
                        </Fragment>
                    );
                })}
            </List>
        </InView >
    );
}


interface xMountPointData extends MountPointData {
    custom_flags_values: MountFlag[]; // Array of custom flags (enum) for the TextField
}



function VolumeMountDialog(props: { open: boolean, onClose: (data?: MountPointData) => void, objectToEdit?: Partition }) {
    const { control, handleSubmit, watch, reset, formState: { errors, isDirty }, register, setValue } = useForm<xMountPointData>({
        defaultValues: { path: '', fstype: ''/*, flagsNames: [], customFlagsNames: []*/ }, // Default values for the form
    });
    const { fields, append, prepend, remove, swap, move, insert, replace } = useFieldArray({
        control, // control props comes from useForm (optional: if you are using FormProvider)
        name: "custom_flags_values", // unique name for your Field Array
    });
    const { data: filesystems, isLoading: fsLoading, error: fsError } = useGetFilesystemsQuery();

    // Use useEffect to update form values when objectToEdit changes or dialog opens
    useEffect(() => {
        if (props.open && props.objectToEdit) {
            const suggestedName = decodeEscapeSequence(props.objectToEdit.name || props.objectToEdit.id || 'new_mount');
            const sanitizedName = suggestedName.replace(/[\s\\/:"*?<>|]+/g, '_');
            const existingMountData = props.objectToEdit.mount_point_data?.[0];

            reset({
                path: existingMountData?.path || `/mnt/${sanitizedName}`,
                fstype: existingMountData?.fstype || undefined, // Use existing or let backend detect
                flags: existingMountData?.flags || [], // Keep numeric flags if needed internally
                custom_flags: existingMountData?.custom_flags || [], // Keep numeric flags if needed internally
                is_to_mount_at_startup: existingMountData?.is_to_mount_at_startup || false, // Initialize the switch state
            });
        } else if (!props.open) {
            reset(); // Reset to default values when closing
        }
    }, [props.open, props.objectToEdit, reset]);


    function handleCloseSubmit(formData: xMountPointData) {
        if (!props.objectToEdit) {
            console.error("Mount dialog submitted without an objectToEdit.");
            props.onClose();
            return;
        }


        const custom_flags = (formData.custom_flags || []).map(flag => {
            if (formData.custom_flags_values && formData.custom_flags_values.length > 0) {
                const flagValue = formData.custom_flags_values.find(fv => fv.name === flag.name);
                return {
                    ...flag,
                    value: flagValue ? flagValue.value : '' // Use the value from custom_flags_values if available
                };
            }
            return flag // Return the flag as is if no custom values are provided
        });
        //console.debug("Form Data:", formData,custom_flags);

        const submitData: MountPointData = {
            path: formData.path,
            path_hash: MD5(formData.path).toString(),
            fstype: formData.fstype || undefined,
            flags: formData.flags,
            custom_flags: custom_flags,
            //device: props.objectToEdit.device, // Ensure device name is included
            is_to_mount_at_startup: formData.is_to_mount_at_startup, // Include the switch value in submitted data
            type: Type.Addon,
        };
        console.log("Submitting Mount Data:", submitData);
        props.onClose(submitData);
    }

    function handleCancel() {
        props.onClose(); // Call onClose without data
    }

    const partitionNameDecoded = decodeEscapeSequence(props.objectToEdit?.name || '');
    const partitionId = props.objectToEdit?.id || 'N/A';

    return (
        <Fragment>
            <Dialog open={props.open} onClose={handleCancel} maxWidth="sm" fullWidth>
                <DialogTitle>
                    Mount Volume: {partitionNameDecoded} ({partitionId})
                </DialogTitle>
                <form id="mountvolumeform" onSubmit={handleSubmit(handleCloseSubmit)} noValidate>
                    <DialogContent>
                        <Stack spacing={2} sx={{ pt: 1 }}>
                            <DialogContentText>
                                Configure mount options for the volume. The suggested path is based on the volume name.
                            </DialogContentText>
                            <Grid container spacing={2}>
                                <Grid size={6}> {/* Corrected Grid usage */}
                                    <TextFieldElement
                                        size="small"
                                        name="path"
                                        label="Mount Path"
                                        control={control}
                                        required
                                        fullWidth
                                        slotProps={{ inputLabel: { shrink: true } }} // Ensure label is always shrunk
                                        helperText="Path must start with /mnt/"
                                    />
                                </Grid>
                                <Grid size={6}> {/* FS Type */}
                                    <AutocompleteElement
                                        name="fstype"
                                        label="File System Type"
                                        control={control}
                                        options={fsLoading ? [] : (filesystems as FilesystemType[] || []).map(fs => fs.name)}
                                        autocompleteProps={{
                                            freeSolo: true,
                                            size: "small",
                                            onChange: (event, value) => {
                                                console.log("FS Type changed:", value);
                                                setValue('custom_flags', []); // Clear custom flags when FS type changes
                                                setValue('custom_flags_values', []); // Clear custom flags values when FS type changes
                                            }
                                        }}
                                        textFieldProps={{
                                            helperText: fsError ? 'Error loading filesystems' : (fsLoading ? 'Loading...' : 'Leave blank to auto-detect'),
                                            error: !!fsError,
                                            InputLabelProps: { shrink: true }
                                        }}

                                    />
                                </Grid>
                                <Grid size={6}> {/* FS Flags */}
                                    {
                                        !fsLoading && ((filesystems as FilesystemType[])[0]?.mountFlags || []).length > 0 &&
                                        <AutocompleteElement
                                            multiple
                                            name="flags"
                                            label="Mount Flags"
                                            options={fsLoading ? [] : (filesystems as FilesystemType[])[0]?.mountFlags || []} // Use string keys for options
                                            control={control}
                                            autocompleteProps={{
                                                size: "small",
                                                limitTags: 5,
                                                getOptionKey: (option) => (option as MountFlag).name,
                                                getOptionLabel: (option) => (option as MountFlag).name,
                                                renderOption: (props, option) => (
                                                    <li {...props} >
                                                        <Tooltip title={option.description || ""}>
                                                            <span>{option.name} {option.needsValue ? <span style={{ fontSize: '0.8em', color: '#888' }}>(Requires Value)</span> : null}</span>
                                                        </Tooltip>
                                                    </li>
                                                ),
                                                isOptionEqualToValue(option, value) {
                                                    return option.name === value.name;
                                                },
                                            }}
                                            textFieldProps={{
                                                //helperText: fsError ? 'Error loading filesystems' : (fsLoading ? 'Loading...' : 'Leave blank to auto-detect'),
                                                //error: !!fsError,
                                                InputLabelProps: { shrink: true }
                                            }}
                                        />
                                    }
                                </Grid>
                                <Grid size={6}> {/* FS CustomFlags */}
                                    {
                                        !fsLoading && ((filesystems as FilesystemType[]).find(fs => fs.name === watch('fstype'))?.customMountFlags || []).length > 0 &&
                                        <AutocompleteElement
                                            multiple
                                            name="custom_flags"
                                            label="FileSystem specific Mount Flags"
                                            options={fsLoading ? [] : (filesystems as FilesystemType[]).find(fs => fs.name === watch('fstype'))?.customMountFlags/*?.filter(mf => !mf.needsValue)*/ || []} // Use string keys for options
                                            control={control}
                                            autocompleteProps={{
                                                size: "small",
                                                limitTags: 5,
                                                getOptionKey: (option) => (option as MountFlag).name,
                                                // getOptionLabel: (option) => option.name,
                                                renderOption: (props, option) => (
                                                    <li {...props} >
                                                        <Tooltip title={option.description || ""}>
                                                            <span>{option.name} {option.needsValue ? <span style={{ fontSize: '0.8em', color: '#888' }}>(Requires Value)</span> : null}</span>
                                                        </Tooltip>
                                                    </li>
                                                ),
                                                isOptionEqualToValue(option, value) {
                                                    return option.name === value.name;
                                                },
                                                renderValue: (values, getItemProps) =>
                                                    values.map((option, index) => {
                                                        const { key, ...itemProps } = getItemProps({ index });
                                                        //console.log(values, option)
                                                        return (
                                                            <Chip
                                                                color={(option as MountFlag).needsValue ? "warning" : "default"}
                                                                key={key}
                                                                variant="filled"
                                                                label={(option as MountFlag)?.name || "bobo"}
                                                                size="small"
                                                                {...itemProps}
                                                            />
                                                        );
                                                    }),
                                                onChange: (event, value) => {
                                                    console.log(event, value)
                                                    replace((value as MountFlag[]).filter(v => v.needsValue)); // Only keep flags that need values
                                                },
                                            }}
                                        />
                                    }
                                </Grid>
                                {fields.map((field, index) => (
                                    <Grid size={6}> {/* FS CustomFlags Values */}
                                        <TextFieldElement
                                            key={field.id}
                                            size="small"
                                            name={`custom_flags_values.${index}.value`}
                                            label={field.name}
                                            control={control}
                                            required
                                            fullWidth
                                            variant="outlined"
                                            rules={{
                                                required: true,
                                                pattern: {
                                                    value: RegExp(field.value_validation_regex || ".*"),
                                                    message: `Invalid value for ${field.name}. ${field.value_description}`,
                                                },
                                            }}
                                            slotProps={{ inputLabel: { shrink: true } }} // Ensure label is always shrunk
                                            helperText={field.value_description}
                                        />
                                    </Grid>
                                ))}
                                <Grid size={12}>
                                    <CheckboxElement
                                        name="is_to_mount_at_startup"
                                        label="Mount at startup"
                                        control={control}
                                        size="small"
                                    />
                                </Grid>
                            </Grid>
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={handleCancel} color="secondary">Cancel</Button>
                        <Button type="submit" variant="contained">Mount</Button> {/* Disable if form not changed */}
                    </DialogActions>
                </form>
            </Dialog>
        </Fragment>
    );
}

/*
// Helper to check if a value is a string key of the Flags enum (remains the same)
function isFlagsKey(key: string): key is keyof typeof Flags {
    // Ensure Flags is treated as an object for Object.keys
    return Object.keys(Flags as object).includes(key);
}
*/