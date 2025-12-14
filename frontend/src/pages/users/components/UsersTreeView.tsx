import AdminPanelSettingsIcon from "@mui/icons-material/AdminPanelSettings";
import AssignmentIndIcon from "@mui/icons-material/AssignmentInd";
import GroupIcon from "@mui/icons-material/Group";
import {
    Box,
    Chip,
    Typography,
    useTheme,
} from "@mui/material";
import { SimpleTreeView } from "@mui/x-tree-view/SimpleTreeView";
import { TreeItem } from "@mui/x-tree-view/TreeItem";
import { useMemo } from "react";
import type { User } from "../../../store/sratApi";

interface UsersTreeViewProps {
    users?: User[];
    selectedUserKey?: string;
    onUserSelect: (userKey: string, user: User) => void;
    readOnly?: boolean;
    expandedItems: string[];
    onExpandedItemsChange: (items: string[]) => void;
}

export function UsersTreeView({
    users,
    selectedUserKey,
    onUserSelect,
    readOnly = false,
    expandedItems,
    onExpandedItemsChange,
}: UsersTreeViewProps) {
    const theme = useTheme();

    const groupedAndSortedUsers = useMemo(() => {
        if (!users || !Array.isArray(users)) {
            return [];
        }

        const adminUsers: User[] = [];
        const regularUsers: User[] = [];

        users.forEach((user) => {
            if (user.is_admin) {
                adminUsers.push(user);
            } else {
                regularUsers.push(user);
            }
        });

        // Sort users within each group by username
        adminUsers.sort((a, b) =>
            (a.username || "").localeCompare(b.username || ""),
        );
        regularUsers.sort((a, b) =>
            (a.username || "").localeCompare(b.username || ""),
        );

        const groups: Array<[string, User[]]> = [];
        if (adminUsers.length > 0) {
            groups.push(["admin", adminUsers]);
        }
        if (regularUsers.length > 0) {
            groups.push(["users", regularUsers]);
        }

        return groups;
    }, [users]);

    const renderUserItem = (user: User) => {
        const userKey = user.username || "";
        const isSelected = selectedUserKey === userKey;
        const userRwShares = user.rw_shares || [];
        const userRoShares = user.ro_shares || [];
        const totalShares = userRwShares.length + userRoShares.length;

        return (
            <TreeItem
                key={userKey}
                itemId={userKey}
                label={
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            py: 0.5,
                            px: 1,
                            backgroundColor: isSelected ? theme.palette.action.selected : "transparent",
                            borderRadius: 1,
                            "&:hover": {
                                backgroundColor: theme.palette.action.hover,
                            },
                        }}
                        onClick={(e) => {
                            e.stopPropagation();
                            onUserSelect(userKey, user);
                        }}
                    >
                        {user.is_admin ? (
                            <AdminPanelSettingsIcon color="warning" sx={{ mr: 1 }} />
                        ) : (
                            <AssignmentIndIcon sx={{ mr: 1 }} />
                        )}

                        <Box sx={{ flexGrow: 1, mr: 1 }}>
                            <Typography variant="body2" fontWeight={isSelected ? 600 : 400}>
                                {user.username}
                            </Typography>
                            <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5, mt: 0.5 }}>
                                {user.is_admin && (
                                    <Chip
                                        size="small"
                                        variant="outlined"
                                        color="warning"
                                        label="Admin"
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                                {totalShares > 0 && (
                                    <Chip
                                        size="small"
                                        variant="outlined"
                                        label={`${totalShares} share${totalShares > 1 ? "s" : ""}`}
                                        sx={{ fontSize: "0.7rem", height: 16 }}
                                    />
                                )}
                            </Box>
                        </Box>
                    </Box>
                }
            />
        );
    };

    const getGroupLabel = (group: string) => {
        switch (group) {
            case "admin":
                return "Administrators";
            case "users":
                return "Users";
            default:
                return group;
        }
    };

    const getGroupIcon = (group: string) => {
        switch (group) {
            case "admin":
                return <AdminPanelSettingsIcon color="warning" />;
            case "users":
                return <GroupIcon />;
            default:
                return <GroupIcon />;
        }
    };

    return (
        <SimpleTreeView
            aria-label="users tree view"
            expandedItems={expandedItems}
            onExpandedItemsChange={(_event, items) => onExpandedItemsChange(items)}
            sx={{
                flexGrow: 1,
                overflow: "auto",
            }}
        >
            {groupedAndSortedUsers.map(([group, userList]) => (
                <TreeItem
                    key={`group-${group}`}
                    itemId={`group-${group}`}
                    label={
                        <Box sx={{ display: "flex", alignItems: "center", py: 0.5 }}>
                            {getGroupIcon(group)}
                            <Typography variant="subtitle2" sx={{ ml: 1, fontWeight: 600 }}>
                                {getGroupLabel(group)}
                            </Typography>
                            <Chip
                                size="small"
                                label={userList.length}
                                sx={{ ml: 1, height: 20, fontSize: "0.75rem" }}
                            />
                        </Box>
                    }
                >
                    {userList.map((user) => renderUserItem(user))}
                </TreeItem>
            ))}
        </SimpleTreeView>
    );
}
