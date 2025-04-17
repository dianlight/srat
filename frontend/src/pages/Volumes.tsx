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
import ComputerIcon from '@mui/icons-material/Computer'; // Example icon for parent disk
import { useConfirm } from "material-ui-confirm";
import { filesize } from "filesize";
import { faPlug, faPlugCircleXmark, faPlugCircleMinus } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeSvgIcon } from "../components/FontAwesomeSvgIcon";
import { AutocompleteElement, useForm, TextFieldElement } from "react-hook-form-mui"; // Import TextFieldElement
import { toast } from "react-toastify";
import { useVolume } from "../hooks/volumeHook";
import { useReadOnly } from "../hooks/readonlyHook";
import { Flags, useDeleteVolumeByMountPathMountMutation, useGetFilesystemsQuery, usePostVolumeByMountPathMountMutation, type Partition, type MountPointData } from "../store/sratApi";


export function Volumes() {
    const read_only = useReadOnly();
    const [showPreview, setShowPreview] = useState<boolean>(false);
    const [showMount, setShowMount] = useState<boolean>(false);

    const { disks, isLoading, error } = useVolume(); // disks are Partition[]
    const [selected, setSelected] = useState<Partition | undefined>(undefined);
    const confirm = useConfirm();
    const [mountVolume, mountVolumeResult] = usePostVolumeByMountPathMountMutation();
    const [umountVolume, umountVolumeResult] = useDeleteVolumeByMountPathMountMutation();
    const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({}); // State for collapse


    // Toggle collapse state for a group
    const handleToggleGroup = (groupName: string) => {
        setOpenGroups(prev => ({ ...prev, [groupName]: !prev[groupName] }));
    };

    // --- Helper functions (decodeEscapeSequence, onSubmitMountVolume, etc.) remain the same ---
    function decodeEscapeSequence(source: string) {
        return source.replace(/\\x([0-9A-Fa-f]{2})/g, function () {
            return String.fromCharCode(parseInt(arguments[1], 16));
        });
    };

    function onSubmitMountVolume(data?: MountPointData) {
        console.log("Mount", data)
        if (!data || !data.path) return
        mountVolume({ mountPath: data.path, mountPointData: data }).unwrap().then((res) => {
            toast.info(`Volume ${(res as MountPointData).path} mounted successfully.`);
            setSelected(undefined);
        }).catch(err => {
            console.error("Error:", err, err.data);
            toast.error(`${err.data?.code || 'Error'}:${err.data?.message || 'Unknown mount error'}`, { data: { error: err } });
        })
    }

    function shareExists(mountPoint: string) {
        // TODO: Implement actual share checking logic if needed
        // return shares.some(share => share.path === mountPoint);
        return true; // Placeholder
    }

    function handleCreateShare(partition: Partition) {
        // TODO: Implement navigation or action to create a share
        // navigate(`/shares/create?path=${partition.mount_point}`);
        console.log("Create share for:", partition.mount_point);
        toast.info("Create share functionality not yet implemented.");
    }

    function handleGoToShare(partition: Partition) {
        // TODO: Implement navigation or action to go to an existing share
        console.log("Go to share for:", partition.mount_point);
        toast.info("Go to share functionality not yet implemented.");
    }

    function onSubmitUmountVolume(data: Partition, force = false) {
        console.log("Umount", data)
        if (!data.mount_point_data?.path) {
            toast.error("Cannot unmount: Missing mount point path.");
            return;
        }
        confirm({
            title: `Unmount ${data.label || data.name}?`,
            description: `Do you really want to ${force ? "forcefully " : ""}unmount the Volume ${data.label || data.name}?`
        })
            .then(({ confirmed, reason }) => {
                if (confirmed) {
                    umountVolume({
                        mountPath: data.mount_point_data!.path, // Updated to use path instead of id
                        force: force,
                        lazy: true, // Consider if lazy unmount is always desired
                    }).unwrap().then((res) => {
                        setSelected(undefined);
                        toast.info(`Volume ${data.label || data.name} unmounted successfully.`);
                    }).catch(err => {
                        console.error("Unmount Error:", err);
                        const errorMsg = err.data?.message || err.status || 'Unknown error';
                        toast.error(`Error unmounting ${data.label || data.name}: ${errorMsg}`, { data: { error: err } });
                    })
                } else if (reason === "cancel") {
                    console.log("Unmount cancelled")
                }
            })
            .catch(() => {
                // Handle potential error from the confirm dialog itself if needed
                console.log("Confirmation dialog closed or errored.");
            });
    }


    // Handle loading and error states
    if (isLoading) {
        return <Typography>Loading volumes...</Typography>;
    }

    if (error) {
        // You might want a more user-friendly error display
        return <Typography color="error">Error loading volumes: {JSON.stringify(error)}</Typography>;
    }

    return <InView>
        <VolumeMountDialog objectToEdit={selected} open={showMount} onClose={(data) => { setSelected(undefined); if (data) onSubmitMountVolume(data); setShowMount(false) }} />
        <PreviewDialog title={selected?.name || ""} objectToDisplay={selected} open={showPreview} onClose={() => { setSelected(undefined); setShowPreview(false) }} />
        <br />
        <List dense={true}>
            <Divider />
            {/* Iterate over grouped disks */}
            {disks?.map((disk, idx) => (
                <Fragment key={disk.id || idx}> {/* Use disk.id or idx for unique key */}
                    {/* Header Row for the Physical Disk */}
                    <ListItemButton onClick={() => handleToggleGroup(disk.id || `${idx}`)} sx={{ pl: 0 }}>
                        <ListItemAvatar>
                            <Avatar>
                                <ComputerIcon /> {/* Icon for the parent disk */}
                            </Avatar>
                        </ListItemAvatar>
                        <ListItemText primary={`Disk: ${disk.model?.toUpperCase()}`} secondary={`${disk.partitions?.length} partition(s)`} />
                        {openGroups[disk.id || `${idx}`] ? <ExpandLess /> : <ExpandMore />}
                    </ListItemButton>

                    {/* Collapsible Section for Partitions */}
                    <Collapse in={openGroups[disk.id || `${idx}`]} timeout="auto" unmountOnExit>
                        <List component="div" disablePadding dense={true} sx={{ pl: 4 }}> {/* Indent sublist */}
                            {disk.partitions?.map((partition, idx) => (
                                <Fragment key={partition.id || `${disk.id}-${idx}`}> {/* Unique key for partition */}
                                    <ListItemButton sx={{ pl: 0 }}> {/* Adjust padding if needed */}
                                        <ListItem
                                            secondaryAction={!read_only && <>
                                                {/* Mount Button */}
                                                {!partition.mount_point_data &&
                                                    <Tooltip title="Mount Partition">
                                                        <IconButton onClick={(e) => { e.stopPropagation(); setSelected(partition); setShowMount(true) }} edge="end" aria-label="mount">
                                                            <FontAwesomeSvgIcon icon={faPlug} />
                                                        </IconButton>
                                                    </Tooltip>
                                                }
                                                {/* Unmount/Share Buttons */}
                                                {partition.mount_point_data && partition.mount_point_data[0]?.path.startsWith("/mnt/") && <>
                                                    <Tooltip title="Unmount Partition">
                                                        <IconButton onClick={(e) => { e.stopPropagation(); onSubmitUmountVolume(partition, false) }} edge="end" aria-label="unmount">
                                                            <FontAwesomeSvgIcon icon={faPlugCircleMinus} />
                                                        </IconButton>
                                                    </Tooltip>
                                                    <Tooltip title="Force Unmount Partition">
                                                        <IconButton onClick={(e) => { e.stopPropagation(); onSubmitUmountVolume(partition, true) }} edge="end" aria-label="force unmount">
                                                            <FontAwesomeSvgIcon icon={faPlugCircleXmark} />
                                                        </IconButton>
                                                    </Tooltip>
                                                    {/* Share Buttons */}
                                                    {!shareExists(partition.mount_point!) ? ( // Added non-null assertion assuming mount_point exists if mount_point_data does
                                                        <Tooltip title="Create Share">
                                                            <IconButton onClick={(e) => { e.stopPropagation(); handleCreateShare(partition) }} edge="end" aria-label="create share">
                                                                <AddIcon />
                                                            </IconButton>
                                                        </Tooltip>
                                                    ) : (
                                                        <Tooltip title="Go to Share">
                                                            <IconButton onClick={(e) => { e.stopPropagation(); handleGoToShare(partition) }} edge="end" aria-label="go to share">
                                                                <ShareIcon />
                                                            </IconButton>
                                                        </Tooltip>
                                                    )}
                                                </>}
                                            </>}
                                        >
                                            <ListItemAvatar>
                                                <Avatar>
                                                    {partition.name === 'hassos-data' ? <CreditScoreIcon /> : <StorageIcon />}
                                                </Avatar>
                                            </ListItemAvatar>
                                            <ListItemText
                                                primary={decodeEscapeSequence(partition.name || "Unknown Volume")}
                                                onClick={() => { setSelected(partition); setShowPreview(true) }}
                                                disableTypography
                                                secondary={<Stack spacing={1} direction="row" flexWrap="wrap">
                                                    {partition.size != null && <Typography variant="caption">Size: {filesize(partition.size, { round: 0 })}</Typography>}
                                                    {partition.mount_point_data && partition.mount_point_data[0]?.fstype && <Typography variant="caption">Type: {partition.mount_point_data[0]?.fstype}</Typography>}
                                                    {partition.mount_point_data && partition.mount_point_data.length > 0 && <Typography variant="caption">Mount: {partition.mount_point_data?.map((mpd) => mpd.path).join(" ")}</Typography>}
                                                    {partition.id && <Typography variant="caption">UUID: {partition.id}</Typography>}
                                                    {partition.name && <Typography variant="caption">Dev: {partition.name}</Typography>}
                                                </Stack>}
                                            />
                                        </ListItem>
                                    </ListItemButton>
                                </Fragment>
                            ))}
                        </List>
                    </Collapse>
                    <Divider /> {/* Divider between disk groups */}
                </Fragment>
            ))}
        </List>
    </InView >
}

// --- VolumeMountDialog component remains the same ---
// (Make sure VolumeMountDialog and its dependencies like xMountPointData, isFlagsKey are still present below)

interface xMountPointData extends MountPointData {
    flagsNames?: string[];
}


function VolumeMountDialog(props: { open: boolean, onClose: (data?: MountPointData) => void, objectToEdit?: Partition }) {
    const { control, handleSubmit, watch, reset, formState: { errors } } = useForm<xMountPointData>(); // Removed default values here
    const { data: filesystems, isLoading, error } = useGetFilesystemsQuery()

    // Use useEffect to update form values when objectToEdit changes or dialog opens
    useEffect(() => {
        if (props.open && props.objectToEdit) {
            reset({
                path: props.objectToEdit.mount_point || `/mnt/${props.objectToEdit.label || props.objectToEdit.name || props.objectToEdit.uuid || 'new_mount'}`, // Suggest a mount point
                id: props.objectToEdit.mount_point_data?.id || -1, // Should likely be undefined or null if new
                fstype: props.objectToEdit.type || undefined, // Use undefined if no type
                flags: props.objectToEdit.mount_point_data?.flags,
                // Convert numeric flags back to string names if your API/type expects numbers
                flagsNames: props.objectToEdit.mount_point_data?.flags?.map(flag => Flags[flag]).filter(Boolean) || [],
                // data: props.objectToEdit.mount_point_data?.data || '', // If you add the data field back
            });
        } else if (!props.open) {
            reset({}); // Clear form when closing if desired
        }
    }, [props.open, props.objectToEdit, reset]);


    function handleCloseSubmit(formData?: xMountPointData) { // Receive form data directly
        let submitData: MountPointData | undefined = undefined;
        if (formData && props.objectToEdit) { // Check if formData exists
            // Convert flag names (strings) back to numeric enum values
            const numericFlags = formData.flagsNames
                ?.map(name => Flags[name as keyof typeof Flags]) // Get numeric value from enum
                .filter(value => typeof value === 'number') as Flags[] | undefined; // Filter out non-numeric results

            submitData = {
                path: formData.path, // Use path from form
                id: formData.id, // Use id from form
                fstype: formData.fstype,
                flags: numericFlags,
                // data: formData.data, // If you add the data field back
                // You might need to add other required fields from Partition like uuid or name here
                // depending on what your mountVolume mutation expects in MountPointData
                // Example: device: props.objectToEdit.name
            };
            console.log("Submitting Mount Data:", submitData);
        } else {
            console.log("Close without submitting data");
        }
        props.onClose(submitData); // Pass processed data or undefined
    }

    function handleCancel() {
        props.onClose(); // Call onClose without data
    }


    return (
        <Fragment>
            <Dialog
                open={props.open}
                onClose={handleCancel} // Use specific cancel handler
                maxWidth="sm" // Optional: set max width
                fullWidth      // Optional: make dialog full width
            >
                <DialogTitle>
                    {/* More descriptive title */}
                    Mount Volume: {props.objectToEdit?.label || props.objectToEdit?.name} ({props.objectToEdit?.name})
                </DialogTitle>
                {/* Use form tag here to wrap content and actions */}
                <form id="mountvolumeform" onSubmit={handleSubmit(handleCloseSubmit)}>
                    <DialogContent>
                        <Stack spacing={2}>
                            <DialogContentText>
                                Configure mount options for the volume. The suggested path is based on the volume label or name.
                            </DialogContentText>
                            {/* Removed duplicate form tag */}
                            <Grid container spacing={2}>
                                <Grid item xs={12}> {/* Use item prop */}
                                    {/* Add TextField for Mount Path */}
                                    <TextFieldElement name="path" label="Mount Path" control={control} required fullWidth />
                                </Grid>
                                <Grid item xs={6}> {/* Use item prop */}
                                    <AutocompleteElement name="fstype" label="File System Type"
                                        required
                                        control={control}
                                        options={filesystems || []} // Ensure options is always an array
                                        autocompleteProps={{
                                            loading: isLoading, // Show loading indicator
                                            disabled: !!props.objectToEdit?.type // Disable if type is pre-filled?
                                        }}
                                        textFieldProps={{
                                            helperText: error ? 'Error loading filesystems' : (isLoading ? 'Loading...' : ''),
                                            error: !!error
                                        }}
                                    />
                                </Grid>
                                <Grid item xs={6}> {/* Use item prop */}
                                    <AutocompleteElement
                                        multiple
                                        name="flagsNames"
                                        label="Mount Flags"
                                        // Provide string options based on the Flags enum keys
                                        options={Object.keys(Flags).filter((v) => isNaN(Number(v)))}
                                        control={control}
                                        autocompleteProps={{
                                            disableCloseOnSelect: true // Keep dropdown open for multi-select
                                        }}
                                    />
                                </Grid>
                                {/*
                                    <Grid item xs={12}> // Use item prop
                                        <TextFieldElement name="data" label="Additional Mount Data (e.g., uid=1000)" control={control} fullWidth />
                                    </Grid>
                                    */}
                            </Grid>
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        {/* Use the cancel handler */}
                        <Button onClick={handleCancel} color="secondary">Cancel</Button>
                        {/* Submit button triggers the form's onSubmit */}
                        <Button type="submit" variant="contained">Mount</Button>
                    </DialogActions>
                </form> {/* Close form tag */}
            </Dialog>
        </Fragment>
    );
}

// Helper to check if a value is a string key of the Flags enum
function isFlagsKey(key: string): key is keyof typeof Flags {
    return Object.keys(Flags).includes(key);
}

