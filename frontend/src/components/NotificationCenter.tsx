import { Alert, Badge, Box, CardContent, Divider, FormControlLabel, IconButton, Popover, Stack, Switch, Toolbar, Tooltip, Typography, useColorScheme, useMediaQuery } from "@mui/material";
import { ToastContainer, Slide } from "react-toastify";
import { useNotificationCenter } from "react-toastify/addons/use-notification-center"
import NotificationsIcon from '@mui/icons-material/Notifications';
import CloseIcon from '@mui/icons-material/Close';
import { useState } from "react";
import Card from '@mui/material/Card';
import { FontAwesomeSvgIcon } from "./FontAwesomeSvgIcon";
import { faCheckDouble, faTrashCan } from "@fortawesome/free-solid-svg-icons";
import CheckIcon from '@mui/icons-material/Check';

interface Data {
    exclude: boolean
}

export function NotificationCenter() {
    const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null);
    const [showRead, setShowRead] = useState(false);
    const prefersDarkMode = useMediaQuery('(prefers-color-scheme: dark)');
    const { mode } = useColorScheme();


    const { notifications, unreadCount, clear, markAllAsRead, remove, markAsRead } = useNotificationCenter<Data>({
        //        data: [
        //            {
        //                id: "anId", createdAt: Date.now(), data: { exclude: false },
        //                read: false
        //            },
        //            {
        //                id: "anotherId", createdAt: Date.now(), data: { exclude: true },
        //                read: false
        //            }
        //        ],
        sort: (l, r) => l.createdAt - r.createdAt,
        //  filter: (item) => item.read || showUnRead
    })

    const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const open = Boolean(anchorEl);
    const id = open ? 'simple-popover' : undefined;

    function toMUISeverity(severity?: string): 'error' | 'info' | 'success' | 'warning' {
        switch (severity) {
            case 'error':
                return 'error';
            case 'warning':
                return 'warning';
            case 'info':
                return 'info';
            case 'success':
                return 'success';
            default: return 'info';
        }
    }


    /*
    useEffect(() => {
        toast.error("Error message!");
        toast.warn("Warning message!");
        toast.dark("Dark mode message molto ma molto lungo pure troppo cosa possiamo fare con questo lunghissimo messaggio?");
        toast.success("Success message!");
        toast.warning("Warning message!");
        toast.info("Info message!");
    }, []);
    */

    return (
        <>
            <IconButton aria-describedby={id} onClick={handleClick} >
                <Tooltip title="Report issue!" arrow>
                    <Badge color="secondary" badgeContent={unreadCount} max={999}>
                        <NotificationsIcon sx={{ color: 'white' }} />
                    </Badge>
                </Tooltip>
            </IconButton>
            <Popover
                id={id}
                open={open}
                anchorEl={anchorEl}
                onClose={handleClose}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'right',
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'right',
                }}
            >
                <Card variant="outlined" sx={{ minWidth: '22em' }}>
                    <CardContent>
                        <Toolbar variant="dense">
                            <Box sx={{ flexGrow: 1, display: { xs: 'none', md: 'flex' } }}>
                                <Typography variant="subtitle1" color="inherit" component="div">Notifications</Typography>
                            </Box>
                            <Box sx={{ flexGrow: 0 }}>
                                <FormControlLabel
                                    value="end"
                                    control={<Switch color="primary" checked={showRead} size="small" onChange={(e) => setShowRead(e.target.checked)} />}
                                    label={showRead ? "All" : "Not readed"}
                                    labelPlacement="end"
                                />
                            </Box>
                        </Toolbar>
                        <Divider />
                        {/*
                    <ul>
                        {notifications.map(notification => (
                            <li key={notification.id}>
                                <span>{JSON.stringify(notification)}</span>
                            </li>
                        ))}
                    </ul>
                    */}
                        <Stack sx={{ width: '100%', minHeight: '10em' }} spacing={0}>
                            {notifications.filter((n) => !n.read || showRead).map(notification => (
                                <Alert
                                    key={notification.id}
                                    variant={notification.read ? 'standard' : 'outlined'}
                                    severity={toMUISeverity(notification.type)}
                                    action={
                                        <>
                                            <Tooltip title="Remove notification" arrow>
                                                <IconButton onClickCapture={() => remove(notification.id)}
                                                    aria-label="close"
                                                    color="inherit"
                                                    size="small"
                                                    onClick={() => {
                                                    }}
                                                >
                                                    <CloseIcon fontSize="inherit" />
                                                </IconButton>
                                            </Tooltip>
                                            {!notification.read &&
                                                <Tooltip title="Mark as read" arrow>

                                                    <IconButton onClickCapture={() => markAsRead(notification.id)}
                                                        aria-label="close"
                                                        color="inherit"
                                                        size="small"
                                                        onClick={() => {
                                                        }}
                                                    >
                                                        <CheckIcon fontSize="inherit" />
                                                    </IconButton>
                                                </Tooltip>
                                            }
                                        </>
                                    }
                                    sx={{ mb: 2 }}
                                >
                                    {notification.content?.toLocaleString()}
                                </Alert>
                            ))}
                        </Stack>
                        <Divider />
                        <Toolbar variant="dense" disableGutters>
                            <Box sx={{ flexGrow: 1, display: { xs: 'flex', md: 'flex' } }}>
                                <Tooltip title="Clear notifications" arrow>
                                    <IconButton color="inherit" onClick={clear} disabled={notifications.filter((n) => !n.read || showRead).length == 0}>
                                        <FontAwesomeSvgIcon icon={faTrashCan} />
                                    </IconButton>
                                </Tooltip>
                            </Box>
                            <Box sx={{ flexGrow: 0 }}>
                                <Tooltip title="Mark all read" arrow>
                                    <IconButton color="inherit" onClick={markAllAsRead} disabled={notifications.filter((n) => !n.read).length == 0}>
                                        <FontAwesomeSvgIcon icon={faCheckDouble} />
                                    </IconButton>
                                </Tooltip>
                            </Box>
                        </Toolbar>
                    </CardContent>

                </Card>
            </Popover>
            <ToastContainer
                stacked
                position="top-right"
                autoClose={5000}
                hideProgressBar={false}
                newestOnTop={false}
                closeOnClick
                rtl={false}
                pauseOnFocusLoss
                draggable
                pauseOnHover
                theme={mode === "system" ? (prefersDarkMode ? "dark" : "light") : mode}
                transition={Slide}
                limit={10}
            />
        </>

    )
}