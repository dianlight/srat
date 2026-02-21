import CheckCircleOutlineIcon from "@mui/icons-material/CheckCircleOutline";
import ErrorOutlineIcon from "@mui/icons-material/ErrorOutline";
import HelpOutlineIcon from "@mui/icons-material/HelpOutline";
import { Box, Typography } from "@mui/material";
import { useMemo, type ReactNode } from "react";
import { useGetApiFilesystemStateQuery, type FilesystemState } from "../../../store/sratApi";

export interface UseFilesystemStateResult {
    filesystemState: FilesystemState | null;
    filesystemStateLoading: boolean;
    filesystemStateError: boolean;
    filesystemStatus: "clean" | "has_error" | "no_status";
    filesystemStatusIcon: ReactNode;
    filesystemStatusTooltip: ReactNode;
}

export function useFilesystemState(partitionId?: string): UseFilesystemStateResult {
    const {
        data: filesystemStateResponse,
        currentData: filesystemStateCurrentData,
        isLoading: filesystemStateLoading,
        isError: filesystemStateError,
    } = useGetApiFilesystemStateQuery(
        { partitionId },
        { skip: !partitionId },
    );

    const filesystemStatePayload = filesystemStateResponse ?? filesystemStateCurrentData;

    const filesystemState = useMemo<FilesystemState | null>(() => {
        if (!filesystemStatePayload) {
            return null;
        }
        if ("hasErrors" in filesystemStatePayload) {
            return filesystemStatePayload;
        }
        return null;
    }, [filesystemStatePayload]);

    const filesystemStatus = useMemo(() => {
        if (!filesystemState) {
            return "no_status" as const;
        }
        if (filesystemState.hasErrors) {
            return "has_error" as const;
        }
        if (filesystemState.isClean) {
            return "clean" as const;
        }
        return "no_status" as const;
    }, [filesystemState]);

    const filesystemStatusIcon = useMemo(() => {
        if (filesystemStatus === "clean") {
            return <CheckCircleOutlineIcon color="success" fontSize="small" />;
        }
        if (filesystemStatus === "has_error") {
            return <ErrorOutlineIcon color="error" fontSize="small" />;
        }
        return <HelpOutlineIcon color="disabled" fontSize="small" />;
    }, [filesystemStatus]);

    const filesystemStatusTooltip = useMemo(() => {
        if (filesystemStateLoading) {
            return "Loading filesystem status...";
        }
        if (!filesystemState) {
            return "No filesystem status available";
        }
        const description = filesystemState.stateDescription || "Filesystem status";
        const additionalInfoEntries = Object.entries(
            filesystemState.additionalInfo || {},
        );
        if (additionalInfoEntries.length === 0) {
            return description;
        }
        return (
            <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                    {description}
                </Typography>
                {additionalInfoEntries.map(([key, value]) => (
                    <Typography key={key} variant="body2">
                        {key}: {(typeof value === "string" ? value : JSON.stringify(value)).split("\n").map((line, index) => (
                            <span key={index}>
                                {line}
                                <br />
                            </span>
                        ))}
                    </Typography>
                ))}
            </Box>
        );
    }, [filesystemState, filesystemStateLoading]);

    return {
        filesystemState,
        filesystemStateLoading,
        filesystemStateError,
        filesystemStatus,
        filesystemStatusIcon,
        filesystemStatusTooltip,
    };
}
