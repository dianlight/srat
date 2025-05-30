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
import { Usage, useDeleteShareByShareNameMutation, useGetUsersQuery, usePostShareMutation, usePutShareByShareNameMutation, type MountPointData, type SharedResource, type User } from "../store/sratApi";
import { useShare } from "../hooks/shareHook";
import { useReadOnly } from "../hooks/readonlyHook";
import { addMessage } from "../store/errorSlice";
import { useVolume } from "../hooks/volumeHook";
import { toast } from "react-toastify";

interface ShareEditProps extends SharedResource {
    org_name: string,
    trashbin?: boolean, // TODO: Implement TrashBin support
    //usersNames?: string[],
    //roUsersNames?: string[],
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
    const [deleteShare, updateDeleteShareResult] = useDeleteShareByShareNameMutation();
    const [createShare, createShareResult] = usePostShareMutation();

    function onSubmitDisableShare(cdata?: string, props?: SharedResource) {
        console.log("Disable", cdata, props);
        if (!cdata || !props) return
        confirm({
            title: `Disable ${props?.name}?`,
            description: "If you disable this share, all of its configurations will be retained.",
            acknowledgement: "I understand that disabling the share will retain its configurations but prevent access to it.",
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

    function onSubmitDeleteShare(cdata?: string, props?: SharedResource) {
        console.log("Delete", cdata, props);
        if (!cdata || !props) return
        confirm({
            title: `Delete ${props?.name}?`,
            description: "This action cannot be undone. Are you sure you want to delete this share?",
            acknowledgement: "I understand that deleting the share will remove it permanently.",
        })
            .then(({ confirmed, reason }) => {
                if (confirmed) {
                    deleteShare({ shareName: props?.name || "" }).unwrap()
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

    function onSubmitEditShare(data?: ShareEditProps) {
        console.log("Edit Share", data, selected);
        if (!data) return;
        if (!data.name || !data.mount_point_data?.path) {
            dispatch(addMessage("Unable to open share!"));
            return;
        }

        // Save Data
        console.log(data);
        if (data.org_name !== "") {
            updateShare({ shareName: data.org_name, sharedResource: data }).unwrap()
                .then((res) => {
                    toast.info(`Share ${(res as SharedResource).name} modified successfully.`);
                    setSelected(null);
                    setShowEdit(false);
                })
                .catch(err => {
                    dispatch(addMessage(JSON.stringify(err)));
                });
        } else {
            createShare({ sharedResource: data }).unwrap()
                .then((res) => {
                    toast.info(`Share ${(res as SharedResource).name || data.name} created successfully.`);
                    setSelected(null);
                    setShowEdit(false);
                })
                .catch(err => {
                    dispatch(addMessage(JSON.stringify(err)));
                });
        }

        return false;
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
                                {props.usage !== Usage.Internal &&
                                    <IconButton onClick={() => onSubmitDeleteShare(share, props)} edge="end" aria-label="delete">
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
                                        {/*props.mount_point_data?.is_mounted && (
                                            <Chip
                                                size="small"
                                                color="success"
                                                label="Mounted"
                                                icon={<CheckCircleIcon />}
                                            />
                                        )*/}
                                    </Box>
                                }
                                onClick={() => { setSelected([share, props]); setShowPreview(true) }}
                                secondary={
                                    <Typography variant="body2" component="div">
                                        {props.mount_point_data?.path && (
                                            <Box component="span" sx={{ display: 'block' }}>
                                                Path: {props.mount_point_data.path}
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
                                                        variant="outlined"
                                                        label={
                                                            <Typography variant="body2" component="span">
                                                                Users: {props.users.map(u => (
                                                                    <Typography variant="body2" component="span" key={u.username} color={u.is_admin ? 'warning' : 'inherit'}>
                                                                        {u.username}
                                                                        {u !== props.users![props.users!.length - 1] && ', '}
                                                                    </Typography>
                                                                ))}
                                                            </Typography>
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
                                                        variant="outlined"
                                                        label={
                                                            <span>
                                                                Read-only Users: {props.ro_users.map(u => (
                                                                    <span key={u.username} style={{ color: u.is_admin ? 'warning' : 'inherit' }}>
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
                                                <Tooltip title={`Share Usage: ${props.is_ha_mounted ? 'HA Mounted' : 'Not HA Mounted'}`}>
                                                    <Chip
                                                        onClick={(e) => { e.stopPropagation(); setSelected([share, props]); setShowEdit(true) }}
                                                        size="small"
                                                        variant="outlined"
                                                        icon={<FolderSpecialIcon />}
                                                        disabled={!props.is_ha_mounted}
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
                                                        variant="outlined"
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

function ShareEditDialog(props: { open: boolean, onClose: (data?: ShareEditProps) => void, objectToEdit?: ShareEditProps }) {
    const { data: users, isLoading: usLoading, error: usError } = useGetUsersQuery()
    const { disks: volumes, isLoading: vlLoading, error: vlError } = useVolume()
    const shares = useShare()
    const [editName, setEditName] = useState(false);
    const { control, handleSubmit, watch, formState: { errors } } = useForm<ShareEditProps>(
        {
            values: !props.objectToEdit ? {
                org_name: "",
                name: "",
                users: [],
                ro_users: [],
                timemachine: false,
                usage: Usage.None,
            } : {
                ...props.objectToEdit,
                //usersNames: props.objectToEdit?.users?.map(user => user.username as string) || [],
                //roUsersNames: props.objectToEdit?.ro_users?.map(user => user.username as string) || [],
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


    function handleCloseSubmit(data?: ShareEditProps) {
        setEditName(false)
        if (!data) {
            props.onClose()
            return
        }
        console.log(data)
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
                            Please enter or modify share properties.
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
                                            size="small"
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
                                            <AutocompleteElement
                                                label="Volume"
                                                name="mount_point_data"
                                                options={volumes?.flatMap(disk => disk.partitions)?.filter(Boolean).filter(partition => !partition?.system).flatMap(partition => partition?.mount_point_data).filter(mp => mp?.path !== "") as MountPointData[] || [] as MountPointData[]}
                                                control={control}
                                                required
                                                loading={vlLoading}
                                                autocompleteProps={{
                                                    size: "small",
                                                    renderValue: (value) => {
                                                        return value.path || "--";
                                                    },
                                                    getOptionLabel: (option: MountPointData) => option?.path || "",
                                                    getOptionKey: (option) => option?.path_hash || "",
                                                    renderOption: (props, option) => (
                                                        <li key={option.path_hash} {...props} >
                                                            <Typography variant="body2">{option.path}</Typography>
                                                        </li>
                                                    ),
                                                    isOptionEqualToValue(option, value) {
                                                        //console.log("Comparing", option, value);
                                                        if (!value || !option) return false;
                                                        return option.path_hash === value?.path_hash;
                                                    },
                                                }}
                                            />
                                        </Grid>
                                        <Grid size={6}>
                                            <CheckboxElement label="Support Timemachine Backups" name="timemachine" control={control} />
                                        </Grid>
                                        <Grid size={6}>
                                            <CheckboxElement disabled label="Support TrashBin" name="trashbin" control={control} />
                                        </Grid>
                                    </>
                                }
                                <Grid size={6}>
                                    {
                                        !usLoading && ((users as User[]) || []).length > 0 &&
                                        <AutocompleteElement
                                            multiple
                                            name="users"
                                            label="Read and Write users"
                                            options={usLoading ? [] : (users as User[]) || []} // Use string keys for options
                                            control={control}
                                            loading={usLoading}
                                            autocompleteProps={{
                                                size: "small",
                                                limitTags: 5,
                                                getOptionKey: (option) => option.username || "",
                                                getOptionLabel: (option) => option.username || "",
                                                renderOption: (props, option) => (
                                                    <li key={option.username} {...props} >
                                                        <Typography variant="body2" color={option.is_admin ? "warning" : "default"}>{option.username}</Typography>
                                                    </li>
                                                ),
                                                getOptionDisabled: (option) => {
                                                    if (watch("ro_users")?.find(user => user.username === option.username)) {
                                                        return true; // Disable if the user is already in the users list
                                                    }
                                                    return false;
                                                },
                                                isOptionEqualToValue(option, value) {
                                                    return option.username === value.username;
                                                },
                                                renderValue: (values, getItemProps) =>
                                                    values.map((option, index) => {
                                                        const { key, ...itemProps } = getItemProps({ index });
                                                        //console.log(values, option)
                                                        return (
                                                            <Chip
                                                                color={option.is_admin ? "warning" : "default"}
                                                                key={key}
                                                                variant="outlined"
                                                                label={option?.username || "bobo"}
                                                                size="small"
                                                                {...itemProps}
                                                            />
                                                        );
                                                    }),
                                            }}
                                            textFieldProps={{
                                                //helperText: fsError ? 'Error loading filesystems' : (fsLoading ? 'Loading...' : 'Leave blank to auto-detect'),
                                                //error: !!fsError,
                                                InputLabelProps: { shrink: true }
                                            }}
                                        />
                                    }
                                </Grid>
                                <Grid size={6}>
                                    {
                                        !usLoading && ((users as User[]) || []).length > 0 &&
                                        <AutocompleteElement
                                            multiple
                                            name="ro_users"
                                            label="Read Only users"
                                            options={usLoading ? [] : (users as User[]) || []} // Use string keys for options
                                            control={control}
                                            loading={usLoading}
                                            autocompleteProps={{
                                                size: "small",
                                                limitTags: 5,
                                                getOptionKey: (option) => option.username || "",
                                                getOptionLabel: (option) => option.username || "",
                                                renderOption: (props, option) => (
                                                    <li key={option.username} {...props}>
                                                        <Typography variant="body2" color={option.is_admin ? "warning" : "default"}>{option.username}</Typography>
                                                    </li>
                                                ),
                                                getOptionDisabled: (option) => {
                                                    if (watch("users")?.find(user => user.username === option.username)) {
                                                        return true; // Disable if the user is already in the users list
                                                    }
                                                    return false;
                                                },
                                                isOptionEqualToValue(option, value) {
                                                    return option.username === value.username;
                                                },
                                                renderValue: (values, getItemProps) =>
                                                    values.map((option, index) => {
                                                        const { key, ...itemProps } = getItemProps({ index });
                                                        //console.log(values, option)
                                                        return (
                                                            <Chip
                                                                color={option.is_admin ? "warning" : "default"}
                                                                key={key}
                                                                variant="outlined"
                                                                label={option?.username || "bobo"}
                                                                size="small"
                                                                {...itemProps}
                                                            />
                                                        );
                                                    }),
                                            }}
                                            textFieldProps={{
                                                //helperText: fsError ? 'Error loading filesystems' : (fsLoading ? 'Loading...' : 'Leave blank to auto-detect'),
                                                //error: !!fsError,
                                                InputLabelProps: { shrink: true }
                                            }}
                                        />
                                    }
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
