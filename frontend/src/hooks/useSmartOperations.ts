import { useCallback } from "react";
import { toast } from "react-toastify";
import type { ErrorModel } from "../store/sratApi";
import {
	usePostApiDiskByDiskIdSmartDisableMutation,
	usePostApiDiskByDiskIdSmartEnableMutation,
	usePostApiDiskByDiskIdSmartTestAbortMutation,
	usePostApiDiskByDiskIdSmartTestStartMutation,
} from "../store/sratApi";

// Smart test types
export type SmartTestType = "short" | "long" | "conveyance";

export function useSmartOperations(diskId?: string) {
	const [startTest, { isLoading: isStarting }] =
		usePostApiDiskByDiskIdSmartTestStartMutation();
	const [abortTest, { isLoading: isAborting }] =
		usePostApiDiskByDiskIdSmartTestAbortMutation();
	const [enableSMARTApi, { isLoading: isEnabling }] =
		usePostApiDiskByDiskIdSmartEnableMutation();
	const [disableSMARTApi, { isLoading: isDisabling }] =
		usePostApiDiskByDiskIdSmartDisableMutation();

	const isLoading = isStarting || isAborting || isEnabling || isDisabling;

	const startSelfTest = useCallback(
		async (testType: SmartTestType) => {
			if (!diskId) {
				toast.error("Disk ID is required");
				return;
			}

			try {
				await startTest({
					diskId,
					postDiskByDiskIdSmartTestStartRequest: { test_type: testType },
				}).unwrap();
				toast.success(`Starting SMART ${testType} test...`);
			} catch (error) {
				const errorMessage =
					error && typeof error === "object" && "data" in error
						? (error.data as ErrorModel)?.detail || "Failed to start self-test"
						: "Failed to start self-test";
				console.error("Failed to start self-test:", error);
				toast.error(errorMessage);
			}
		},
		[diskId, startTest],
	);

	const abortSelfTest = useCallback(async () => {
		if (!diskId) {
			toast.error("Disk ID is required");
			return;
		}

		try {
			await abortTest({ diskId }).unwrap();
			toast.success("SMART test aborted");
		} catch (error) {
			const errorMessage =
				error && typeof error === "object" && "data" in error
					? (error.data as ErrorModel)?.detail || "Failed to abort self-test"
					: "Failed to abort self-test";
			console.error("Failed to abort test:", error);
			toast.error(errorMessage);
		}
	}, [diskId, abortTest]);

	const enableSmart = useCallback(async () => {
		if (!diskId) {
			toast.error("Disk ID is required");
			return;
		}

		try {
			await enableSMARTApi({ diskId }).unwrap();
			toast.success("SMART enabled");
		} catch (error) {
			const errorMessage =
				error && typeof error === "object" && "data" in error
					? (error.data as ErrorModel)?.detail || "Failed to enable SMART"
					: "Failed to enable SMART";
			console.error("Failed to enable SMART:", error);
			toast.error(errorMessage);
		}
	}, [diskId, enableSMARTApi]);

	const disableSmart = useCallback(async () => {
		if (!diskId) {
			toast.error("Disk ID is required");
			return;
		}

		try {
			await disableSMARTApi({ diskId }).unwrap();
			toast.success("SMART disabled");
		} catch (error) {
			const errorMessage =
				error && typeof error === "object" && "data" in error
					? (error.data as ErrorModel)?.detail || "Failed to disable SMART"
					: "Failed to disable SMART";
			console.error("Failed to disable SMART:", error);
			toast.error(errorMessage);
		}
	}, [diskId, disableSMARTApi]);

	return {
		startSelfTest,
		abortSelfTest,
		enableSmart,
		disableSmart,
		isLoading,
	};
}
