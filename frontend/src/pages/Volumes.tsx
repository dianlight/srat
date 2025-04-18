import { Fragment, useState, useEffect } from "react";
import { InView } from "react-intersection-observer";
import { PreviewDialog } from "../components/PreviewDialog";
import List from "@mui/material/List";
import { ListItemButton, ListItem, IconButton, ListItemAvatar, Avatar, ListItemText, Divider, Stack, Typography, Tooltip, Dialog, Button, DialogActions, DialogContent, DialogContentText, DialogTitle, Grid, Collapse } from "@mui/material"; // Import Collapse
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
import { AutocompleteElement, useForm, TextFieldElement } from "react-hook-form-mui"; // Import TextFieldElement
import { toast } from "react-toastify";
import { useVolume } from "../hooks/volumeHook";
import { useReadOnly } from "../hooks/readonlyHook";
import { Flags, useDeleteVolumeByMountPathMountMutation, useGetFilesystemsQuery, usePostVolumeByMountPathMountMutation, type Partition, type Disk, type MountPointData } from "../store/sratApi";
// Add EjectIcon to your imports
import EjectIcon from '@mui/icons-material/Eject';
import UsbIcon from '@mui/icons-material/Usb';
import SdStorageIcon from '@mui/icons-material/SdStorage';
// ... other icon imports ...
import ComputerIcon from '@mui/icons-material/Computer';
import SettingsSuggestIcon from '@mui/icons-material/SettingsSuggest';


export function Volumes() {
    const read_only = useReadOnly();
    const [showPreview, setShowPreview] = useState<boolean>(false);
    const [showMount, setShowMount] = useState<boolean>(false);

    // Assuming useVolume returns an array of disk objects, where each disk has a 'partitions' array
    // If the structure is different (e.g., a flat list of partitions with parent info), the grouping logic needs adjustment
    const { disks, isLoading, error } = useVolume();
    const [selected, setSelected] = useState<Partition | Disk | undefined>(undefined); // Can hold a disk or partition
    const confirm = useConfirm();
    const [mountVolume, mountVolumeResult] = usePostVolumeByMountPathMountMutation();
    const [umountVolume, umountVolumeResult] = useDeleteVolumeByMountPathMountMutation();
    const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({}); // State for collapse, key is disk identifier


    // Toggle collapse state for a group (disk)
    const handleToggleGroup = (diskIdentifier: string) => {
        setOpenGroups(prev => ({ ...prev, [diskIdentifier]: !prev[diskIdentifier] }));
    };

    // --- Helper functions (decodeEscapeSequence, onSubmitMountVolume, etc.) remain the same ---
    function decodeEscapeSequence(source: string) {
        return source.replace(/\\x([0-9A-Fa-f]{2})/g, function () {
            // Ensure arguments[1] is treated as a string before parseInt
            return String.fromCharCode(parseInt(String(arguments[1]), 16));
        });
    };

    function onSubmitMountVolume(data?: MountPointData) {
        console.log("Mount", data)
        // Ensure selected is a partition and data/path exist
        if (!selected || (selected as Disk).partitions || !data || !data.path) {
            toast.error("Cannot mount: Invalid selection or missing data.");
            return;
        }
        // Assuming your Partition type includes 'name' which is the device path like /dev/sda1
        const submitData: MountPointData = {
            ...data,
        };
        mountVolume({ mountPath: data.path, mountPointData: submitData }).unwrap().then((res) => {
            toast.info(`Volume ${(res as MountPointData).path} mounted successfully.`);
            setSelected(undefined); // Clear selection after successful mount
            setShowMount(false); // Close the mount dialog
        }).catch(err => {
            console.error("Mount Error:", err, err.data);
            const errorMsg = err.data?.message || err.status || 'Unknown mount error';
            const errorCode = err.data?.code || 'Error';
            toast.error(`${errorCode}: ${errorMsg}`, { data: { error: err } });
        })
    }

    function shareExists(mountPoint: string | undefined): boolean {
        if (!mountPoint) return false;
        // TODO: Implement actual share checking logic if needed
        // Example: return shares.some(share => share.path === mountPoint);
        console.warn("Share existence check not implemented, assuming true for path:", mountPoint);
        return true; // Placeholder - Adjust as needed
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
        const mountPath = partition.mount_point_data?.[0]?.path;
        if (!mountPath) {
            toast.error("Cannot unmount: Missing mount point path.");
            console.error("Missing mount path for partition:", partition);
            return;
        }

        // Use partition label or name for confirmation dialog
        const displayName = partition.name || 'this volume';

        confirm({
            title: `Unmount ${displayName}?`,
            description: `Do you really want to ${force ? "forcefully " : ""}unmount the Volume ${displayName} (${partition.name}) mounted at ${mountPath}?`,
            confirmationText: force ? "Force Unmount" : "Unmount",
            cancellationText: "Cancel",
            confirmationButtonProps: { color: force ? "error" : "primary" }
        })
            .then(() => { // Only proceed if confirmed (no need to check 'confirmed' boolean)
                console.log(`Proceeding with ${force ? 'forced ' : ''}unmount for:`, mountPath);
                umountVolume({
                    mountPath: mountPath, // Use the extracted path
                    force: force,
                    // lazy: true, // Consider if lazy unmount is always desired, maybe make it conditional?
                }).unwrap().then(() => {
                    toast.info(`Volume ${displayName} unmounted successfully.`);
                    // Optionally clear selection if the unmounted item was selected
                    if (selected?.id === partition.id) {
                        setSelected(undefined);
                    }
                }).catch(err => {
                    console.error("Unmount Error:", err);
                    const errorMsg = err.data?.message || err.status || 'Unknown error';
                    toast.error(`Error unmounting ${displayName}: ${errorMsg}`, { data: { error: err } });
                })
            })
            .catch(() => {
                // This catch block handles cancellation of the confirm dialog
                console.log("Unmount cancelled by user.");
            });
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

    return <InView>
        {/* Pass selected (could be disk or partition) to mount dialog, but mounting only makes sense for partitions */}
        <VolumeMountDialog
            objectToEdit={selected && !(selected as Disk).partitions ? selected : undefined} // Only pass if selected is likely a partition
            open={showMount}
            onClose={(data) => {
                // Keep selected state until mount attempt finishes or is cancelled
                if (data) {
                    onSubmitMountVolume(data); // onSubmitMountVolume handles setSelected(undefined) on success
                } else {
                    // If cancelled, clear selection and hide dialog
                    setSelected(undefined);
                    setShowMount(false);
                }
            }} />
        {/* PreviewDialog can show details for both disks and partitions */}
        <PreviewDialog
            title={(selected as Disk)?.model || (selected as Partition)?.name || "Details"} // Use model, label, or name for title
            objectToDisplay={selected}
            open={showPreview}
            onClose={() => {
                setSelected(undefined);
                setShowPreview(false);
            }} />
        <br />
        <List dense={true}>
            <Divider />
            {/* Iterate over disks */}
            {validDisks.map((disk, diskIdx) => {
                // Use a stable identifier for the disk (id preferred, fallback to index)
                const diskIdentifier = disk.id || `disk-${diskIdx}`;
                const isGroupOpen = !!openGroups[diskIdentifier]; // Check if this disk's group is open

                return (
                    <Fragment key={diskIdentifier}> {/* Use stable identifier as key */}
                        {/* Header Row for the Physical Disk */}
                        <ListItemButton
                            onClick={() => { // <-- Click row to show preview
                                setSelected(disk);
                                setShowPreview(true);
                            }}
                            sx={{ pl: 0 }} // Adjust padding as needed
                        >
                            <ListItemAvatar>
                                <Avatar>
                                    {/* --- UPDATED DYNAMIC ICON LOGIC --- */}
                                    {(() => {
                                        // Prioritize specific known removable bus types
                                        switch (disk.connection_bus?.toLowerCase()) {
                                            case 'usb':
                                                return <UsbIcon />;
                                            case 'sdio':
                                            case 'mmc': // Often used for SD cards
                                                return <SdStorageIcon />;
                                            // Add other specific bus cases if needed
                                        }

                                        // If not a specific removable bus, check the removable flag
                                        if (disk.removable) {
                                            // You could use EjectIcon or maybe UsbIcon as a generic removable indicator
                                            return <EjectIcon />;
                                        }

                                        // Default for non-removable or unknown bus types
                                        // You could add specific icons for 'sata', 'nvme' etc. here if desired
                                        return <ComputerIcon />;
                                    })()}
                                    {/* --- END UPDATED DYNAMIC ICON LOGIC --- */}
                                </Avatar>
                            </ListItemAvatar>
                            <ListItemText
                                primary={`Disk: ${disk.model?.toUpperCase() || `Disk ${diskIdx + 1}`}`} // Display model or name
                                secondary={`${disk.partitions?.length || 0} partition(s)`} // Handle case where partitions might be missing
                            />
                            {/* Conditionally render IconButton only if there are partitions */}
                            {disk.partitions && disk.partitions.length > 0 && (
                                <IconButton
                                    edge="end"
                                    size="small"
                                    onClick={(e) => {
                                        e.stopPropagation(); // Prevent ListItemButton onClick (preview)
                                        handleToggleGroup(diskIdentifier);
                                    }}
                                    aria-label={isGroupOpen ? 'Collapse partitions' : 'Expand partitions'}
                                    aria-expanded={isGroupOpen} // Accessibility improvement
                                >
                                    {isGroupOpen ? <ExpandLess /> : <ExpandMore />}
                                </IconButton>
                            )}
                        </ListItemButton>

                        {/* Collapsible Section for Partitions */}
                        {/* Ensure disk.partitions exists before trying to map */}
                        {disk.partitions && disk.partitions.length > 0 && (
                            <Collapse in={isGroupOpen} timeout="auto" unmountOnExit>
                                <List component="div" disablePadding dense={true} sx={{ pl: 4 }}> {/* Indent sublist */}
                                    {disk.partitions.map((partition, partIdx) => {
                                        // Use a stable identifier for the partition
                                        const partitionIdentifier = partition.id || `${diskIdentifier}-part-${partIdx}`;
                                        const isMounted = partition.mount_point_data && partition.mount_point_data.length > 0;
                                        const firstMountPath = partition.mount_point_data?.[0]?.path;
                                        // Determine if share actions should be shown (mounted under /mnt/ typically)
                                        const showShareActions = isMounted && firstMountPath?.startsWith("/mnt/");

                                        return (
                                            <Fragment key={partitionIdentifier}> {/* Unique key for partition */}
                                                {/* Wrap ListItem in ListItemButton for consistent click behavior */}
                                                <ListItemButton
                                                    sx={{ pl: 1 }} // Indent partition button slightly less than list padding
                                                    onClick={() => { // Click partition row to show its preview
                                                        setSelected(partition);
                                                        setShowPreview(true);
                                                    }}
                                                >
                                                    <ListItem
                                                        // Prevent ListItem's default padding from interfering
                                                        disablePadding
                                                        secondaryAction={!read_only && !partition.system && (
                                                            <Stack direction="row" spacing={0.5} alignItems="center">
                                                                {/* Mount Button */}
                                                                {!isMounted && (
                                                                    <Tooltip title="Mount Partition">
                                                                        {/* Prevent preview dialog when clicking mount */}
                                                                        <IconButton onClick={(e) => { e.stopPropagation(); setSelected(partition); setShowMount(true); }} edge="end" aria-label="mount" size="small">
                                                                            <FontAwesomeSvgIcon icon={faPlug} />
                                                                        </IconButton>
                                                                    </Tooltip>
                                                                )}
                                                                {/* Unmount/Share Buttons */}
                                                                {isMounted && (
                                                                    <>
                                                                        <Tooltip title="Unmount Partition">
                                                                            {/* Prevent preview dialog when clicking unmount */}
                                                                            <IconButton onClick={(e) => { e.stopPropagation(); onSubmitUmountVolume(partition, false); }} edge="end" aria-label="unmount" size="small">
                                                                                <FontAwesomeSvgIcon icon={faPlugCircleMinus} />
                                                                            </IconButton>
                                                                        </Tooltip>
                                                                        <Tooltip title="Force Unmount Partition">
                                                                            {/* Prevent preview dialog when clicking force unmount */}
                                                                            <IconButton onClick={(e) => { e.stopPropagation(); onSubmitUmountVolume(partition, true); }} edge="end" aria-label="force unmount" size="small">
                                                                                <FontAwesomeSvgIcon icon={faPlugCircleXmark} />
                                                                            </IconButton>
                                                                        </Tooltip>
                                                                        {/* Share Buttons */}
                                                                        {showShareActions && !shareExists(firstMountPath) ? (
                                                                            <Tooltip title="Create Share">
                                                                                {/* Prevent preview dialog when clicking create share */}
                                                                                <IconButton onClick={(e) => { e.stopPropagation(); handleCreateShare(partition); }} edge="end" aria-label="create share" size="small">
                                                                                    <AddIcon />
                                                                                </IconButton>
                                                                            </Tooltip>
                                                                        ) : showShareActions && (
                                                                            <Tooltip title="Go to Share">
                                                                                {/* Prevent preview dialog when clicking go to share */}
                                                                                <IconButton onClick={(e) => { e.stopPropagation(); handleGoToShare(partition); }} edge="end" aria-label="go to share" size="small">
                                                                                    <ShareIcon />
                                                                                </IconButton>
                                                                            </Tooltip>
                                                                        )}
                                                                    </>
                                                                )}
                                                            </Stack>
                                                        )}
                                                    >
                                                        <ListItemAvatar sx={{ minWidth: 'auto', pr: 1.5 }}> {/* Adjust padding */}
                                                            <Avatar sx={{ width: 32, height: 32 }}> {/* Slightly smaller avatar */}
                                                                {/* Updated Icon Logic */}
                                                                {partition.name === 'hassos-data'
                                                                    ? <CreditScoreIcon fontSize="small" />
                                                                    : partition.system
                                                                        ? <SettingsSuggestIcon fontSize="small" />
                                                                        : <StorageIcon fontSize="small" />
                                                                }
                                                            </Avatar>
                                                        </ListItemAvatar>
                                                        <ListItemText
                                                            primary={decodeEscapeSequence(partition.name || "Unknown Partition")} // Prefer label, fallback to name
                                                            // Remove onClick here, handled by parent ListItemButton
                                                            disableTypography
                                                            secondary={<Stack spacing={1} direction="row" flexWrap="wrap" alignItems="center" sx={{ pt: 0.5 }}> {/* Add slight padding top */}
                                                                {partition.size != null && <Typography variant="caption" noWrap>Size: {filesize(partition.size, { round: 0 })}</Typography>}
                                                                {/* Display first fstype if available */}
                                                                {partition.mount_point_data?.[0]?.fstype && <Typography variant="caption" noWrap>Type: {partition.mount_point_data[0].fstype}</Typography>}
                                                                {/* Display all mount paths */}
                                                                {isMounted && <Typography variant="caption" noWrap>Mount: {partition.mount_point_data?.map((mpd) => mpd.path).join(" ")}</Typography>}
                                                                {partition.id && <Typography variant="caption" noWrap>UUID: {partition.id}</Typography>}
                                                                {partition.device && <Typography variant="caption" noWrap>Dev: {partition.device}</Typography>}
                                                            </Stack>}
                                                        />
                                                    </ListItem>
                                                </ListItemButton>
                                                {/* Render divider only if not the last partition in this group */}
                                                {partIdx < (disk.partitions?.length || 0) - 1 && (
                                                    <Divider variant="inset" component="li" sx={{ ml: 4 }} /> // Ensure divider is indented
                                                )}
                                            </Fragment>
                                        );
                                    })}
                                </List>
                            </Collapse>
                        )}
                        <Divider /> {/* Divider between disk groups */}
                    </Fragment>
                );
            })}
        </List>
    </InView >
}

// --- VolumeMountDialog component remains the same ---
// Interface and component definition...

interface xMountPointData extends MountPointData {
    flagsNames?: string[]; // Array of flag names (strings) for the Autocomplete
}


function VolumeMountDialog(props: { open: boolean, onClose: (data?: MountPointData) => void, objectToEdit?: Partition }) {
    const { control, handleSubmit, watch, reset, formState: { errors } } = useForm<xMountPointData>({
        // Set default values within useForm for better control
        defaultValues: {
            path: '',
            fstype: '',
            flagsNames: [],
            // data: '' // If you re-add the data field
        }
    });
    const { data: filesystems, isLoading, error: fsError } = useGetFilesystemsQuery();

    // Use useEffect to update form values when objectToEdit changes or dialog opens
    useEffect(() => {
        if (props.open && props.objectToEdit) {
            // Suggest a mount path: /mnt/label or /mnt/name or /mnt/uuid or /mnt/new_mount
            const suggestedName = props.objectToEdit.name || props.objectToEdit.id || 'new_mount';
            // Basic sanitization: replace spaces and potentially problematic characters
            const sanitizedName = suggestedName.replace(/[\s\\/:"*?<>|]+/g, '_');

            // Get data from the first mount point if it exists (for editing scenarios)
            const existingMountData = props.objectToEdit.mount_point_data?.[0];

            reset({
                // Use the actual mount point if it exists, otherwise generate suggestion
                path: existingMountData?.path || `/mnt/${sanitizedName}`,
                // Use existing fstype, fallback to partition type, then undefined
                fstype: existingMountData?.fstype || undefined,
                // Use existing flags if available
                flags: existingMountData?.flags,
                // Convert numeric flags back to string names for the Autocomplete
                // Ensure Flags enum is correctly used for reverse mapping
                flagsNames: existingMountData?.flags
                    ?.map(flagValue => flagValue.toString()) // Get string name from numeric value
                    .filter((flagName): flagName is string => typeof flagName === 'string') // Ensure it's a string
                    || [], // Default to empty array if no flags
                // data: existingMountData?.data || '', // Use data from first mount point
            });
        } else if (!props.open) {
            reset(); // Reset to default values when closing
        }
    }, [props.open, props.objectToEdit, reset]);


    function handleCloseSubmit(formData: xMountPointData) { // Receive validated form data directly
        let submitData: MountPointData | undefined = undefined;
        if (props.objectToEdit) { // Check if objectToEdit exists (it should if the dialog was opened for mounting)
            // Convert flag names (strings) back to numeric enum values
            const numericFlags = formData.flagsNames
                ?.map(name => Flags[name as keyof typeof Flags]) // Get numeric value from enum
                .filter((value): value is Flags => typeof value === 'number') // Filter out non-numeric results and ensure type
                || undefined; // Use undefined if no flags selected

            submitData = {
                path: formData.path, // Use path from form
                // id: formData.id, // Usually backend-assigned, maybe omit unless editing an existing mount config
                fstype: formData.fstype || undefined, // Send undefined if empty to let backend auto-detect
                flags: numericFlags,
                // data: formData.data, // If you add the data field back
                // Add the device identifier (partition name/path) required by the API
                device: props.objectToEdit.name // Assuming 'name' is the device identifier like /dev/sda1
            };
            console.log("Submitting Mount Data:", submitData);
            props.onClose(submitData); // Pass processed data
        } else {
            console.error("Mount dialog submitted without an objectToEdit.");
            props.onClose(); // Close without data if something went wrong
        }
    }

    function handleCancel() {
        props.onClose(); // Call onClose without data
    }


    return (
        <Fragment>
            <Dialog
                open={props.open}
                onClose={handleCancel} // Use specific cancel handler
                maxWidth="sm"
                fullWidth
            >
                <DialogTitle>
                    {/* More descriptive title */}
                    Mount Volume: {props.objectToEdit?.name} ({props.objectToEdit?.id})
                </DialogTitle>
                {/* Use form tag here to wrap content and actions */}
                {/* Add noValidate to prevent browser default validation interfering with react-hook-form */}
                {/* Pass handleSubmit directly to the form's onSubmit */}
                <form id="mountvolumeform" onSubmit={handleSubmit(handleCloseSubmit)} noValidate>
                    <DialogContent>
                        <Stack spacing={2} sx={{ pt: 1 }}> {/* Add some padding top */}
                            <DialogContentText>
                                Configure mount options for the volume. The suggested path is based on the volume label or name.
                            </DialogContentText>
                            <Grid container spacing={2}>
                                <Grid size={12}> {/* Use item prop with xs */}
                                    <TextFieldElement
                                        name="path"
                                        label="Mount Path"
                                        control={control}
                                        required
                                        fullWidth
                                        InputLabelProps={{ shrink: true }} // Keep label floated when value exists
                                    />
                                </Grid>
                                <Grid size={6}> {/* Use item prop */}
                                    <AutocompleteElement name="fstype" label="File System Type"
                                        control={control}
                                        options={filesystems as [] || []} // Ensure options is always an array
                                        loading={isLoading}
                                        autocompleteProps={{
                                            freeSolo: true, // Allow typing custom fstype
                                            value: watch('fstype') || null, // Control the value explicitly for Autocomplete
                                            // Handle potential object values if API returns them
                                            getOptionLabel: (option) => typeof option === 'string' ? option : '',
                                            isOptionEqualToValue: (option, value) => option === value,
                                        }}
                                        textFieldProps={{
                                            helperText: fsError ? 'Error loading filesystems' : (isLoading ? 'Loading...' : 'Leave blank to auto-detect (if supported)'),
                                            error: !!fsError,
                                            InputLabelProps: { shrink: true } // Keep label floated
                                        }}
                                    />
                                </Grid>
                                <Grid size={6}> {/* Use item prop */}
                                    <AutocompleteElement
                                        multiple
                                        name="flagsNames"
                                        label="Mount Flags"
                                        // Provide string options based on the Flags enum keys (filtering out numeric keys)
                                        options={Object.keys(Flags).filter((v) => isNaN(Number(v)))}
                                        control={control}
                                        autocompleteProps={{
                                            disableCloseOnSelect: true, // Keep dropdown open for multi-select
                                            value: watch('flagsNames') || [], // Control the value explicitly
                                            getOptionLabel: (option) => typeof option === 'string' ? option : '',
                                            isOptionEqualToValue: (option, value) => option === value,
                                        }}
                                        textFieldProps={{
                                            InputLabelProps: { shrink: true } // Keep label floated
                                        }}
                                    />
                                </Grid>
                                {/*
                                    <Grid item xs={12}>
                                        <TextFieldElement name="data" label="Options (e.g., uid=1000,gid=1000)" control={control} fullWidth
                                            helperText="Comma-separated key=value pairs"
                                            InputLabelProps={{ shrink: true }}
                                        />
                                    </Grid>
                                    */}
                            </Grid>
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={handleCancel} color="secondary">Cancel</Button>
                        {/* Submit button triggers the form's onSubmit */}
                        <Button type="submit" variant="contained">Mount</Button>
                    </DialogActions>
                </form> {/* Close form tag */}
            </Dialog>
        </Fragment>
    );
}

// Helper to check if a value is a string key of the Flags enum (remains the same)
function isFlagsKey(key: string): key is keyof typeof Flags {
    return Object.keys(Flags as object).includes(key);
}
