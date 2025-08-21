import ManageAccountsIcon from "@mui/icons-material/ManageAccounts";
import PersonRemoveIcon from "@mui/icons-material/PersonRemove";
import { IconButton, Stack } from "@mui/material";
import { TabIDs } from "../../store/locationState";
import type { User } from "../../store/sratApi";

interface UserActionsProps {
	user: User;
	read_only: boolean;
	onEdit: (user: User) => void;
	onDelete: (user: User) => void;
}

export function UserActions({ user, read_only, onEdit, onDelete }: UserActionsProps) {
	return (
		!read_only && (
			<Stack direction="column" spacing={0} sx={{ pl: 1 }}>
				<IconButton
					onClick={() => onEdit(user)}
					edge="end"
					aria-label="settings"
					size="small"
					data-tutor={`reactour__tab${TabIDs.USERS}__step3`}
				>
					<ManageAccountsIcon />
				</IconButton>
				{!user.is_admin && (
					<IconButton
						onClick={() => onDelete(user)}
						edge="end"
						aria-label="delete"
						size="small"
						data-tutor={`reactour__tab${TabIDs.USERS}__step4`}
					>
						<PersonRemoveIcon />
					</IconButton>
				)}
			</Stack>
		)
	);
}
