import { faCheckDouble, faTrashCan } from "@fortawesome/free-solid-svg-icons";
import CheckIcon from "@mui/icons-material/Check";
import CloseIcon from "@mui/icons-material/Close";
import NotificationsIcon from "@mui/icons-material/Notifications";
import {
  Alert,
  Badge,
  Box,
  CardContent,
  Divider,
  FormControlLabel,
  IconButton,
  Popover,
  Stack,
  Switch,
  Toolbar,
  Tooltip,
  Typography,
  useColorScheme,
  useMediaQuery,
} from "@mui/material";
import Card from "@mui/material/Card";
import { isValidElement, type ReactNode, useState } from "react";
import { Slide, ToastContainer } from "react-toastify";
import { useNotificationCenter } from "react-toastify/addons/use-notification-center";
import type { ErrorModel } from "../store/sratApi";
import { FontAwesomeSvgIcon } from "./FontAwesomeSvgIcon";

interface Data {
  exclude: boolean;
  error?: unknown;
}

export function NotificationCenter() {
  const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null);
  const [showRead, setShowRead] = useState(false);
  const prefersDarkMode = useMediaQuery("(prefers-color-scheme: dark)");
  const { mode } = useColorScheme();

  const {
    notifications,
    unreadCount,
    clear,
    markAllAsRead,
    remove,
    markAsRead,
  } = useNotificationCenter<Data>({
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
  });

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const open = Boolean(anchorEl);
  const id = open ? "simple-popover" : undefined;

  function toMUISeverity(
    severity?: string,
  ): "error" | "info" | "success" | "warning" {
    switch (severity) {
      case "error":
        return "error";
      case "warning":
        return "warning";
      case "info":
        return "info";
      case "success":
        return "success";
      default:
        return "info";
    }
  }

  function formatProblemValue(value: unknown): string | undefined {
    if (value === null || value === undefined) {
      return undefined;
    }

    if (
      typeof value === "string" ||
      typeof value === "number" ||
      typeof value === "boolean" ||
      typeof value === "bigint"
    ) {
      return String(value);
    }

    if (Array.isArray(value)) {
      const items = value
        .map((entry) => formatProblemValue(entry))
        .filter((entry): entry is string => Boolean(entry));
      return items.length > 0 ? items.join(", ") : undefined;
    }

    if (typeof value === "object") {
      const structuredValue = value as Partial<ErrorModel> & {
        message?: string;
      };
      const summary =
        structuredValue.title ??
        structuredValue.detail ??
        structuredValue.message;
      if (summary) {
        return summary;
      }

      try {
        return JSON.stringify(value, null, 2);
      } catch {
        return String(value);
      }
    }

    return String(value);
  }

  function renderNotificationContent(content: unknown): ReactNode {
    if (content === null || content === undefined || content === false) {
      return null;
    }

    if (isValidElement(content)) {
      return content;
    }

    return formatProblemValue(content) ?? null;
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
      <IconButton aria-describedby={id} onClick={handleClick}>
        <Tooltip title="Report issue!" arrow>
          <Badge color="secondary" badgeContent={unreadCount} max={999}>
            <NotificationsIcon sx={{ color: "white" }} />
          </Badge>
        </Tooltip>
      </IconButton>
      <Popover
        id={id}
        open={open}
        anchorEl={anchorEl}
        onClose={handleClose}
        anchorOrigin={{
          vertical: "bottom",
          horizontal: "right",
        }}
        transformOrigin={{
          vertical: "top",
          horizontal: "right",
        }}
      >
        <Card variant="outlined" sx={{ minWidth: "22em" }}>
          <CardContent>
            <Toolbar variant="dense">
              <Box sx={{ flexGrow: 1, display: { xs: "none", md: "flex" } }}>
                <Typography variant="subtitle1" color="inherit" component="div">
                  Notifications
                </Typography>
              </Box>
              <Box sx={{ flexGrow: 0 }}>
                <FormControlLabel
                  value="end"
                  control={
                    <Switch
                      color="primary"
                      checked={showRead}
                      size="small"
                      onChange={(e) => setShowRead(e.target.checked)}
                    />
                  }
                  label={showRead ? "All" : "Not readed"}
                  labelPlacement="end"
                />
              </Box>
            </Toolbar>
            <Divider />
            <Stack sx={{ width: "100%", minHeight: "10em" }} spacing={0}>
              {notifications
                .filter((n) => !n.read || showRead)
                .map((notification) => {
                  const renderedContent = renderNotificationContent(
                    notification.content,
                  );
                  const structuredError =
                    notification.data?.error &&
                    typeof notification.data.error === "object"
                      ? (notification.data.error as Partial<ErrorModel> & {
                          message?: string;
                          errors?: Array<{
                            message?: string;
                            value?: unknown;
                            location?: string | string[];
                          }> | null;
                        })
                      : undefined;
                  const fallbackError =
                    !structuredError?.title &&
                    !structuredError?.detail &&
                    !structuredError?.message &&
                    !structuredError?.errors?.length
                      ? formatProblemValue(notification.data?.error)
                      : undefined;

                  return (
                    <Alert
                      key={notification.id}
                      variant={notification.read ? "standard" : "outlined"}
                      severity={toMUISeverity(notification.type)}
                      action={
                        <>
                          <Tooltip title="Remove notification" arrow>
                            <IconButton
                              onClickCapture={() => remove(notification.id)}
                              aria-label="close"
                              color="inherit"
                              size="small"
                              onClick={() => {}}
                            >
                              <CloseIcon fontSize="inherit" />
                            </IconButton>
                          </Tooltip>
                          {!notification.read && (
                            <Tooltip title="Mark as read" arrow>
                              <IconButton
                                onClickCapture={() =>
                                  markAsRead(notification.id)
                                }
                                aria-label="close"
                                color="inherit"
                                size="small"
                                onClick={() => {}}
                              >
                                <CheckIcon fontSize="inherit" />
                              </IconButton>
                            </Tooltip>
                          )}
                        </>
                      }
                      sx={{ mb: 2 }}
                    >
                      {renderedContent && (
                        <>
                          <Box sx={{ mb: 1, whiteSpace: "pre-wrap" }}>
                            {renderedContent}
                          </Box>
                          <Divider />
                        </>
                      )}
                      {structuredError?.title && (
                        <Typography variant="body2" gutterBottom>
                          {structuredError.title}
                        </Typography>
                      )}
                      {structuredError?.detail && (
                        <Typography variant="body2" gutterBottom>
                          {structuredError.detail}
                        </Typography>
                      )}
                      {structuredError?.message &&
                        structuredError.message !== structuredError.title &&
                        structuredError.message !== structuredError.detail && (
                          <Typography variant="body2" gutterBottom>
                            {structuredError.message}
                          </Typography>
                        )}
                      {structuredError?.errors?.map((error, index) => {
                        const valueText = formatProblemValue(error.value);
                        const locationText = Array.isArray(error.location)
                          ? error.location.join(" → ")
                          : error.location;
                        const line = [error.message, valueText, locationText]
                          .filter(Boolean)
                          .join(" ");

                        return (
                          // biome-ignore lint/suspicious/noArrayIndexKey: no unique id available on error items
                          <Typography key={index} variant="body2" gutterBottom>
                            {line}
                          </Typography>
                        );
                      })}
                      {fallbackError && (
                        <Typography
                          variant="body2"
                          gutterBottom
                          sx={{ whiteSpace: "pre-wrap" }}
                        >
                          {fallbackError}
                        </Typography>
                      )}
                    </Alert>
                  );
                })}
            </Stack>
            <Divider />
            <Toolbar variant="dense" disableGutters>
              <Box sx={{ flexGrow: 1, display: { xs: "flex", md: "flex" } }}>
                <Tooltip title="Clear notifications" arrow>
                  <Box component="span" sx={{ display: "inline-flex" }}>
                    <IconButton
                      color="inherit"
                      onClick={clear}
                      disabled={
                        notifications.filter((n) => !n.read || showRead)
                          .length === 0
                      }
                    >
                      <FontAwesomeSvgIcon icon={faTrashCan} />
                    </IconButton>
                  </Box>
                </Tooltip>
              </Box>
              <Box sx={{ flexGrow: 0 }}>
                <Tooltip title="Mark all read" arrow>
                  <Box component="span" sx={{ display: "inline-flex" }}>
                    <IconButton
                      color="inherit"
                      onClick={markAllAsRead}
                      disabled={
                        notifications.filter((n) => !n.read).length === 0
                      }
                    >
                      <FontAwesomeSvgIcon icon={faCheckDouble} />
                    </IconButton>
                  </Box>
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
  );
}
