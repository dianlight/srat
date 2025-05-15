import { Fragment, useState } from "react";
import { useForm } from "react-hook-form";
import { Fab, List, ListItemButton, ListItem, IconButton, ListItemAvatar, Avatar, ListItemText, Divider, Dialog, DialogTitle, Stack, DialogContent, DialogContentText, Grid as Grid, DialogActions, Button } from "@mui/material";
import { InView } from "react-intersection-observer";
import { useConfirm } from "material-ui-confirm";
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import ManageAccountsIcon from '@mui/icons-material/ManageAccounts';
import PersonRemoveIcon from '@mui/icons-material/PersonRemove';
import AssignmentIndIcon from '@mui/icons-material/AssignmentInd';
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings';
import { PasswordElement, PasswordRepeatElement, TextFieldElement } from "react-hook-form-mui";
import { useDeleteUserByUsernameMutation, useGetUsersQuery, usePostUserMutation, usePutUseradminMutation, usePutUserByUsernameMutation, type User } from "../store/sratApi";
import { useReadOnly } from "../hooks/readonlyHook";
import { toast } from "react-toastify";



interface UsersProps extends User {
    doCreate?: boolean
    "password-repeat"?: string
}

export function Users() {
    const read_only = useReadOnly();
    const users = useGetUsersQuery();
    //const admin = useGetUseradminQuery();
    const [errorInfo, setErrorInfo] = useState<string>('')
    const [selected, setSelected] = useState<UsersProps>({ username: "", password: "" });
    const confirm = useConfirm();
    const [showEdit, setShowEdit] = useState<boolean>(false);
    const [userCreate] = usePostUserMutation();
    const [userAdminUpdate] = usePutUseradminMutation();
    const [userUpdate] = usePutUserByUsernameMutation();
    const [userDelete] = useDeleteUserByUsernameMutation();

    function onSubmitEditUser(data?: UsersProps) {
        if (!data || !data.username || !data.password) {
            console.log("Data is invalid", data)
            setErrorInfo('Unable to update user!');
            return;
        }

        data.username = data.username.trim();
        data.password = data.password.trim();

        // Save Data
        console.log(data);
        if (data.doCreate) {
            userCreate({ user: data }).unwrap().then((res) => {
                setErrorInfo('')
                setSelected({ username: "", password: "" });
                users.refetch();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
                console.error(err);
                toast.error(`Error userCreate ${data.username}`, { data: { error: err.data } });
            })
            return;
        } else if (data.is_admin) {
            userAdminUpdate({ user: data }).unwrap().then((res) => {
                setErrorInfo('')
                users.refetch();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
                toast.error(`Error userAdminUpdate ${data.username}`, { data: { error: err.data } });
                console.error(err);
            })
        } else {
            userUpdate({ username: data.username, user: data }).unwrap().then((res) => {
                setErrorInfo('');
                setSelected({ username: "", password: "" });
                users.refetch();
            }).catch(err => {
                setErrorInfo(JSON.stringify(err));
                toast.error(`Error userUpdate ${data.username}`, { data: { error: err.data } });
                console.error(err);
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
            .then(({ confirmed, reason }) => {
                if (confirmed) {
                    if (!data.username) {
                        setErrorInfo('Unable to delete user!');
                        return;
                    }
                    userDelete({ username: data.username }).unwrap().then((res) => {
                        setErrorInfo('')
                        setSelected({ username: "", password: "" });
                        users.refetch();
                    }).catch(err => {
                        setErrorInfo(JSON.stringify(err));
                    })
                } else if (reason === "cancel") {
                    console.log("cancel")
                }
            })
    }

    return (
        <InView>
            <UserEditDialog objectToEdit={selected} open={showEdit} onClose={(data) => { setSelected({ username: "", password: "", doCreate: false }); onSubmitEditUser(data); setShowEdit(false) }} />
            {read_only || <Fab color="primary" aria-label="add" sx={{
                float: 'right',
                top: '-20px',
                margin: '-8px'
            }} size="small"
                onClick={() => { setSelected({ username: "", password: "", doCreate: true }); setShowEdit(true) }}
            >
                <PersonAddIcon />
            </Fab>}
            <List dense={true}>
                {users.isSuccess && Array.isArray(users.data) && users.data.slice().sort((a, b) => {
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
                                secondaryAction={!read_only && <>
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
                is_admin: false,
                doCreate: true
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
                                    <TextFieldElement name="username" label="User Name" required control={control}
                                        slotProps={props.objectToEdit?.username ? (props.objectToEdit.is_admin ? {} : {
                                            input: {
                                                readOnly: true,
                                            },
                                        }) : {}}
                                    />
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
