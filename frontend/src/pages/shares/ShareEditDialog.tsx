import {
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogContentText,
	DialogTitle,
} from "@mui/material";
import { Fragment } from "react";
import type { ShareEditProps } from "./types";
import { ShareEditForm } from "./components/ShareEditForm";
import type { SharedResource } from "../../store/sratApi";

interface ShareEditDialogProps {
	open: boolean;
	onClose: (data?: ShareEditProps) => void;
	objectToEdit?: ShareEditProps;
	shares?: SharedResource[]; // Added to receive shares data
	onDeleteSubmit?: (shareName: string, shareData: SharedResource) => void; // Added for delete action
	onToggleEnabled?: (enabled: boolean) => void; // Added for enable/disable toggle
}

export function ShareEditDialog(props: ShareEditDialogProps) {
	function handleCloseSubmit(data?: ShareEditProps) {
		if (!data) {
			props.onClose();
			return;
		}
		console.log(data);
		props.onClose(data);
	}

	const handleDelete = (shareName: string, shareData: SharedResource) => {
		if (props.onDeleteSubmit) {
			props.onDeleteSubmit(shareName, shareData);
		}
		props.onClose(); // Close the dialog after delete
	};

	return (
		<Fragment>
			<Dialog
				open={props.open}
				onClose={(_event, reason) => {
					if (reason && reason === "backdropClick") {
						return; // Prevent dialog from closing on backdrop click
					}
					handleCloseSubmit(); // Proceed with closing for other reasons (e.g., explicit button calls)
				}}
				maxWidth="md"
				fullWidth
			>
				<DialogTitle>
					{props.objectToEdit?.org_name === undefined ? "Create New Share" : "Edit Share"}
				</DialogTitle>
				<DialogContent>
					<DialogContentText sx={{ mb: 2 }}>
						Please enter or modify share properties.
					</DialogContentText>
					<ShareEditForm
						shareData={props.objectToEdit}
						shares={props.shares}
						onSubmit={handleCloseSubmit}
						onDelete={handleDelete}
						showActions={false}
						variant="plain"
					/>
				</DialogContent>
				<DialogActions>
					{props.objectToEdit?.org_name && props.onDeleteSubmit && (
						<Button
							onClick={() => {
								// Ensure objectToEdit and org_name are valid before calling onDeleteSubmit
								if (props.objectToEdit?.org_name && props.onDeleteSubmit) {
									props.onDeleteSubmit(
										props.objectToEdit.org_name,
										props.objectToEdit,
									);
								}
								handleCloseSubmit(); // Close the dialog
							}}
							color="error"
							variant="outlined"
						>
							Delete
						</Button>
					)}
					<Button onClick={() => handleCloseSubmit()} variant="outlined" color="secondary">Cancel</Button>
					<Button
						type="submit"
						form="editshareform"
						variant="outlined"
						color="success"
					>
						{props.objectToEdit?.org_name === undefined ? "Create" : "Apply"}
					</Button>
				</DialogActions>
			</Dialog>
		</Fragment>
	);
}
