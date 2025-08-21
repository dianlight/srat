import AdminPanelSettingsIcon from "@mui/icons-material/AdminPanelSettings";
import AssignmentIndIcon from "@mui/icons-material/AssignmentInd";
import EditIcon from "@mui/icons-material/Edit";
import PersonAddIcon from "@mui/icons-material/PersonAdd";
import VisibilityIcon from "@mui/icons-material/Visibility";
import {
	Avatar,
	Box,
	Chip,
	Divider,
	Fab,
	List,
	ListItemAvatar,
	ListItemButton,
	ListItemText,
	Stack,
	Tooltip,
	Typography,
	useTheme,
} from "@mui/material";
import { useConfirm } from "material-ui-confirm";
import { Fragment, useState } from "react";
import { InView } from "react-intersection-observer";
import { toast } from "react-toastify";
import { useReadOnly } from "../../hooks/readonlyHook";
import { TabIDs } from "../../store/locationState";
import {
	useDeleteApiUserByUsernameMutation,
	useGetApiUsersQuery,
	usePostApiUserMutation,
	usePutApiUseradminMutation,
	usePutApiUserByUsernameMutation,
} from "../../store/sratApi";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import { UserActions } from "./UserActions";
import { UserEditDialog } from "./UserEditDialog";
import type { UsersProps } from "./types";

export function Users() {
	const read_only = useReadOnly();
	const users = useGetApiUsersQuery();
	const [_errorInfo, setErrorInfo] = useState<string>("");
	const [selected, setSelected] = useState<UsersProps>({
		username: "",
		password: "",
	});
	const confirm = useConfirm();
	const [showEdit, setShowEdit] = useState<boolean>(false);
	const [userCreate] = usePostApiUserMutation();
	const [userAdminUpdate] = usePutApiUseradminMutation();
	const [userUpdate] = usePutApiUserByUsernameMutation();
	const [userDelete] = useDeleteApiUserByUsernameMutation();

	function onSubmitEditUser(data?: UsersProps) {
		if (!data || !data.username || !data.password) {
			console.log("Data is invalid", data);
			setErrorInfo("Unable to update user!");
			return;
		}

		data.username = data.username.trim();
		data.password = data.password.trim();

		// Save Data
		console.log(data);
		if (data.doCreate) {
			userCreate({ user: data })
				.unwrap()
				.then((_res) => {
					setErrorInfo("");
					setSelected({ username: "", password: "" });
					users.refetch();
				})
				.catch((err) => {
					setErrorInfo(JSON.stringify(err));
					console.error(err);
					toast.error(`Error userCreate ${data.username}`, {
						data: { error: err.data },
					});
				});
			return;
		} else if (data.is_admin) {
			userAdminUpdate({ user: data })
				.unwrap()
				.then((_res) => {
					setErrorInfo("");
					users.refetch();
				})
				.catch((err) => {
					setErrorInfo(JSON.stringify(err));
					toast.error(`Error userAdminUpdate ${data.username}`, {
						data: { error: err.data },
					});
					console.error(err);
				});
		} else {
			userUpdate({ username: data.username, user: data })
				.unwrap()
				.then((_res) => {
					setErrorInfo("");
					setSelected({ username: "", password: "" });
					users.refetch();
				})
				.catch((err) => {
					setErrorInfo(JSON.stringify(err));
					toast.error(`Error userUpdate ${data.username}`, {
						data: { error: err.data },
					});
					console.error(err);
				});
		}
	}

	function onSubmitDeleteUser(data: UsersProps) {
		console.log("Delete", data);
		if (!data) return;

		confirm({
			title: `Delete ${data.username}?`,
			description: "Do you really would delete this user?",
			acknowledgement:
				"I understand that deleting the share will remove it permanently.",
		}).then(({ confirmed, reason }) => {
			if (confirmed) {
				if (!data.username) {
					setErrorInfo("Unable to delete user!");
					return;
				}
				userDelete({ username: data.username })
					.unwrap()
					.then((_res) => {
						setErrorInfo("");
						setSelected({ username: "", password: "" });
						users.refetch();
					})
					.catch((err) => {
						setErrorInfo(JSON.stringify(err));
					});
			} else if (reason === "cancel") {
				console.log("cancel");
			}
		});
	}

	TourEvents.on(TourEventTypes.USERS_STEP_3, (user) => {
		setSelected(user);
		setShowEdit(true);
	});

	return (
		<InView>
			<UserEditDialog
				objectToEdit={selected}
				open={showEdit}
				onClose={(data) => {
					setSelected({ username: "", password: "", doCreate: false });
					onSubmitEditUser(data);
					setShowEdit(false);
				}}
			/>
			<br />
			<Stack
				direction="row"
				justifyContent="flex-end"
				sx={{ px: 2, mb: 1, alignItems: "center" }}
				data-tutor={`reactour__tab${TabIDs.USERS}__step0`}
			>
				{read_only || (
					<Fab
						color="primary"
						aria-label="add"
						// sx removed: float, top, margin - FAB is now in normal flow within Stack
						size="small"
						onClick={() => {
							setSelected({ username: "", password: "", doCreate: true });
							setShowEdit(true);
						}}
					>
						<PersonAddIcon data-tutor={`reactour__tab${TabIDs.USERS}__step2`} />
					</Fab>
				)}
			</Stack>
			<List
				dense={true}
				component="span"
				data-tutor={`reactour__tab${TabIDs.USERS}__step1`}
			>
				<Divider />
				{users.isSuccess &&
					Array.isArray(users.data) &&
					users.data
						.slice()
						.sort((a, b) => {
							// Sort admin users to the top, then alphabetically by username
							if (a.is_admin && !b.is_admin) return -1;
							if (!a.is_admin && b.is_admin) return 1;
							return (a.username || "").localeCompare(b.username || "");
						})
						.map((user) => {
							const userRwShares = user.rw_shares || [];
							const userRoShares = user.ro_shares || [];

							return (
								<Fragment key={user.username || "admin"}>
									<ListItemButton
										sx={[
											(theme) => ({
												backgroundColor: theme.vars?.palette.background.default,
											}),
											(theme) =>
												theme.applyStyles('dark', {
													backgroundColor: theme.vars?.palette.grey[900],
												}),
										]//{
											//											alignItems: "flex-start",
											//											bgcolor: theme.vars?.palette.background.paper,
											//										}
										}

									>
										<ListItemAvatar sx={{ pt: 1 }}>
											<Avatar
												data-tutor={`reactour__tab${TabIDs.USERS}__step5`}
											>
												{user.is_admin ? (
													<AdminPanelSettingsIcon />
												) : (
													<AssignmentIndIcon />
												)}
											</Avatar>
										</ListItemAvatar>
										<ListItemText
											sx={{ flexGrow: 1, overflowWrap: "break-word" }}
											primary={user.username}
											slotProps={{
												secondary: {
													component: "span",
												},
											}}
											secondary={
												<Stack
													direction="row"
													spacing={1}
													flexWrap="wrap"
													alignItems="center"
													sx={{ mt: 0.5, display: { xs: "none", sm: "flex" } }}
												>
													{userRwShares.length > 0 && (
														<Tooltip
															title={`Shares with read-write access for ${user.username}`}
														>
															<Chip
																icon={<EditIcon fontSize="small" />}
																label={
																	<Box
																		component="span"
																		sx={{
																			display: "flex",
																			alignItems: "center",
																			flexWrap: "wrap",
																			gap: 0.5,
																		}}
																	>
																		Shares:
																		{userRwShares.map((share, index) => (
																			<Typography
																				component="span"
																				variant="caption"
																				key={share}
																			>
																				{share}
																				{index < userRwShares.length - 1
																					? ","
																					: ""}
																			</Typography>
																		))}
																	</Box>
																}
																size="small"
																variant="outlined"
																onClick={(e) => {
																	e.stopPropagation(); /* Optionally handle click, e.g., navigate to shares page */
																}}
															/>
														</Tooltip>
													)}
													{userRoShares.length > 0 && (
														<Tooltip
															title={`Shares with read-only access for ${user.username}`}
														>
															<Chip
																icon={<VisibilityIcon fontSize="small" />}
																label={
																	<Box
																		component="span"
																		sx={{
																			display: "flex",
																			alignItems: "center",
																			flexWrap: "wrap",
																			gap: 0.5,
																		}}
																	>
																		Shares:
																		{userRoShares.map((share, index) => (
																			<Typography
																				component="span"
																				variant="caption"
																				key={share}
																			>
																				{share}
																				{index < userRoShares.length - 1
																					? ","
																					: ""}
																			</Typography>
																		))}
																	</Box>
																}
																size="small"
																variant="outlined"
																onClick={(e) => {
																	e.stopPropagation(); /* Optionally handle click */
																}}
															/>
														</Tooltip>
													)}
													{userRwShares.length === 0 &&
														userRoShares.length === 0 && (
															<Typography
																variant="caption"
																sx={{ fontStyle: "italic" }}
															>
																No shares assigned
															</Typography>
														)}
												</Stack>
											}
										/>
										<UserActions
											user={user}
											read_only={read_only || false}
											onEdit={(user) => {
												setSelected(user);
												setShowEdit(true);
											}}
											onDelete={onSubmitDeleteUser}
										/>
									</ListItemButton>
									<Divider component="li" />
								</Fragment>
							);
						})}
			</List>
		</InView>
	);
}