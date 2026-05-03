import { useEffect, useState } from "react";
import {
  type Settings,
  Telemetry_mode,
  type User,
  useGetApiSettingsQuery,
  useGetApiTelemetryInternetConnectionQuery,
  useGetApiUsersQuery,
} from "../store/sratApi";

/**
 * Hook to determine if the setup wizard should be shown.
 * Replaces both useBaseConfigModal and useTelemetryModal:
 * shows wizard if admin password is default, hostname/workgroup are unset,
 * or telemetry preference has not been chosen yet.
 */
export const useSetupWizard = () => {
  const [shouldShow, setShouldShow] = useState(false);

  const { data: settings, isLoading: isSettingsLoading } =
    useGetApiSettingsQuery();
  const { data: users, isLoading: isUsersLoading } = useGetApiUsersQuery();
  const { data: internetConnection, isLoading: isInternetLoading } =
    useGetApiTelemetryInternetConnectionQuery();

  useEffect(() => {
    if (isSettingsLoading || isUsersLoading || isInternetLoading) {
      return;
    }

    const isValidSettings = (data: unknown): data is Settings =>
      data !== null && typeof data === "object";

    const isValidUsers = (data: unknown): data is User[] =>
      Array.isArray(data) &&
      data.every((u) => typeof u === "object" && u !== null && "password" in u);

    const adminUser = isValidUsers(users)
      ? users.find((u) => u.is_admin)
      : null;

    const needsBaseConfig =
      isValidSettings(settings) &&
      (!adminUser?.password || !settings.hostname || !settings.workgroup);

    const needsTelemetry =
      isValidSettings(settings) &&
      settings.telemetry_mode === Telemetry_mode.Ask &&
      internetConnection === true;

    if (needsBaseConfig || needsTelemetry) {
      setShouldShow(true);
    }
  }, [
    settings,
    users,
    internetConnection,
    isSettingsLoading,
    isUsersLoading,
    isInternetLoading,
  ]);

  const dismiss = () => {
    setShouldShow(false);
  };

  return { shouldShow, dismiss };
};
