import { Fragment, useContext, useEffect, useRef, useState } from "react";
import { apiContext as api, ModeContext, wsContext as ws } from "../Contexts";
import { MainEventType, type Api, type ConfigShare, type ConfigShares, type ConfigUser } from "../srat";
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
import { Box, Container, Fab, Paper, Stack } from "@mui/material";
import ModeEditIcon from '@mui/icons-material/ModeEdit';
import AddIcon from '@mui/icons-material/Add';

interface ShareEditProps extends ConfigShare {
    org_name: string,
}


export function Shares() {
    const mode = useContext(ModeContext);
    const [status, setStatus] = useState<ConfigShares>({});
    const [selected, setSelected] = useState<[string, ConfigShare] | null>(null);
    const [showPreview, setShowPreview] = useState<boolean>(false);
    const [showEdit, setShowEdit] = useState<boolean>(false);
    const [errorInfo, setErrorInfo] = useState<string>('')
    const formRef = useRef<HTMLFormElement>(null);
    const confirm = useConfirm();


    useEffect(() => {
        const chr = ws.subscribe<ConfigShares>(MainEventType.EventShare, (data) => {
            console.log("Got shares", data)
            setStatus(data);
        })
        return () => {
            ws.unsubscribe(chr);
        };
    }, [])

    function onSubmitDeleteShare(data?: string) {
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
        if (!data.name || !data.path) {
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
                                <IconButton onClick={() => onSubmitDeleteShare(share)} edge="end" aria-label="delete">
                                    <FolderDeleteIcon />
                                </IconButton>
                            </>
                            }
                        >
                            <ListItemAvatar>
                                <Avatar>
                                    <FolderSharedIcon />
                                </Avatar>
                            </ListItemAvatar>
                            <ListItemText
                                primary={share}
                                onClick={() => { setSelected([share, props]); setShowPreview(true) }}
                                secondary={props.path}
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
    const admin = useSWR<ConfigUser>('/admin/user', () => api.admin.userList().then(res => res.data));
    const users = useSWR<ConfigUser[]>('/users', () => api.users.usersList().then(res => res.data));
    const [editName, setEditName] = useState(false);
    const { control, handleSubmit, watch, formState: { errors } } = useForm<ShareEditProps>(
        {
            values: props.objectToEdit?.org_name === "" ? {
                org_name: "",
                name: "",
                path: "",
                users: [],
                ro_users: [],
                timemachine: false,
                usage: ""
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
                                    <TextFieldElement sx={{ display: "flex" }} name="path" label="Mount Path" required control={control} />
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
