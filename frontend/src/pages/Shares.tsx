import { Fragment, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { InView } from "react-intersection-observer";
import Grid from "@mui/material/Grid";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import IconButton from "@mui/material/IconButton";
import ListItemAvatar from "@mui/material/ListItemAvatar";
import Avatar from "@mui/material/Avatar";
import ListItemText from "@mui/material/ListItemText";
import FolderSharedIcon from '@mui/icons-material/FolderShared';
import SettingsIcon from '@mui/icons-material/Settings';
import FolderSpecialIcon from '@mui/icons-material/FolderSpecial';
import { PreviewDialog } from "../components/PreviewDialog";
import { useConfirm } from "material-ui-confirm";
import Divider from "@mui/material/Divider";
import ListItemButton from "@mui/material/ListItemButton";
import Button from "@mui/material/Button";
import Dialog from "@mui/material/Dialog";
import DialogTitle from "@mui/material/DialogTitle";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogActions from "@mui/material/DialogActions";
import { AutocompleteElement, CheckboxElement, SelectElement, TextFieldElement } from 'react-hook-form-mui'
import { Box, Fab, Stack, Tooltip } from "@mui/material";
import ModeEditIcon from '@mui/icons-material/ModeEdit';
import AddIcon from '@mui/icons-material/Add';
import { Eject, DriveFileMove } from '@mui/icons-material';
import { Chip, Typography } from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import EditIcon from '@mui/icons-material/Edit';
import BlockIcon from '@mui/icons-material/Block';
import BackupIcon from '@mui/icons-material/Backup';
import DeleteIcon from '@mui/icons-material/Delete';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { useAppDispatch, useAppSelector } from "../store/store";
import { Usage, useGetUsersQuery, usePutShareByShareNameMutation, type SharedResource, type User } from "../store/sratApi";
import { useShare } from "../hooks/shareHook";
import { useReadOnly } from "../hooks/readonlyHook";
import { addMessage } from "../store/errorSlice";
import { useVolume } from "../hooks/volumeHook";

interface ShareEditProps extends SharedResource {
    org_name: string,
}


export function Shares() {
    const read_only = useReadOnly();
    const dispatch = useAppDispatch();
    const errors = useAppSelector((state) => state.errors.messages);
    const { shares, isLoading, error } = useShare();
    const [selected, setSelected] = useState<[string, SharedResource] | null>(null);
    const [showPreview, setShowPreview] = useState<boolean>(false);
    const [showEdit, setShowEdit] = useState<boolean>(false);
    const [showUserEdit, setShowUserEdit] = useState<boolean>(false);
    //const formRef = useRef<HTMLFormElement>(null);
    const confirm = useConfirm();
    const [updateShare, updateShareResult] = usePutShareByShareNameMutation();

    function onSubmitDisableShare(cdata?: string, props?: SharedResource) {
        console.log("Disable", cdata, props);
        if (!cdata || !props) return
        confirm({
            title: `Disable ${props?.name}?`,
            description: "If you disable this share, all of its configurations will be retained."
        })
            .then(({ confirmed, reason }) => {
                if (confirmed) {
                    updateShare({ shareName: props?.name || "", sharedResource: { ...props, disabled: true } }).unwrap()
                        .then(() => {
                            //                        setErrorInfo('');
                        })
                        .catch(err => {
                            dispatch(addMessage(JSON.stringify(err)));
                        });
                } else if (reason === "cancel") {
                    console.log("cancel")
                }
            })
    }

    function onSubmitEnableShare(cdata?: string, props?: SharedResource) {
        console.log("Enable", cdata, props);
        if (!cdata || !props) return
        updateShare({ shareName: props?.name || "", sharedResource: { ...props, disabled: false } }).unwrap()
            .then(() => {
                //            setErrorInfo('');
            })
            .catch(err => {
                dispatch(addMessage(JSON.stringify(err)));
            });
    }

    function onSubmitEditShare(data?: ShareEditProps) {
        if (!data || !selected) return;
        if (!data.name || !data.mount_point_data?.path) {
            dispatch(addMessage("Unable to open share!"));
            return;
        }

        // Save Data
        console.log(data);
        if (data.org_name !== "") {

            updateShare({ shareName: data.org_name, sharedResource: { ...data, disabled: false } }).unwrap()
                .then(() => {
                    //            setErrorInfo('');
                })
                .catch(err => {
                    dispatch(addMessage(JSON.stringify(err)));
                });
        }
        setShowEdit(false);
        return false;
    }

    function onSubmitUnmount(share: string): void {
        confirm({
            title: `Unmount ${share}?`,
            description: "Are you sure you want to unmount this share?"
        })
            .then(() => {
                /*
                api.share.unmountShare(share).then((res) => {
                    console.log(`Share ${share} unmounted successfully`);
                    // Optionally update the state or perform other actions
                }).catch(err => {
                    console.error(`Failed to unmount share ${share}`, err);
                    setErrorInfo(`Failed to unmount share ${share}: ${err.message}`);
                });
                */
            })
            .catch(() => {
                // Handle cancel action if needed
            });
    }

    function onSubmitMount(share: string): void {
        confirm({
            title: `Mount ${share}?`,
            description: "Are you sure you want to mount this share?"
        })
            .then(({ confirmed, reason }) => {
                if (confirmed) {
                    // TODO: 
                } else if (reason === "cancel") {
                    console.log("cancel")
                }
            })
    }

    return <InView>
        <PreviewDialog title={selected?.[1].name || ""} objectToDisplay={selected?.[1]} open={showPreview} onClose={() => { setSelected(null); setShowPreview(false) }} />
        <ShareEditDialog objectToEdit={{ ...selected?.[1], org_name: selected?.[1].name || "" }} open={showEdit} onClose={(data) => { onSubmitEditShare(data); setSelected(null); setShowEdit(false) }} />
        {read_only || <Fab color="primary" aria-label="add" sx={{
            float: 'right',
            top: '-20px',
            margin: '-8px'

        }} size="small"
            onClick={() => { setSelected(null); setShowEdit(true) }}
        >
            <AddIcon />
        </Fab>}
        <br />
        <List dense={true}>
            {shares ? Object.entries(shares).sort((a, b) => a[1].name!.localeCompare(b[1].name || "")).map(([share, props]) =>
                <Fragment key={share}>
                    <ListItemButton sx={{
                        opacity: props.disabled ? 0.5 : 1,
                        '&:hover': {
                            opacity: 1,
                        },
                    }}>
                        <ListItem
                            secondaryAction={!read_only && <>
                                <IconButton onClick={() => { setSelected([share, props]); setShowEdit(true) }} edge="end" aria-label="settings">
                                    <SettingsIcon />
                                </IconButton>
                                {props.mount_point_data?.invalid &&
                                    <IconButton onClick={() => { }} edge="end" aria-label="delete">
                                        <Tooltip title="Delete share">
                                            <DeleteIcon color="error" />
                                        </Tooltip>
                                    </IconButton>
                                }
                                {/* 
                                <IconButton onClick={() => { setSelected([share, props]); setShowUserEdit(true) }} edge="end" aria-label="users">
                                    <Tooltip title="Manage Users">
                                        <GroupIcon />
                                    </Tooltip>
                                </IconButton>
                                */}
                                {(props.usage !== Usage.Internal) && (props.mount_point_data?.is_mounted ? (
                                    <IconButton onClick={() => onSubmitUnmount(share)} edge="end" aria-label="unmount">
                                        <Tooltip title="Unmount">
                                            <Eject />
                                        </Tooltip>
                                    </IconButton>
                                ) : (
                                    <IconButton onClick={() => onSubmitMount(share)} edge="end" aria-label="mount">
                                        <Tooltip title="Mount">
                                            <DriveFileMove />
                                        </Tooltip>
                                    </IconButton>
                                ))}
                                {props.disabled ? (
                                    <Tooltip title="Enable share">
                                        <span>
                                            <IconButton
                                                onClick={() => onSubmitEnableShare(share, props)}
                                                edge="end"
                                                aria-label="disable"
                                            >
                                                <CheckCircleIcon />
                                            </IconButton>
                                        </span>
                                    </Tooltip>
                                ) : (
                                    <Tooltip title="Disable share">
                                        <span>
                                            <IconButton
                                                onClick={() => onSubmitDisableShare(share, props)}
                                                edge="end"
                                                aria-label="disable"
                                            >
                                                <BlockIcon />
                                            </IconButton>
                                        </span>
                                    </Tooltip>
                                )}

                            </>
                            }
                        >
                            <ListItemAvatar>
                                <Avatar>
                                    {props.mount_point_data?.invalid && <Tooltip title={props.mount_point_data?.invalid_error} arrow>
                                        <FolderSharedIcon color="error" />
                                    </Tooltip> || <Tooltip title={props.mount_point_data?.warnings} arrow>
                                            <FolderSharedIcon />
                                        </Tooltip>
                                    }
                                </Avatar>
                            </ListItemAvatar>
                            <ListItemText
                                primary={
                                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                        {props.name}
                                        {props.mount_point_data?.is_mounted && (
                                            <Chip
                                                size="small"
                                                color="success"
                                                label="Mounted"
                                                icon={<CheckCircleIcon />}
                                            />
                                        )}
                                    </Box>
                                }
                                onClick={() => { setSelected([share, props]); setShowPreview(true) }}
                                secondary={
                                    <Typography variant="body2" component="div">
                                        {props.mount_point_data?.path && (
                                            <Box component="span" sx={{ display: 'block' }}>
                                                Mount Point: {props.mount_point_data.path}
                                            </Box>
                                        )}
                                        {props.mount_point_data?.warnings && props.usage !== Usage.Internal && (
                                            <Box component="span" sx={{ display: 'block', color: 'orange' }}>
                                                Warning: {props.mount_point_data.warnings}
                                            </Box>
                                        )}
                                        <Box component="div" sx={{ mt: 1, display: 'flex', flexDirection: 'row', flexWrap: 'wrap', gap: 1 }}>
                                            {props.users && props.users.length > 0 && (
                                                <Tooltip title="Users with write access">
                                                    <Chip
                                                        onClick={(e) => { e.stopPropagation(); setSelected([share, props]); setShowEdit(true) }}
                                                        size="small"
                                                        icon={<EditIcon />}
                                                        label={
                                                            <span>
                                                                Users: {props.users.map(u => (
                                                                    <span key={u.username} style={{ color: u.is_admin ? 'yellow' : 'inherit' }}>
                                                                        {u.username}
                                                                        {u !== props.users![props.users!.length - 1] && ', '}
                                                                    </span>
                                                                ))}
                                                            </span>
                                                        }
                                                        sx={{ my: 0.5 }}
                                                    />
                                                </Tooltip>
                                            )}
                                            {props.ro_users && props.ro_users.length > 0 && (
                                                <Tooltip title="Users with read-only access">
                                                    <Chip
                                                        onClick={(e) => { e.stopPropagation(); setSelected([share, props]); setShowEdit(true) }}
                                                        size="small"
                                                        icon={<VisibilityIcon />}
                                                        label={
                                                            <span>
                                                                Read-only Users: {props.ro_users.map(u => (
                                                                    <span key={u.username} style={{ color: u.is_admin ? 'yellow' : 'inherit' }}>
                                                                        {u.username}
                                                                        {u !== props.ro_users![props.ro_users!.length - 1] && ', '}
                                                                    </span>
                                                                ))}
                                                            </span>
                                                        }
                                                        sx={{ my: 0.5 }}
                                                    />
                                                </Tooltip>
                                            )}
                                            {(props.usage && props.usage !== Usage.Internal) && (
                                                <Tooltip title="Share Usage">
                                                    <Chip
                                                        onClick={(e) => { e.stopPropagation(); setSelected([share, props]); setShowEdit(true) }}
                                                        size="small"
                                                        icon={<FolderSpecialIcon />}
                                                        label={`Usage: ${props.usage}`}
                                                        sx={{ my: 0.5 }}
                                                    />
                                                </Tooltip>
                                            )}
                                            {props.timemachine && (
                                                <Tooltip title="TimeMachine Enabled">
                                                    <Chip
                                                        onClick={(e) => { e.stopPropagation(); setSelected([share, props]); setShowEdit(true) }}
                                                        size="small"
                                                        icon={<BackupIcon />}
                                                        label="TimeMachine"
                                                        color="secondary"
                                                        sx={{ my: 0.5 }}
                                                    />
                                                </Tooltip>
                                            )}
                                        </Box>
                                    </Typography>
                                }
                            />
                        </ListItem>
                    </ListItemButton>
                    <Divider component="li" />
                </Fragment>
            ) : null}
        </List>
    </InView>
}


interface ShareEditPropsEdit extends ShareEditProps {
    usersNames?: string[],
    roUsersNames?: string[],
}

function ShareEditDialog(props: { open: boolean, onClose: (data?: ShareEditProps) => void, objectToEdit?: ShareEditProps }) {
    const { data: users, isLoading } = useGetUsersQuery()
    const volumes = useVolume()
    const shares = useShare()
    const [editName, setEditName] = useState(false);
    const { control, handleSubmit, watch, formState: { errors } } = useForm<ShareEditPropsEdit>(
        {
            values: !props.objectToEdit ? {
                org_name: "",
                name: "",
                users: [],
                ro_users: [],
                timemachine: false,
                usersNames: [],
                roUsersNames: [],
            } : {
                ...props.objectToEdit,
                usersNames: props.objectToEdit?.users?.map(user => user.username as string) || [],
                roUsersNames: props.objectToEdit?.ro_users?.map(user => user.username as string) || [],
                /*
                volumeId: props.objectToEdit.mount_point_data?.path ?
                    volumes.disks?.partitions?.
                        filter(mount => mount.mount_point_data?.path?.startsWith("/mnt/")).
                        find(mount => mount.mount_point_data?.id === props.objectToEdit?.mount_point_data?.path)?.mount_point_data?.id
                    : undefined
                    */
            }
        },
    );
    const selected_users = watch("usersNames")
    const selected_ro_users = watch("roUsersNames")


    function handleCloseSubmit(data?: ShareEditPropsEdit) {
        setEditName(false)
        if (!data) {
            props.onClose()
            return
        }
        data.mount_point_data = volumes.disks?.flatMap(disk => disk.partitions)?.flatMap(partition => partition?.mount_point_data).find(mount_point_data => mount_point_data?.path === data?.mount_point_data?.path);
        data.users = data.usersNames?.map(username => (users as User[])?.find(userobj => userobj.username === username)).filter(v3 => v3 !== undefined)
        data.ro_users = data.usersNames?.map(username => (users as User[])?.find(userobj => userobj.username === username)).filter(v3 => v3 !== undefined)
        //console.log(data)
        props.onClose(data)
    }

    return (
        <Fragment>
            <Dialog
                open={props.open}
                onClose={() => handleCloseSubmit()}
            >
                <DialogTitle>
                    {!(editName || props.objectToEdit?.org_name === "") && <>
                        <IconButton onClick={() => setEditName(true)}>
                            <ModeEditIcon fontSize="small" />
                        </IconButton>
                        {props.objectToEdit?.name}
                    </>
                    }
                    {(editName || props.objectToEdit?.org_name === "") && <TextFieldElement sx={{ display: "flex" }} name="name" label="Share Name" required size="small" control={control} />
                    }
                </DialogTitle>
                <DialogContent>
                    <Stack spacing={2}>
                        <DialogContentText>
                            Please enter or modify the Samba share data.
                        </DialogContentText>
                        <form id="editshareform" onSubmit={handleSubmit(handleCloseSubmit)} noValidate>
                            <Grid container spacing={2}>
                                {/*
                                <Grid size={6}>
                                    <TextFieldElement name="name" label="Share Name" required control={control} />
                                </Grid>
                                */}
                                {props.objectToEdit?.usage !== Usage.Internal &&
                                    <Grid size={6}>
                                        <SelectElement
                                            sx={{ display: "flex" }}
                                            label="Usage"
                                            name="usage"
                                            options={Object.keys(Usage)
                                                .filter(usage => usage.toLowerCase() !== Usage.Internal)
                                                .map(usage => { return { id: usage.toLowerCase(), label: usage } })}
                                            required control={control} />
                                    </Grid>
                                }
                                {
                                    props.objectToEdit?.usage !== Usage.Internal && <>
                                        <Grid size={6}>
                                            <SelectElement sx={{ display: "flex" }}
                                                label="Volume"
                                                name="mount_point_data"
                                                options={volumes.disks?.flatMap(disk => disk.partitions)?.flatMap(partition => partition?.mount_point_data).
                                                    filter(mount => mount?.path?.startsWith("/mnt/")).
                                                    filter(mount => (shares.shares.map(share => share.mount_point_data?.path).indexOf(mount?.path) == -1 || mount?.path === props.objectToEdit?.mount_point_data?.path)).
                                                    map(mount => {
                                                        return {
                                                            id: mount?.path,
                                                            label: mount?.path + "(" + mount?.device + ")",
                                                            //  disabled: mount.label === "Invalid Volume"
                                                        }
                                                    })}
                                                required
                                                control={control} />
                                        </Grid>
                                        <Grid size={6}>
                                            <CheckboxElement label="Timemachine" name="timemachine" control={control} />
                                        </Grid></>
                                }
                                <Grid size={12}>
                                    <AutocompleteElement
                                        name="usersNames"
                                        label="Read and Write users"
                                        loading={isLoading}
                                        options={
                                            ((users as User[])?.
                                                filter(user => selected_ro_users?.indexOf(user.username || "") == -1).
                                                map(user => ({ id: user.username, label: user.username })) || [])
                                        }
                                        control={control}
                                        matchId
                                        multiple
                                    />
                                    <AutocompleteElement
                                        name="roUsersNames"
                                        label="ReadOnly users"
                                        loading={isLoading}
                                        options={
                                            ((users as User[])?.
                                                filter(user => selected_users?.indexOf(user.username || "") == -1).
                                                map(user => ({ id: user.username, label: user.username })) || [])
                                        }
                                        control={control}
                                        matchId
                                        multiple
                                    />
                                </Grid>
                            </Grid>
                        </form>
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => handleCloseSubmit()}>Cancel</Button>
                    <Button type="submit" form="editshareform">{(props.objectToEdit?.org_name === "") ? "Create" : "Apply"}</Button>
                </DialogActions>
            </Dialog>
        </Fragment>
    );
}
