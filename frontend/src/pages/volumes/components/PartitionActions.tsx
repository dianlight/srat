import {
  faPlug,
  faPlugCircleBolt,
  faPlugCircleExclamation,
  faPlugCircleMinus,
  faPlugCircleXmark,
} from "@fortawesome/free-solid-svg-icons";
import AddIcon from "@mui/icons-material/Add";
import DeleteSweepIcon from "@mui/icons-material/DeleteSweep";
import FactCheckIcon from "@mui/icons-material/FactCheck";
import LabelIcon from "@mui/icons-material/Label";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import ShareIcon from "@mui/icons-material/Share";
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
import { type ReactElement, useState } from "react";
import { FontAwesomeSvgIcon } from "../../../components/FontAwesomeSvgIcon";
import type { Partition } from "../../../store/sratApi";
import { usePartitionActions } from "../hooks/usePartitionActions";
import type { PartitionActionKey } from "./PartitionActionItems";

interface PartitionActionsProps {
  partition: Partition;
  protected_mode: boolean;
  onToggleAutomount: (partition: Partition) => void;
  onMount: (partition: Partition) => void;
  onUnmount: (partition: Partition, force: boolean) => void;
  onCreateShare: (partition: Partition) => void;
  onGoToShare: (partition: Partition) => void;
  onCheckFilesystem?: (partition: Partition) => void;
  onSetFilesystemLabel?: (partition: Partition) => void;
  onFormatPartition?: (partition: Partition) => void;
}

export function PartitionActions({
  partition,
  protected_mode,
  onToggleAutomount,
  onMount,
  onUnmount,
  onCreateShare,
  onGoToShare,
  onCheckFilesystem,
  onSetFilesystemLabel,
  onFormatPartition,
}: PartitionActionsProps) {
  const theme = useTheme();
  const isSmallScreen = useMediaQuery(theme.breakpoints.down("lg"));
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
    event.stopPropagation();
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = (
    e?: React.MouseEvent<HTMLElement> | Record<string, never>,
  ) => {
    (e as React.MouseEvent<HTMLElement>)?.stopPropagation();
    setAnchorEl(null);
  };
  const actionItems = usePartitionActions({
    partition,
    protectedMode: protected_mode,
    onToggleAutomount,
    onMount,
    onUnmount,
    onCreateShare,
    onGoToShare,
    onCheckFilesystem,
    onSetFilesystemLabel,
    onFormatPartition,
  });

  if (!actionItems || actionItems.length === 0) {
    return null;
  }

  // All FontAwesome partition action icons share the same test id
  // intentionally; tests must query with getAllByTestId("partition-action-icon").
  const actionIcons: Record<PartitionActionKey, ReactElement | null> = {
    mount: (
      <FontAwesomeSvgIcon
        icon={faPlug}
        fontSize="small"
        data-testid="partition-action-icon"
      />
    ),
    "enable-automount": (
      <FontAwesomeSvgIcon
        icon={faPlugCircleBolt}
        fontSize="small"
        data-testid="partition-action-icon"
      />
    ),
    "disable-automount": (
      <FontAwesomeSvgIcon
        icon={faPlugCircleXmark}
        fontSize="small"
        data-testid="partition-action-icon"
      />
    ),
    unmount: (
      <FontAwesomeSvgIcon
        icon={faPlugCircleMinus}
        fontSize="small"
        data-testid="partition-action-icon"
      />
    ),
    "force-unmount": (
      <FontAwesomeSvgIcon
        icon={faPlugCircleExclamation}
        fontSize="small"
        data-testid="partition-action-icon"
      />
    ),
    "create-share": <AddIcon fontSize="small" />,
    "go-to-share": <ShareIcon fontSize="small" />,
    "check-filesystem": <FactCheckIcon fontSize="small" />,
    "set-label": <LabelIcon fontSize="small" />,
    format: <DeleteSweepIcon fontSize="small" />,
  };

  if (isSmallScreen) {
    return (
      <>
        <IconButton
          aria-label="more actions"
          aria-controls="partition-actions-menu"
          aria-haspopup="true"
          onClick={handleMenuOpen}
          edge="end"
          size="small"
        >
          <MoreVertIcon />
        </IconButton>
        <Menu
          id="partition-actions-menu"
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
              <ListItemIcon>{actionIcons[action.key]}</ListItemIcon>
              <ListItemText>{action.title}</ListItemText>
            </MenuItem>
          ))}
        </Menu>
      </>
    );
  }

  return (
    <Stack
      direction="row"
      spacing={0}
      sx={{
        alignItems: "center",
        pr: 1,
        flexWrap: "wrap",
      }}
    >
      {actionItems
        .filter((action) => actionIcons[action.key])
        .map((action) => (
          <Tooltip title={action.title} key={action.key}>
            <IconButton
              onClick={(e) => {
                e.stopPropagation();
                action.onClick();
              }}
              edge="end"
              aria-label={action.title.toLowerCase()}
              size="small"
              color={action.color}
            >
              {actionIcons[action.key]}
            </IconButton>
          </Tooltip>
        ))}
    </Stack>
  );
}
