import { Fragment, useState } from "react";
import { useForm } from "react-hook-form";
import { Fab, List, ListItemButton, ListItem, IconButton, ListItemAvatar, Avatar, ListItemText, Divider, Dialog, DialogTitle, Stack, DialogContent, DialogContentText, Grid as Grid, DialogActions, Button, Chip, Box, Tooltip, Typography } from "@mui/material";
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
import EditIcon from '@mui/icons-material/Edit';
import VisibilityIcon from '@mui/icons-material/Visibility';



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
            description: "Do you really would delete this user?",
            acknowledgement: "I understand that deleting the share will remove it permanently.",

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
            <br />
            <Stack direction="row" justifyContent="flex-end" sx={{ px: 2, mb: 1, alignItems: 'center' }}>
                {read_only || <Fab
                    color="primary"
                    aria-label="add"
                    // sx removed: float, top, margin - FAB is now in normal flow within Stack
                    size="small"
                    onClick={() => { setSelected({ username: "", password: "", doCreate: true }); setShowEdit(true) }}
                >
                    <PersonAddIcon />
                </Fab>}
            </Stack>
            <List dense={true} component="span">
                <Divider />
                {users.isSuccess && Array.isArray(users.data) && users.data.slice().sort((a, b) => {
                    // Sort admin users to the top, then alphabetically by username
                    if (a.is_admin && !b.is_admin) return -1;
                    if (!a.is_admin && b.is_admin) return 1;
                    return (a.username || "").localeCompare(b.username || "");
                }).map((user) => {
                    const userRwShares = user.rw_shares || [];
                    const userRoShares = user.ro_shares || [];

                    return <Fragment key={user.username || "admin"}>
                        <ListItemButton sx={{ alignItems: 'flex-start' }}>
                            <ListItemAvatar sx={{ pt: 1 }}>
                                <Avatar>
                                    {user.is_admin ?
                                        <AdminPanelSettingsIcon /> :
                                        <AssignmentIndIcon />}
                                </Avatar>
                            </ListItemAvatar>
                            <ListItemText
                                sx={{ flexGrow: 1, overflowWrap: 'break-word' }}
                                primary={user.username}
                                slotProps={{
                                    secondary: {
                                        component: 'span',

                                    }
                                }}
                                secondary={
                                    <Stack direction="row" spacing={1} flexWrap="wrap" alignItems="center" sx={{ mt: 0.5, display: { xs: 'none', sm: 'flex' } }}>
                                        {userRwShares.length > 0 && (
                                            <Tooltip title={`Shares with read-write access for ${user.username}`}>
                                                <Chip
                                                    icon={<EditIcon fontSize="small" />}
                                                    label={
                                                        <Box component="span" sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 0.5 }}>
                                                            Shares:
                                                            {userRwShares.map((share, index) => (
                                                                <Typography component="span" variant="caption" key={share}>
                                                                    {share}{index < userRwShares.length - 1 ? ',' : ''}
                                                                </Typography>
                                                            ))}
                                                        </Box>
                                                    }
                                                    size="small"
                                                    variant="outlined"
                                                    onClick={(e) => { e.stopPropagation(); /* Optionally handle click, e.g., navigate to shares page */ }}
                                                />
                                            </Tooltip>
                                        )}
                                        {userRoShares.length > 0 && (
                                            <Tooltip title={`Shares with read-only access for ${user.username}`}>
                                                <Chip
                                                    icon={<VisibilityIcon fontSize="small" />}
                                                    label={
                                                        <Box component="span" sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 0.5 }}>
                                                            Shares:
                                                            {userRoShares.map((share, index) => (
                                                                <Typography component="span" variant="caption" key={share}>
                                                                    {share}{index < userRoShares.length - 1 ? ',' : ''}
                                                                </Typography>
                                                            ))}
                                                        </Box>
                                                    }
                                                    size="small"
                                                    variant="outlined"
                                                    onClick={(e) => { e.stopPropagation(); /* Optionally handle click */ }}
                                                />
                                            </Tooltip>
                                        )}
                                        {(userRwShares.length === 0 && userRoShares.length === 0) && (
                                            <Typography variant="caption" sx={{ fontStyle: 'italic' }}>No shares assigned</Typography>
                                        )}
                                    </Stack>
                                }
                            />
                            {!read_only && <Stack direction="column" spacing={0} sx={{ pl: 1 }}>
                                <IconButton onClick={() => { setSelected(user); setShowEdit(true) }} edge="end" aria-label="settings" size="small">
                                    <ManageAccountsIcon />
                                </IconButton>
                                {!user.is_admin &&
                                    <IconButton onClick={() => onSubmitDeleteUser(user)} edge="end" aria-label="delete" size="small">
                                        <PersonRemoveIcon />
                                    </IconButton>
                                }
                            </Stack>}
                        </ListItemButton>
                        <Divider component="li" />
                    </Fragment>
                })}
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
                            Please enter the username and password for the user.
                        </DialogContentText>
                        <form id="editshareform" onSubmit={handleSubmit(handleCloseSubmit)} noValidate>
                            <Grid container spacing={2}>
                                <Grid size={6}>
                                    <TextFieldElement size="small" name="username" autoComplete="username" label="User Name" required control={control}
                                        slotProps={props.objectToEdit?.username ? (props.objectToEdit.is_admin ? {} : {
                                            input: {
                                                readOnly: true,
                                            },
                                        }) : {}}
                                    />
                                </Grid>
                                <Grid size={6}>
                                    <PasswordElement size="small" autoComplete="new-password" name="password" label="Password"
                                        required control={control} />
                                    <PasswordRepeatElement size="small" autoComplete="new-password" passwordFieldName={'password'} name={'password-repeat'} margin={'dense'} label={'Repeat Password'} required control={control} />
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
