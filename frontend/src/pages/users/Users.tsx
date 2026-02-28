import AddIcon from "@mui/icons-material/Add";
import {
	Box,
	Grid,
	IconButton,
	Paper,
	Stack,
	Tooltip,
	Typography,
} from "@mui/material";
import { useConfirm } from "material-ui-confirm";
import { useEffect, useState } from "react";
import { InView } from "react-intersection-observer";
import { toast } from "react-toastify";
import { TabIDs } from "../../store/locationState";
import {
	type User,
	useDeleteApiUserByUsernameMutation,
	useGetApiUsersQuery,
	usePostApiUserMutation,
	usePutApiUseradminMutation,
	usePutApiUserByUsernameMutation,
} from "../../store/sratApi";
import { useGetServerEventsQuery } from "../../store/sseApi";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import { UserEditDialog } from "./UserEditDialog";
import { UserDetailsPanel, UserEditForm, UsersTreeView } from "./components";
import type { UsersProps } from "./types";

export function Users() {
	const { data: evdata } = useGetServerEventsQuery();
	const users = useGetApiUsersQuery();
	const confirm = useConfirm();

	// Selection state
	const [selectedUserKey, setSelectedUserKey] = useState<string | undefined>(() =>
		localStorage.getItem("users.selectedUserKey") || undefined
	);
	const [selectedUser, setSelectedUser] = useState<User | null>(null);
	const [expandedGroups, setExpandedGroups] = useState<string[]>(() => {
		try {
			const savedExpanded = localStorage.getItem("users.expandedGroups");
			if (savedExpanded) {
				const parsed = JSON.parse(savedExpanded);
				if (Array.isArray(parsed)) return parsed as string[];
			}
		} catch { }
		return ["group-admin", "group-users"];
	});

	// Edit/Create state
	const [showEdit, setShowEdit] = useState<boolean>(false);
	const [showCreateDialog, setShowCreateDialog] = useState<boolean>(false);
	const [createUserData, setCreateUserData] = useState<UsersProps>({
		username: "",
		password: "",
		doCreate: true,
	});

	// API mutations
	const [userCreate] = usePostApiUserMutation();
	const [userAdminUpdate] = usePutApiUseradminMutation();
	const [userUpdate] = usePutApiUserByUsernameMutation();
	const [userDelete] = useDeleteApiUserByUsernameMutation();

	// Persist selection and expanded groups to localStorage
	useEffect(() => {
		try {
			if (selectedUserKey) {
				localStorage.setItem("users.selectedUserKey", selectedUserKey);
			} else {
				localStorage.removeItem("users.selectedUserKey");
			}
		} catch (err) {
			console.warn("Could not persist selectedUserKey", err);
		}
	}, [selectedUserKey]);

	useEffect(() => {
		try {
			if (expandedGroups.length > 0) {
				localStorage.setItem("users.expandedGroups", JSON.stringify(expandedGroups));
			} else {
				localStorage.removeItem("users.expandedGroups");
			}
		} catch (err) {
			console.warn("Could not persist expandedGroups", err);
		}
	}, [expandedGroups]);

	// When users data is available and there's a selectedUserKey, find and select it
	useEffect(() => {
		if (!users.data || !Array.isArray(users.data) || users.data.length === 0) return;
		if (!selectedUserKey) return;

		const foundUser = users.data.find((user) => user.username === selectedUserKey);

		if (foundUser) {
			setSelectedUser(foundUser);
			// Ensure the containing group is expanded
			const groupId = foundUser.is_admin ? "group-admin" : "group-users";
			setExpandedGroups((prev) => {
				if (prev.includes(groupId)) return prev;
				return [...prev, groupId];
			});
		} else {
			setSelectedUser(null);
			setSelectedUserKey(undefined);
		}
	}, [users.data, selectedUserKey]);

	const handleUserSelect = (userKey: string, user: User) => {
		setSelectedUserKey(userKey);
		setSelectedUser(user);
		setShowEdit(false);

		// Ensure the containing group is expanded
		const groupId = user.is_admin ? "group-admin" : "group-users";
		setExpandedGroups((prev) => {
			if (prev.includes(groupId)) return prev;
			return [...prev, groupId];
		});
	};

	function onSubmitEditUser(data?: UsersProps) {
		if (!data || !data.username || (!data.password && data.doCreate)) {
			console.log("Data is invalid", data);
			return;
		}

		data.username = data.username.toLocaleLowerCase().trim();
		data.password = data?.password?.trim();

		if (data.doCreate) {
			userCreate({ user: data })
				.unwrap()
				.then((_res) => {
					setShowCreateDialog(false);
					setCreateUserData({ username: "", password: "", doCreate: true });
					users.refetch();
					toast.success(`User ${data.username} created successfully`);
				})
				.catch((err) => {
					console.error(err);
					toast.error(`Error creating user ${data.username}`, {
						data: { error: err.data },
					});
				});
		} else if (data.is_admin) {
			userAdminUpdate({ user: data })
				.unwrap()
				.then((_res) => {
					users.refetch();
					setShowEdit(false);
					toast.success(`Admin ${data.username} updated successfully`);
				})
				.catch((err) => {
					toast.error(`Error updating admin ${data.username}`, {
						data: { error: err.data },
					});
					console.error(err);
				});
		} else {
			userUpdate({ username: data.username, user: data })
				.unwrap()
				.then((_res) => {
					users.refetch();
					setShowEdit(false);
					toast.success(`User ${data.username} updated successfully`);
				})
				.catch((err) => {
					toast.error(`Error updating user ${data.username}`, {
						data: { error: err.data },
					});
					console.error(err);
				});
		}
	}

	function onSubmitDeleteUser(user: User) {
		if (!user) return;

		confirm({
			title: `Delete ${user.username}?`,
			description: "Do you really want to delete this user?",
			acknowledgement:
				"I understand that deleting the user will remove it permanently.",
		}).then(({ confirmed }) => {
			if (confirmed) {
				if (!user.username) {
					toast.error("Unable to delete user!");
					return;
				}
				userDelete({ username: user.username })
					.unwrap()
					.then((_res) => {
						// Clear selection if deleted user was selected
						if (selectedUserKey === user.username) {
							setSelectedUserKey(undefined);
							setSelectedUser(null);
							setShowEdit(false);
						}
						users.refetch();
						toast.success(`User ${user.username} deleted successfully`);
					})
					.catch((err) => {
						toast.error(`Error deleting user ${user.username}`, {
							data: { error: err.data },
						});
					});
			}
		});
	}

	// Tour event handler
	TourEvents.on(TourEventTypes.USERS_STEP_3, () => {
		setCreateUserData({ username: "", password: "", doCreate: true });
		setShowCreateDialog(true);
	});

	const isReadOnly = evdata?.hello?.read_only || false;

	return (
		<InView>
			{/* Create User Dialog */}
			<UserEditDialog
				objectToEdit={createUserData}
				open={showCreateDialog}
				onClose={(data) => {
					if (data) {
						onSubmitEditUser(data);
					} else {
						setShowCreateDialog(false);
						setCreateUserData({ username: "", password: "", doCreate: true });
					}
				}}
			/>

			{/* Main Layout Grid */}
			<Grid container spacing={2} sx={{ minHeight: "calc(100vh - 200px)", mt: 1 }} data-tutor={`reactour__tab${TabIDs.USERS}__step0`}>
				{/* Left Panel - Tree View */}
				<Grid size={{ xs: 12, md: 4, lg: 3 }}>
					<Paper sx={{ height: "100%", p: 1 }} data-tutor={`reactour__tab${TabIDs.USERS}__step1`}>
						<Stack
							direction="row"
							justifyContent="space-between"
							alignItems="center"
							sx={{ mb: 2, px: 2 }}
						>
							<Typography variant="h6">
								Users
							</Typography>
							{!isReadOnly && (
								<Tooltip title="Create new user" data-tutor={`reactour__tab${TabIDs.USERS}__step2`}>
									<IconButton
										id="create_new_user"
										color="primary"
										aria-label="Create new user"
										size="small"
										onClick={() => {
											setCreateUserData({ username: "", password: "", doCreate: true });
											setShowCreateDialog(true);
										}}
									>
										<AddIcon />
									</IconButton>
								</Tooltip>
							)}
						</Stack>
						<UsersTreeView
							users={Array.isArray(users.data) ? users.data : undefined}
							selectedUserKey={selectedUserKey}
							onUserSelect={handleUserSelect}
							readOnly={isReadOnly}
							expandedItems={expandedGroups}
							onExpandedItemsChange={setExpandedGroups}
						/>
					</Paper>
				</Grid>

				{/* Right Panel - Details and Edit Form */}
				<Grid size={{ xs: 12, md: 8, lg: 9 }}>
					<Paper sx={{ height: "100%", overflow: "hidden" }}>
						{selectedUser && selectedUserKey ? (
							<UserDetailsPanel
								user={selectedUser}
								userKey={selectedUserKey}
								onEdit={onSubmitEditUser}
								onDelete={onSubmitDeleteUser}
								onEditClick={() => setShowEdit(true)}
								onCancelEdit={() => setShowEdit(false)}
								isEditing={showEdit}
								readOnly={isReadOnly}
							>
								{/* Embedded Edit Form */}
								<UserEditForm
									userData={{
										...selectedUser,
										password: "",
										doCreate: false,
									}}
									onSubmit={(data) => {
										onSubmitEditUser(data);
									}}
									onCancel={() => setShowEdit(false)}
									disabled={isReadOnly}
								/>
							</UserDetailsPanel>
						) : (
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
									Select a user from the list to view details
								</Typography>
							</Box>
						)}
					</Paper>
				</Grid>
			</Grid>
		</InView>
	);
} 