import AdminPanelSettingsIcon from "@mui/icons-material/AdminPanelSettings";
import AssignmentIndIcon from "@mui/icons-material/AssignmentInd";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import FolderSharedIcon from "@mui/icons-material/FolderShared";
import VisibilityIcon from "@mui/icons-material/Visibility";
import {
    Avatar,
    Box,
    Card,
    CardContent,
    CardHeader,
    Chip,
    Divider,
    Grid,
    IconButton,
    Stack,
    Tooltip,
    Typography,
} from "@mui/material";
import { useState } from "react";
import { PreviewDialog } from "../../../components/PreviewDialog";
import type { User } from "../../../store/sratApi";
import type { UsersProps } from "../types";

interface UserDetailsPanelProps {
    user?: User;
    userKey?: string;
    onEdit?: (data: UsersProps) => void;
    onDelete?: (user: User) => void;
    onEditClick?: () => void;
    onCancelEdit?: () => void;
    isEditing?: boolean;
    readOnly?: boolean;
    children?: React.ReactNode;
}

export function UserDetailsPanel({
    user,
    userKey,
    onDelete,
    onEditClick,
    onCancelEdit,
    isEditing = false,
    readOnly = false,
    children,
}: UserDetailsPanelProps) {
    const [showUserPreview, setShowUserPreview] = useState(false);
    if (!user || !userKey) {
        return (
            <Box
                sx={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    height: "100%",
                    color: "text.secondary",
                }}
            >
                <Typography variant="h6">
                    Select a user to view details
                </Typography>
            </Box>
        );
    }

    const userRwShares = user.rw_shares || [];
    const userRoShares = user.ro_shares || [];

    return (
        <Box
            sx={{
                height: "100%",
                overflow: "auto",
                p: 2,
            }}
        >
            <Stack spacing={3}>
                {/* User Profile Card */}
                <Card>
                    <CardHeader
                        avatar={
                            <Tooltip title="View user details">
                                <Avatar
                                    sx={{ bgcolor: user.is_admin ? "warning.main" : "primary.main" }}
                                    onClick={() => setShowUserPreview(true)}
                                >
                                    {user.is_admin ? (
                                        <AdminPanelSettingsIcon />
                                    ) : (
                                        <AssignmentIndIcon />
                                    )}
                                </Avatar>
                            </Tooltip>
                        }
                        title={
                            <Typography variant="h5">
                                {user.username}
                            </Typography>
                        }
                        subheader={
                            <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                                {user.is_admin && (
                                    <Chip
                                        label="Administrator"
                                        color="warning"
                                        size="small"
                                    />
                                )}
                                <Chip
                                    label={userRwShares.length + userRoShares.length > 0
                                        ? `${userRwShares.length + userRoShares.length} share(s) assigned`
                                        : "No shares assigned"
                                    }
                                    color={userRwShares.length + userRoShares.length > 0 ? "success" : "default"}
                                    variant="outlined"
                                    size="small"
                                />
                            </Stack>
                        }
                        action={
                            !readOnly && (
                                <Stack direction="row" spacing={1}>
                                    <Tooltip title={isEditing ? "View User" : "Edit User"}>
                                        <IconButton
                                            color="primary"
                                            size="small"
                                            onClick={isEditing ? onCancelEdit : onEditClick}
                                        >
                                            {isEditing ? <VisibilityIcon /> : <EditIcon />}
                                        </IconButton>
                                    </Tooltip>
                                    {!user.is_admin && onDelete && (
                                        <Tooltip title="Delete User">
                                            <IconButton
                                                color="error"
                                                size="small"
                                                onClick={() => onDelete(user)}
                                            >
                                                <DeleteIcon />
                                            </IconButton>
                                        </Tooltip>
                                    )}
                                </Stack>
                            )
                        }
                    />
                </Card>

                {/* Edit Form or User Information Card */}
                <Card>
                    <CardHeader
                        title={isEditing ? "Edit User" : "User Information"}
                    />
                    {isEditing && children ? (
                        children
                    ) : (
                        <CardContent>
                            <Grid container spacing={3}>
                                {/* Read-Write Shares */}
                                <Grid size={{ xs: 12, md: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1, display: "flex", alignItems: "center", gap: 1 }}>
                                        <EditIcon fontSize="small" />
                                        Read/Write Shares
                                    </Typography>
                                    {userRwShares.length > 0 ? (
                                        <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                            {userRwShares.map((share) => (
                                                <Chip
                                                    key={share}
                                                    icon={<FolderSharedIcon />}
                                                    label={share}
                                                    variant="outlined"
                                                    size="small"
                                                />
                                            ))}
                                        </Stack>
                                    ) : (
                                        <Typography variant="body2" color="text.secondary" sx={{ fontStyle: "italic" }}>
                                            No read/write shares assigned
                                        </Typography>
                                    )}
                                </Grid>

                                {/* Read-Only Shares */}
                                <Grid size={{ xs: 12, md: 6 }}>
                                    <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1, display: "flex", alignItems: "center", gap: 1 }}>
                                        <VisibilityIcon fontSize="small" />
                                        Read-Only Shares
                                    </Typography>
                                    {userRoShares.length > 0 ? (
                                        <Stack direction="row" spacing={1} flexWrap="wrap" sx={{ gap: 1 }}>
                                            {userRoShares.map((share) => (
                                                <Chip
                                                    key={share}
                                                    icon={<FolderSharedIcon />}
                                                    label={share}
                                                    variant="outlined"
                                                    color="secondary"
                                                    size="small"
                                                />
                                            ))}
                                        </Stack>
                                    ) : (
                                        <Typography variant="body2" color="text.secondary" sx={{ fontStyle: "italic" }}>
                                            No read-only shares assigned
                                        </Typography>
                                    )}
                                </Grid>

                                {/* User Type Info */}
                                <Grid size={12}>
                                    <Divider sx={{ my: 2 }} />
                                    <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                        User Type
                                    </Typography>
                                    <Typography variant="body2">
                                        {user.is_admin
                                            ? "This is the administrator account. The admin user can be renamed but not deleted. There can only be one admin user."
                                            : "This is a regular user account. Regular users can be edited or deleted."}
                                    </Typography>
                                </Grid>
                            </Grid>
                        </CardContent>
                    )}
                </Card>
            </Stack>


            {/* User Preview Dialog */}
            <PreviewDialog
                title={`User: ${user?.username || 'N/A'}`}
                objectToDisplay={user}
                open={showUserPreview}
                onClose={() => setShowUserPreview(false)}
            />
        </Box>
    );
}
