import { Fragment, useContext, useEffect, useRef, useState } from "react";
import { apiContext as api, ModeContext } from "../Contexts";
import { type DtoUser } from "../srat";
import { useForm, type SubmitHandler } from "react-hook-form";
import useSWR from "swr";
import { Fab, List, ListItemButton, ListItem, IconButton, ListItemAvatar, Avatar, ListItemText, Divider, Dialog, DialogTitle, Stack, DialogContent, DialogContentText, Grid2 as Grid, DialogActions, Button } from "@mui/material";
import { InView } from "react-intersection-observer";
import { useConfirm } from "material-ui-confirm";
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import ManageAccountsIcon from '@mui/icons-material/ManageAccounts';
import PersonRemoveIcon from '@mui/icons-material/PersonRemove';
import AssignmentIndIcon from '@mui/icons-material/AssignmentInd';
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings';
import ModeEditIcon from '@mui/icons-material/ModeEdit';
import { PasswordElement, PasswordRepeatElement, TextFieldElement } from "react-hook-form-mui";
import { stringWidth } from "bun";



interface UsersProps extends DtoUser {
    doCreate?: boolean
    "password-repeat"?: string
}

export function Users() {
    const mode = useContext(ModeContext);
    const users = useSWR<UsersProps[]>('/users', () => api.users.usersList().then(res => res.data));
    const admin = useSWR<UsersProps>('/admin/user', () => api.useradmin.useradminList().then(res => res.data));
    const [errorInfo, setErrorInfo] = useState<string>('')
    const [selected, setSelected] = useState<UsersProps>({});
    const confirm = useConfirm();
    const [showEdit, setShowEdit] = useState<boolean>(false);

    function onSubmitEditUser(data?: UsersProps) {
        if (!data || !data.username || !data.password) {
            setErrorInfo('Unable to update user!');
            return;
        }

        data.username = data.username.trim();
        data.password = data.password.trim();

        // Save Data
        console.log(data);
        if (data.doCreate) {
            api.user.userCreate(data).then((res) => {
                setErrorInfo('')
                setSelected({});
                users.mutate();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
            })
            return;
        } else if (data.is_admin) {
            api.useradmin.useradminUpdate(data).then((res) => {
                setErrorInfo('')
                admin.mutate();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
            })
        } else {
            api.user.userUpdate(data.username, data).then((res) => {
                setErrorInfo('')
                setSelected({});
                users.mutate();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
            })
        }
        // formRef.current?.reset();
    }

    function onSubmitDeleteUser(data: UsersProps) {
        console.log("Delete", data)
        if (!data) return

        confirm({
            title: `Delete ${data.username}?`,
            description: "Do you really would delete this user?"
        })
            .then(() => {
                if (!data.username) {
                    setErrorInfo('Unable to delete user!');
                    return;
                }
                api.user.userDelete(data.username).then((res) => {
                    setErrorInfo('')
                    setSelected({});
                    users.mutate();
                }).catch(err => {
                    setErrorInfo(JSON.stringify(err));
                })
            })
            .catch(() => {
                /* ... */
            });
    }

    return (
        <InView>
            <UserEditDialog objectToEdit={selected} open={showEdit} onClose={(data) => { setSelected({}); onSubmitEditUser(data); setShowEdit(false) }} />
            {mode.read_only || <Fab key="fab_users" color="primary" aria-label="add" sx={{
                float: 'right',
                top: '-20px',
                margin: '-8px'
            }} size="small"
                onClick={() => { setSelected({ doCreate: true }); setShowEdit(true) }}
            >
                <PersonAddIcon />
            </Fab>}
            <List dense={true}>
                {users.data?.sort((a, b) => {
                    if (a.is_admin) {
                        return -1;
                    } else if (b.is_admin) {
                        return 1;
                    } else if (a.username === b.username) {
                        return 0;
                    } else {
                        return (a.username || "") > (b.username || "") ? -1 : 1;
                    }
                }).map((user) =>
                    <Fragment key={user.username || "admin"}>
                        <ListItemButton>
                            <ListItem
                                secondaryAction={!mode.read_only && <>
                                    <IconButton onClick={() => { setSelected(user); setShowEdit(true) }} edge="end" aria-label="settings">
                                        <ManageAccountsIcon />
                                    </IconButton>
                                    {user.is_admin ||
                                        <IconButton onClick={() => onSubmitDeleteUser(user)} edge="end" aria-label="delete">
                                            <PersonRemoveIcon />
                                        </IconButton>
                                    }
                                </>
                                }
                            >
                                <ListItemAvatar>
                                    <Avatar>
                                        {user.is_admin ?
                                            <AdminPanelSettingsIcon /> :
                                            <AssignmentIndIcon />}

                                    </Avatar>
                                </ListItemAvatar>
                                <ListItemText
                                    primary={user.username}
                                    secondary={JSON.stringify(user)}
                                />
                            </ListItem>
                        </ListItemButton>
                        <Divider component="li" />
                    </Fragment>
                )}
            </List>
        </InView>
    )
}

function UserEditDialog(props: { open: boolean, onClose: (data?: UsersProps) => void, objectToEdit?: UsersProps }) {
    const { control, handleSubmit, watch, formState: { errors } } = useForm<UsersProps>(
        {
            defaultValues: {
                username: '',
                password: '',
                is_admin: false
            },
            values: props.objectToEdit?.doCreate ? {
                username: "",
                password: "",
                is_admin: false
            } : props.objectToEdit
        },
    );

    function handleCloseSubmit(data?: UsersProps) {
        props.onClose(data)
    }

    return (
        <Fragment>
            <Dialog
                open={props.open}
                onClose={() => handleCloseSubmit()}
            >
                <DialogTitle>
                    {props.objectToEdit?.is_admin ? "Administrator" : (props.objectToEdit?.username || "New User")}
                </DialogTitle>
                <DialogContent>
                    <Stack spacing={2}>
                        <DialogContentText>
                            To subscribe to this website, please enter your email address here. We
                            will send updates occasionally.
                        </DialogContentText>
                        <form id="editshareform" onSubmit={handleSubmit(handleCloseSubmit)} noValidate>
                            <Grid container spacing={2}>
                                <Grid size={6}>
                                    <TextFieldElement name="username" label="User Name" required control={control} disabled={props.objectToEdit?.username ? (props.objectToEdit.is_admin ? false : true) : false} />
                                </Grid>
                                <Grid size={6}>
                                    <PasswordElement name="password" label="Password"
                                        required control={control} />
                                    <PasswordRepeatElement passwordFieldName={'password'} name={'password-repeat'} margin={'dense'} label={'Repeat Password'} required control={control} />
                                </Grid>
                            </Grid>
                        </form>
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => handleCloseSubmit()}>Cancel</Button>
                    <Button type="submit" form="editshareform">{(props.objectToEdit?.doCreate) ? "Create" : "Apply"}</Button>
                </DialogActions>
            </Dialog>
        </Fragment>
    );
}
