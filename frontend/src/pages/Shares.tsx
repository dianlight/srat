import { Fragment, useContext, useEffect, useRef, useState } from "react";
import { apiContext as api, ModeContext } from "../Contexts";
import { DtoEventType, type Api, type DtoSharedResource, type DtoUser } from "../srat";
import { set, useForm } from "react-hook-form";
import useSWR from "swr";
import { InView } from "react-intersection-observer";
import Grid from "@mui/material/Grid2";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import IconButton from "@mui/material/IconButton";
import ListItemAvatar from "@mui/material/ListItemAvatar";
import Avatar from "@mui/material/Avatar";
import ListItemText from "@mui/material/ListItemText";
import FolderSharedIcon from '@mui/icons-material/FolderShared';
import FolderDeleteIcon from '@mui/icons-material/FolderDelete';
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
import TextField from "@mui/material/TextField";
import DialogActions from "@mui/material/DialogActions";
import { AutocompleteElement, CheckboxElement, FormContainer, SelectElement, TextFieldElement } from 'react-hook-form-mui'
import { Box, Container, Fab, Paper, Stack, Tooltip } from "@mui/material";
import ModeEditIcon from '@mui/icons-material/ModeEdit';
import AddIcon from '@mui/icons-material/Add';
import { Eject, DriveFileMove } from '@mui/icons-material';
import { Chip, Typography } from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import GroupIcon from '@mui/icons-material/Group';
import EditIcon from '@mui/icons-material/Edit';
import BlockIcon from '@mui/icons-material/Block';
import BackupIcon from '@mui/icons-material/Backup';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { useSSE } from "react-hooks-sse";

interface ShareEditProps extends DtoSharedResource {
    org_name: string,
}


export function Shares() {
    const mode = useContext(ModeContext);
    const statusSSE = useSSE(DtoEventType.EventShare, {} as DtoSharedResource, {
        parser(input: any): DtoSharedResource {
            console.log("Got shares", input)
            const c = JSON.parse(input);
            setStatus(c)
            return c;
        },
    });

    const [status, setStatus] = useState<DtoSharedResource[]>([]);
    const [selected, setSelected] = useState<[string, DtoSharedResource] | null>(null);
    const [showPreview, setShowPreview] = useState<boolean>(false);
    const [showEdit, setShowEdit] = useState<boolean>(false);
    const [showUserEdit, setShowUserEdit] = useState<boolean>(false);
    const [errorInfo, setErrorInfo] = useState<string>('')
    const formRef = useRef<HTMLFormElement>(null);
    const confirm = useConfirm();


    useEffect(() => {
        api.shares.sharesList().then((res) => {
            console.log("Got shares", res.data)
            setStatus(res.data);
        }).catch(err => {
            console.error(err);
            //setErrorInfo(JSON.stringify(err));
        })
    }, [])

    function onSubmitDisableShare(data?: string) {
        console.log("Delete", data)
        if (!data) return
        confirm({
            title: `Delete ${data}?`,
            description: "If you delete this share, all of their configurations will be deleted."
        })
            .then(() => {
                api.share.shareDelete(data).then((res) => {
                    setSelected(null);
                    //users.mutate();
                }).catch(err => {
                    console.error(err);
                    //setErrorInfo(JSON.stringify(err));
                })
            })
            .catch(() => {
                /* ... */
            });
    }

    function onSubmitEditShare(data?: ShareEditProps) {
        if (!data) return;
        if (!data.name || !data.mount_point_data?.path) {
            setErrorInfo('Unable to update share!');
            return;
        }

        // Save Data
        console.log(data);
        if (data.org_name === "") {
            api.share.shareCreate(data).then((res) => {
                setErrorInfo('')
                setSelected(null);
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
            })
            return;
        } else {
            api.share.shareUpdate(data.org_name, data).then((res) => {
                setErrorInfo('')
                //setSelectedUser(null);
                //users.mutate();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
            })
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

    return <InView>
        <PreviewDialog title={selected ? selected[0] : ""} objectToDisplay={selected?.[1]} open={showPreview} onClose={() => { setSelected(null); setShowPreview(false) }} />
        <ShareEditDialog objectToEdit={{ ...selected?.[1], org_name: selected?.[0] || "" }} open={showEdit} onClose={(data) => { setSelected(null); onSubmitEditShare(data); setShowEdit(false) }} />
        {mode.read_only || <Fab color="primary" aria-label="add" sx={{
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
            {Object.entries(status).map(([share, props]) =>
                <Fragment key={share}>
                    <ListItemButton>
                        <ListItem
                            secondaryAction={!mode.read_only && <>
                                <IconButton onClick={() => { setSelected([share, props]); setShowEdit(true) }} edge="end" aria-label="settings">
                                    <SettingsIcon />
                                </IconButton>
                                <IconButton onClick={() => { setSelected([share, props]); setShowUserEdit(true) }} edge="end" aria-label="users">
                                    <Tooltip title="Manage Users">
                                        <GroupIcon />
                                    </Tooltip>
                                </IconButton>
                                {props.mount_point_data?.is_mounted ? (
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
                                )}
                                <Tooltip title={props.mount_point_data?.is_mounted ? "Cannot disable mounted share" : "Disable share"}>
                                    <span>
                                        <IconButton
                                            onClick={() => onSubmitDisableShare(share)}
                                            edge="end"
                                            aria-label="disable"
                                            disabled={props.mount_point_data?.is_mounted}
                                        >
                                            <BlockIcon />
                                        </IconButton>
                                    </span>
                                </Tooltip>
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
                                        {props.mount_point_data?.warnings && (
                                            <Box component="span" sx={{ display: 'block', color: 'orange' }}>
                                                Warning: {props.mount_point_data.warnings}
                                            </Box>
                                        )}
                                        <Box component="div" sx={{ mt: 1, display: 'flex', flexDirection: 'row', flexWrap: 'wrap', gap: 1 }}>
                                            {props.users && props.users.length > 0 && (
                                                <Tooltip title="Users with write access">
                                                    <Chip
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
                                            {props.usage && (
                                                <Tooltip title="Share Usage">
                                                    <Chip
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
            )}
        </List>
    </InView>
}

function ShareEditDialog(props: { open: boolean, onClose: (data?: ShareEditProps) => void, objectToEdit?: ShareEditProps }) {
    const admin = useSWR<DtoUser>('/admin/user', () => api.useradmin.useradminList().then(res => res.data));
    const users = useSWR<DtoUser[]>('/users', () => api.users.usersList().then(res => res.data));
    const [editName, setEditName] = useState(false);
    const { control, handleSubmit, watch, formState: { errors } } = useForm<ShareEditProps>(
        {
            values: props.objectToEdit?.org_name === "" ? {
                org_name: "",
                name: "",
                //: "",
                users: [],
                ro_users: [],
                timemachine: false,
                //usage: ""
            } : props.objectToEdit
        },
    );

    function handleCloseSubmit(data?: ShareEditProps) {
        setEditName(false)
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
                            To subscribe to this website, please enter your email address here. We
                            will send updates occasionally.
                        </DialogContentText>
                        <form id="editshareform" onSubmit={handleSubmit(handleCloseSubmit)} noValidate>
                            <Grid container spacing={2}>
                                {/*
                                <Grid size={6}>
                                    <TextFieldElement name="name" label="Share Name" required control={control} />
                                </Grid>
                                */}
                                <Grid size={6}>
                                    <SelectElement sx={{ display: "flex" }} label="Usage" name="usage"
                                        options={[
                                            {
                                                id: 'native', label: 'Native'
                                            },
                                            {
                                                id: 'media', label: 'Media'
                                            },
                                            {
                                                id: 'share', label: 'Share'
                                            },
                                            {
                                                id: 'backup', label: 'Backup'
                                            }
                                        ]} required control={control} />
                                </Grid>
                                <Grid size={6}>
                                    <TextFieldElement sx={{ display: "flex" }} name="mount_point_data.path" label="Mount Path" required control={control} />
                                </Grid>
                                <Grid size={6}>
                                    <CheckboxElement label="Timemachine" name="timemachine" control={control} />
                                </Grid>
                                <Grid size={6}>
                                    <AutocompleteElement
                                        name="users"
                                        label="Read and Write users"
                                        options={
                                            (users.data?.map(user => ({ id: user.username, label: user.username })) || []).concat({ id: admin.data?.username, label: admin.data?.username })
                                        }
                                        control={control}
                                        multiple
                                    />
                                </Grid>
                                <Grid size={6} offset={6}>
                                    <AutocompleteElement
                                        name="ro_users"
                                        label="ReadOnly users"
                                        options={
                                            (users.data?.map(user => ({ id: user.username, label: user.username })) || []).concat({ id: admin.data?.username, label: admin.data?.username })
                                        }
                                        control={control}
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
