import {
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogContentText,
	DialogTitle,
	Grid,
	Stack,
} from "@mui/material";
import { Fragment } from "react";
import { useForm } from "react-hook-form";
import {
	PasswordElement,
	PasswordRepeatElement,
	TextFieldElement,
} from "react-hook-form-mui";
import { TabIDs } from "../../store/locationState";
import type { UsersProps } from "./types";

export function UserEditDialog(props: {
	open: boolean;
	onClose: (data?: UsersProps) => void;
	objectToEdit?: UsersProps;
}) {
	const {
		control,
		handleSubmit,
		watch,
		formState: { errors },
	} = useForm<UsersProps>({
		defaultValues: {
			username: "",
			password: "",
			is_admin: false,
		},
		values: props.objectToEdit?.doCreate
			? {
				username: "",
				password: "",
				is_admin: false,
				doCreate: true,
			}
			: props.objectToEdit,
	});

	function handleCloseSubmit(data?: UsersProps) {
		props.onClose(data);
	}

	return (
		<Fragment>
			<Dialog open={props.open} onClose={() => handleCloseSubmit()}>
				<DialogTitle>
					{props.objectToEdit?.is_admin
						? "Administrator"
						: props.objectToEdit?.username || "New User"}
				</DialogTitle>
				<DialogContent>
					<Stack spacing={2}>
						<DialogContentText>
							Please enter the username and password for the user.
						</DialogContentText>
						<form
							id="editshareform"
							onSubmit={handleSubmit(handleCloseSubmit)}
							noValidate
						>
							<Grid container spacing={2}>
								<Grid size={6}>
									<TextFieldElement
										size="small"
										name="username"
										autoComplete="username"
										label="User Name"
										required
										control={control}
										slotProps={
											props.objectToEdit?.username
												? props.objectToEdit.is_admin
													? {}
													: {
														input: {
															readOnly: true,
														},
													}
												: {}
										}
									/>
								</Grid>
								<Grid size={6}>
									<PasswordElement
										size="small"
										autoComplete="new-password"
										name="password"
										label="Password"
										required
										control={control}
									/>
									<PasswordRepeatElement
										size="small"
										autoComplete="new-password"
										passwordFieldName={"password"}
										name={"password-repeat"}
										margin={"dense"}
										label={"Repeat Password"}
										required
										control={control}
									/>
								</Grid>
							</Grid>
						</form>
					</Stack>
				</DialogContent>
				<DialogActions>
					<Button onClick={() => handleCloseSubmit()} variant="outlined" color="secondary">Cancel</Button>
					<Button
						type="submit"
						form="editshareform"
						data-tutor={`reactour__tab${TabIDs.USERS}__step6`}
						variant="outlined"
						color="success"
					>
						{props.objectToEdit?.doCreate ? "Create" : "Apply"}
					</Button>
				</DialogActions>
			</Dialog>
		</Fragment>
	);
}
