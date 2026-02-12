import {
    Dialog,
    DialogTitle
} from "@mui/material";
import { Fragment } from "react";
import { UserEditForm } from "./components/UserEditForm";
import type { UsersProps } from "./types";

export function UserEditDialog(props: {
	open: boolean;
	onClose: (data?: UsersProps) => void;
	objectToEdit?: UsersProps;
}) {
	function handleCloseSubmit(data: UsersProps) {
		props.onClose(data);
	}

	function handleCancel() {
		props.onClose();
	}

	return (
		<Fragment>
			<Dialog open={props.open} onClose={handleCancel} maxWidth="sm" fullWidth>
				<DialogTitle>
					{props.objectToEdit?.is_admin
						? "Administrator"
						: props.objectToEdit?.username || "New User"}
				</DialogTitle>
				<UserEditForm
					userData={props.objectToEdit}
					onSubmit={handleCloseSubmit}
					onCancel={handleCancel}
				/>
				<div style={{ display: "none" }} id="editshareform" />
			</Dialog>
		</Fragment>
	);
}
