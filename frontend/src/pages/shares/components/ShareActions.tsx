import { DriveFileMove } from "@mui/icons-material";
import BlockIcon from "@mui/icons-material/Block";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import {
    IconButton,
    ListItemIcon,
    ListItemText,
    Menu,
    MenuItem,
    Stack,
    Tooltip,
    useMediaQuery,
    useTheme,
} from "@mui/material";
import { useState, type ReactElement } from "react";
import { TabIDs } from "../../../store/locationState";
import { Usage } from "../../../store/sratApi";
import type { SharedResource } from "../../../store/sratApi";

interface ShareActionsProps {
    shareKey: string;
    shareProps: SharedResource;
    protected_mode: boolean;
    onViewVolumeSettings: (shareProps: SharedResource) => void;
    onEnable: (shareKey: string, shareProps: SharedResource) => void;
    onDisable: (shareKey: string, shareProps: SharedResource) => void;
}

export function ShareActions({
    shareKey,
    shareProps,
    onViewVolumeSettings,
    onEnable,
    onDisable,
}: ShareActionsProps) {
    const theme = useTheme();
    const isSmallScreen = useMediaQuery(theme.breakpoints.between("sm", "md"));
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

    const handleMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
        event.stopPropagation();
        setAnchorEl(event.currentTarget);
    };

    const handleMenuClose = (
        e?: React.MouseEvent<HTMLElement> | {},
        _reason?: "backdropClick" | "escapeKeyDown",
    ) => {
        (e as React.MouseEvent<HTMLElement>)?.stopPropagation();
        setAnchorEl(null);
    };

    const actionItems: { key: string; title: string; icon: ReactElement; onClick: () => void }[] = [];

    if (
        !shareProps.mount_point_data?.invalid &&
        shareProps.usage !== Usage.Internal &&
        shareProps.mount_point_data?.path
    ) {
        actionItems.push({
            key: "view-volume",
            title: "View Volume Mount Settings",
            icon: <DriveFileMove />,
            onClick: () => onViewVolumeSettings(shareProps),
        });
    }

    if (shareProps.disabled) {
        actionItems.push({
            key: "enable",
            title: "Enable share",
            icon: <CheckCircleIcon />,
            onClick: () => onEnable(shareKey, shareProps),
        });
    } else {
        actionItems.push({
            key: "disable",
            title: "Disable share",
            icon: <BlockIcon />,
            onClick: () => onDisable(shareKey, shareProps),
        });
    }

    if (isSmallScreen) {
        return (
            <div data-tutor={`reactour__tab${TabIDs.SHARES}__step4`}>
                <IconButton
                    aria-label="more actions"
                    aria-controls="share-actions-menu"
                    aria-haspopup="true"
                    onClick={handleMenuOpen}
                    edge="end"
                    size="small"
                >
                    <MoreVertIcon />
                </IconButton>
                <Menu
                    id="share-actions-menu"
                    anchorEl={anchorEl}
                    open={Boolean(anchorEl)}
                    onClose={handleMenuClose}
                    onClick={(e) => e.stopPropagation()}
                >
                    {actionItems.map((action) => (
                        <MenuItem
                            key={action.key}
                            onClick={(e) => {
                                e.stopPropagation();
                                action.onClick();
                                handleMenuClose();
                            }}
                        >
                            <ListItemIcon>{action.icon}</ListItemIcon>
                            <ListItemText>{action.title}</ListItemText>
                        </MenuItem>
                    ))}
                </Menu>
            </div>
        );
    }

    return (
        <Stack
            direction="row"
            spacing={0}
            alignItems="center"
            data-tutor={`reactour__tab${TabIDs.SHARES}__step4`}
        >
            {actionItems.map((action) => (
                <Tooltip title={action.title} key={action.key}>
                    <IconButton
                        onClick={(e) => {
                            e.stopPropagation();
                            action.onClick();
                        }}
                        edge="end"
                        aria-label={action.title.toLowerCase()}
                        size="small"
                    >
                        {action.icon}
                    </IconButton>
                </Tooltip>
            ))}
        </Stack>
    );
} 