import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    FormControlLabel,
    LinearProgress,
    Stack,
    Switch,
    TextField,
    Typography,
} from "@mui/material";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "react-toastify";
import {
    type FilesystemTask,
    type Partition,
    usePostApiFilesystemCheckAbortMutation,
    usePostApiFilesystemCheckMutation,
} from "../../../store/sratApi";
import { useGetServerEventsQuery } from "../../../store/sseApi";
import { decodeEscapeSequence } from "../utils";

interface FilesystemCheckDialogProps {
    open: boolean;
    partition?: Partition;
    onClose: () => void;
    taskOverride?: FilesystemTask | null;
    initialVerbose?: boolean;
}

const isRunningStatus = (status?: string) => status === "start" || status === "running";

const matchesPartitionDevice = (task: FilesystemTask, partition?: Partition) => {
    if (!partition) return false;
    const candidates = new Set<string>();
    if (partition.device_path) candidates.add(partition.device_path);
    if (partition.legacy_device_path) candidates.add(partition.legacy_device_path);
    return candidates.has(task.device ?? "");
};

export function FilesystemCheckDialog({ open, partition, onClose, taskOverride, initialVerbose }: FilesystemCheckDialogProps) {
    const { data: eventData } = useGetServerEventsQuery();
    const [startCheck, { isLoading: isStarting }] = usePostApiFilesystemCheckMutation();
    const [abortCheck, { isLoading: isStopping }] = usePostApiFilesystemCheckAbortMutation();

    const [autoFix, setAutoFix] = useState(false);
    const [force, setForce] = useState(false);
    const [verbose, setVerbose] = useState(Boolean(initialVerbose));
    const [logs, setLogs] = useState<string[]>([]);
    const [progress, setProgress] = useState<number>(0);
    const [status, setStatus] = useState<string>("idle");
    const [message, setMessage] = useState<string>("");

    const lastNotesRef = useRef<string>("");

    const task = useMemo<FilesystemTask | null>(() => {
        if (taskOverride) {
            if (taskOverride.operation !== "check") return null;
            return matchesPartitionDevice(taskOverride, partition) ? taskOverride : null;
        }
        const candidate = eventData?.filesystem_task;
        if (!candidate || candidate.operation !== "check") return null;
        return matchesPartitionDevice(candidate, partition) ? candidate : null;
    }, [eventData?.filesystem_task, partition, taskOverride]);

    const isRunning = useMemo(() => isRunningStatus(task?.status) || isRunningStatus(status), [task?.status, status]);

    useEffect(() => {
        if (!open) return;
        if (!partition) return;
        setLogs([]);
        setProgress(0);
        setStatus("idle");
        setMessage("");
        setVerbose(Boolean(initialVerbose));
        lastNotesRef.current = "";
    }, [open, partition?.id, initialVerbose]);

    useEffect(() => {
        if (!open || !task) return;
        if (task.status) {
            setStatus(task.status);
        }
        if (task.message) {
            setMessage(task.message);
        }
        if (typeof task.progress === "number") {
            setProgress(task.progress);
        }
        const taskNotes = task.notes ?? [];
        if (taskNotes.length > 0) {
            const notesSignature = taskNotes.join("\n");
            if (notesSignature !== lastNotesRef.current) {
                lastNotesRef.current = notesSignature;
                setLogs((prev) => [...prev, ...taskNotes]);
            }
        }
    }, [open, task]);

    const handleStart = async () => {
        if (!partition?.id) {
            toast.error("Partition not selected.");
            return;
        }
        setLogs([]);
        setProgress(0);
        setStatus("start");
        setMessage("Starting filesystem check...");
        try {
            await startCheck({
                checkPartitionInput: {
                    partitionId: partition.id,
                    autoFix,
                    force,
                    verbose,
                },
            }).unwrap();
            toast.info("Filesystem check started.");
        } catch (err: any) {
            const errorMsg = err?.data?.detail || err?.data?.message || err?.message || "Failed to start filesystem check";
            toast.error(errorMsg);
            setStatus("failure");
        }
    };

    const handleStop = async () => {
        if (!partition?.id) {
            toast.error("Partition not selected.");
            return;
        }
        try {
            await abortCheck({
                abortCheckPartitionInput: {
                    partitionId: partition.id,
                },
            }).unwrap();
            toast.info("Filesystem check abort requested.");
        } catch (err: any) {
            const errorMsg = err?.data?.detail || err?.data?.message || err?.message || "Failed to abort filesystem check";
            toast.error(errorMsg);
        }
    };

    const progressValue = typeof task?.progress === "number" ? task.progress : progress;
    const showIndeterminate = progressValue === 999 || progressValue <= 0;
    const partitionLabel = decodeEscapeSequence(partition?.name || partition?.id || "Selected partition");

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>Filesystem Check: {partitionLabel}</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{ pt: 1 }}>
                    <DialogContentText>
                        Run a filesystem consistency check. Use AutoFix to repair errors when possible.
                    </DialogContentText>
                    <Box>
                        <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 0.5 }}>
                            Progress
                        </Typography>
                        <LinearProgress
                            variant={showIndeterminate ? "indeterminate" : "determinate"}
                            value={showIndeterminate ? undefined : Math.min(100, Math.max(0, progressValue))}
                        />
                        <Stack direction="row" justifyContent="space-between" sx={{ mt: 0.5 }}>
                            <Typography variant="caption" color="text.secondary">
                                {status ? status.toUpperCase() : "IDLE"}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                                {showIndeterminate ? "Working..." : `${Math.round(progressValue)}%`}
                            </Typography>
                        </Stack>
                    </Box>

                    {message && (
                        <Typography variant="body2" color="text.secondary">
                            {message}
                        </Typography>
                    )}

                    <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={autoFix}
                                    onChange={(event) => setAutoFix(event.target.checked)}
                                    disabled={isRunning}
                                />
                            }
                            label="AutoFix"
                        />
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={force}
                                    onChange={(event) => setForce(event.target.checked)}
                                    disabled={isRunning}
                                />
                            }
                            label="Force"
                        />
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={verbose}
                                    onChange={(event) => setVerbose(event.target.checked)}
                                    disabled={isRunning}
                                />
                            }
                            label="Verbose"
                        />
                    </Stack>

                    {verbose && (
                        <TextField
                            label="Logs"
                            value={logs.join("\n")}
                            multiline
                            minRows={6}
                            maxRows={12}
                            InputProps={{ readOnly: true }}
                            placeholder="No logs yet."
                        />
                    )}
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} color="secondary" variant="outlined">
                    Close
                </Button>
                <Button
                    onClick={isRunning ? handleStop : handleStart}
                    color={isRunning ? "error" : "primary"}
                    variant="contained"
                    disabled={isStarting || isStopping || !partition?.id}
                >
                    {isRunning ? "Stop" : "Start"}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
