import { DriveFileMove } from "@mui/icons-material";
import BlockIcon from "@mui/icons-material/Block";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import DeleteIcon from "@mui/icons-material/Delete";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import SettingsIcon from "@mui/icons-material/Settings";
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
import { useState } from "react";
import { TabIDs } from "../../store/locationState";
import type { SharedResource } from "../../store/sratApi";
import { Usage } from "../../store/sratApi";

interface ShareActionsProps {
	shareKey: string;
	shareProps: SharedResource;
	read_only: boolean;
	onEdit: (shareKey: string, shareProps: SharedResource) => void;
	onViewVolumeSettings: (shareProps: SharedResource) => void;
	onDelete: (shareKey: string, shareProps: SharedResource) => void;
	onEnable: (shareKey: string, shareProps: SharedResource) => void;
	onDisable: (shareKey: string, shareProps: SharedResource) => void;
}

export function ShareActions({
	shareKey,
	shareProps,
	read_only,
	onEdit,
	onViewVolumeSettings,
	onDelete,
	onEnable,
	onDisable,
}: ShareActionsProps) {
	const theme = useTheme();
	const isSmallScreen = useMediaQuery(theme.breakpoints.down("sm"));
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

	if (read_only) {
		return null;
	}

	const actionItems = [];

	actionItems.push({
		key: "edit",
		title: "Settings",
		icon: <SettingsIcon />,
		onClick: () => onEdit(shareKey, shareProps),
	});

	if (
		!shareProps.mount_point_data?.invalid &&
		shareProps.usage !== Usage.Internal &&
		shareProps.mount_point_data?.path_hash
	) {
		actionItems.push({
			key: "view-volume",
			title: "View Volume Mount Settings",
			icon: <DriveFileMove />,
			onClick: () => onViewVolumeSettings(shareProps),
		});
	}

	if (shareProps.usage !== Usage.Internal) {
		actionItems.push({
			key: "delete",
			title: "Delete share",
			icon: <DeleteIcon color="error" />,
			onClick: () => onDelete(shareKey, shareProps),
		});
	}

	if (shareProps.disabled) {
		actionItems.push({
			key: "enable",
			title: "Enable share",
			icon: <CheckCircleIcon />,
			onClick: () => onEnable(shareKey, shareProps),
		});
	} else if (shareProps.usage !== Usage.Internal) {
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
