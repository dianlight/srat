import { useEffect, useState } from "react";
import {
	type Settings,
	type User,
	useGetApiSettingsQuery,
	useGetApiUsersQuery,
} from "../store/sratApi";

/**
 * Hook to determine if the base config modal should be shown
 * Shows the modal if:
 * 1. The admin user's password is still at the default "changeme!"
 * 2. Hostname or workgroup are not set (indicating first-time setup)
 * @returns {object} shouldShow boolean and dismiss function
 */
export const useBaseConfigModal = () => {
	const [shouldShow, setShouldShow] = useState(false);
	const [hasChecked, setHasChecked] = useState(false);

	const { data: settings, isLoading: isSettingsLoading } =
		useGetApiSettingsQuery();
	const { data: users, isLoading: isUsersLoading } = useGetApiUsersQuery();

	useEffect(() => {
		// Don't check until both settings and users are loaded
		if (isSettingsLoading || isUsersLoading || hasChecked) {
			return;
		}

		// Type guard to ensure settings is a Settings object and not an error
		const isValidSettings = (data: unknown): data is Settings => {
			return data !== null && typeof data === "object" && "hostname" in data;
		};

		// Type guard to ensure users is an array and not an error
		const isValidUsers = (data: unknown): data is User[] => {
			return Array.isArray(data);
		};

		// Find the admin user
		const adminUser = isValidUsers(users)
			? users.find((u) => u.is_admin)
			: null;

		// Only show modal if:
		// 1. Settings are loaded and valid
		// 2. Users are loaded and admin user exists with default password
		// 3. Hostname or workgroup are not set (indicating first-time setup)
		if (
			isValidSettings(settings) &&
			adminUser &&
			adminUser.password === "changeme!"
		) {
			setShouldShow(true);
		}

		setHasChecked(true);
	}, [settings, users, isSettingsLoading, isUsersLoading, hasChecked]);

	const dismiss = () => {
		setShouldShow(false);
		// Store in localStorage to remember this choice
		localStorage.setItem("baseConfigModalDismissed", "true");
	};

	return { shouldShow, dismiss };
};
