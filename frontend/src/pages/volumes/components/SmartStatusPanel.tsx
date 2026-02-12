import ErrorIcon from "@mui/icons-material/Error";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import HealthAndSafetyIcon from "@mui/icons-material/HealthAndSafety";
import ThermostatIcon from "@mui/icons-material/Thermostat";
import {
    Box,
    Button,
    Card,
    CardContent,
    CardHeader,
    Chip,
    CircularProgress,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    IconButton,
    InputLabel,
    MenuItem,
    Select,
    Stack,
    Typography
} from "@mui/material";
import Collapse from "@mui/material/Collapse";
import {
    useEffect,
    useState
} from "react";
import { PreviewDialog } from "../../../components/PreviewDialog";
import { useSmartTestStatus } from "../../../hooks/smartTestStatusHook";
import { useSmartOperations } from "../../../hooks/useSmartOperations";
import { useGetApiDiskByDiskIdSmartStatusQuery, type SmartInfo, type SmartStatus } from "../../../store/sratApi";

// Local type definitions for SMART data that isn't in the OpenAPI spec yet
interface SmartHealthStatus {
    passed: boolean;
    failing_attributes?: string[];
    overall_status: "healthy" | "warning" | "failing";
}

interface SmartTestStatus {
    status: "idle" | "running" | "completed" | "failed";
    test_type?: string;
    percent_complete?: number;
    lba_of_first_error?: string;
}

export type SmartTestType = "short" | "long" | "conveyance";

interface SmartStatusPanelProps {
    smartInfo?: SmartInfo;
    diskId?: string;
    healthStatus?: SmartHealthStatus;
    testStatus?: SmartTestStatus;
    isSmartSupported?: boolean;
    isReadOnlyMode?: boolean;
    isExpanded?: boolean;
    onSetExpanded?: (expanded: boolean) => void;
}

export function SmartStatusPanel({
    smartInfo,
    diskId,
    isReadOnlyMode = false,
    isExpanded: initialExpanded = true,
    onSetExpanded,
}: SmartStatusPanelProps) {
    const [smartExpanded, setSmartExpanded] = useState(initialExpanded);
    const [showStartTestDialog, setShowStartTestDialog] = useState(false);
    const [showPreviewDialog, setShowPreviewDialog] = useState(false);
    const [selectedTestType, setSelectedTestType] = useState<SmartTestType>("short");
    const { startSelfTest, abortSelfTest, enableSmart, disableSmart, isLoading: smartOperationLoading, isSuccess: smartOperationSuccess } = useSmartOperations(diskId);
    const { data: smartStatus, isLoading: smartStatusIsLoading, refetch: refetchSmartStatus } = useGetApiDiskByDiskIdSmartStatusQuery({
        diskId: diskId || ""
    }, {
        skip: !diskId,
        refetchOnMountOrArgChange: true,
    });
    const { smartTestStatus, isLoading: smartTestStatusLoading } = useSmartTestStatus(diskId || "");

    // Don't render if SMART is not supported based on backend data
    if (!smartInfo?.supported) {
        return null;
    }

    const handleStartTest = () => {
        startSelfTest(selectedTestType);
        setShowStartTestDialog(false);
    };

    const getHealthIcon = () => {
        if (smartStatusIsLoading || !smartStatus) return null;
        if ((smartStatus as SmartStatus)?.is_in_danger) {
            return <ErrorIcon sx={{ color: "error.main" }} />;
        }
        if ((smartStatus as SmartStatus)?.is_in_warning) {
            return <ThermostatIcon sx={{ color: "warning.main" }} />;
        }
        if ((smartStatus as SmartStatus)?.is_test_passed) {
            return <HealthAndSafetyIcon sx={{ color: "success.main" }} />;
        }
        return <ErrorIcon sx={{ color: "error.main" }} />;
    };
    /*
        const getHealthColor = () => {
            if (smartStatusIsLoading || !smartStatus) return "default";
            if ((smartStatus as SmartStatus)?.is_test_passed) return "success";
            if (!(smartStatus as SmartStatus)?.is_test_passed || (smartStatus as SmartStatus)?.is_in_warning) return "warning";
            return "error";
        };
        */

    useEffect(() => {
        if (smartOperationSuccess && !smartOperationLoading) {
            refetchSmartStatus();
        }
    }, [smartOperationSuccess]);

    const getTestStatusColor = () => {
        if (smartTestStatusLoading || !smartTestStatus) return "default";
        if ((smartStatus as SmartStatus)?.is_test_passed && !smartTestStatus.running) return "success";
        if (smartTestStatus.running) return "info";
        if (!(smartStatus as SmartStatus)?.is_test_passed && !smartTestStatus.running) return "error";
        return "default";
    };

    return (
        <Card>
            <CardHeader
                title="S.M.A.R.T. Status"
                avatar={
                    <IconButton
                        size="small"
                        aria-label="smart preview"
                        onClick={() => setShowPreviewDialog(true)}
                        disabled={smartStatusIsLoading}
                    >
                        {getHealthIcon() || <HealthAndSafetyIcon color="primary" />}
                    </IconButton>
                }
                action={
                    <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                        <IconButton
                            onClick={() => {
                                const newExpanded = !smartExpanded;
                                setSmartExpanded(newExpanded);
                                onSetExpanded?.(newExpanded);
                            }}
                            aria-expanded={smartExpanded}
                            aria-label="show more"
                            sx={{
                                transform: smartExpanded ? "rotate(180deg)" : "rotate(0deg)",
                                transition: "transform 150ms cubic-bezier(0.4, 0, 0.2, 1)",
                            }}
                        >
                            <ExpandMoreIcon />
                        </IconButton>
                    </Box>
                }
            />
            <Collapse in={smartExpanded} timeout="auto" unmountOnExit>
                <CardContent>
                    <Stack spacing={3}>
                        {/* Disk Type & RPM Section */}
                        <Box>
                            <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                Device Information
                            </Typography>
                            <Stack direction="row" spacing={2} alignItems="center" flexWrap="wrap" sx={{ gap: 1 }}>
                                {smartInfo.disk_type && (
                                    <Chip
                                        label={smartInfo.disk_type}
                                        color="primary"
                                        size="small"
                                        variant="outlined"
                                    />
                                )}
                                {smartInfo.model_name && (
                                    <Chip
                                        label={smartInfo.model_name}
                                        color="secondary"
                                        size="small"
                                        variant="outlined"
                                    />
                                )}
                                {smartInfo.model_family && (
                                    <Chip
                                        label={smartInfo.model_family}
                                        color="default"
                                        size="small"
                                        variant="outlined"
                                    />
                                )}
                                {smartInfo.firmware_version && (
                                    <Chip
                                        label={`FW ${smartInfo.firmware_version}`}
                                        color="default"
                                        size="small"
                                        variant="outlined"
                                    />
                                )}
                                {smartInfo.serial_number && (
                                    <Chip
                                        label={`SN ${smartInfo.serial_number}`}
                                        color="default"
                                        size="small"
                                        variant="outlined"
                                    />
                                )}
                                {smartInfo.rotation_rate && smartInfo.rotation_rate > 0 && (
                                    <Chip
                                        label={`${smartInfo.rotation_rate} RPM`}
                                        color="info"
                                        size="small"
                                        variant="outlined"
                                    />
                                )}
                            </Stack>
                        </Box>

                        {/* Temperature Section */}
                        {!smartStatusIsLoading && smartStatus && (smartStatus as SmartStatus)?.temperature && (
                            <Box>
                                <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                                    <ThermostatIcon fontSize="small" color="primary" />
                                    <Typography variant="subtitle2" color="text.secondary">
                                        Temperature
                                    </Typography>
                                </Stack>
                                <Stack direction="row" spacing={2} alignItems="center">
                                    <Typography variant="h6">
                                        {(smartStatus as SmartStatus).temperature.value}°C
                                    </Typography>
                                    {(smartStatus as SmartStatus).temperature.min || (smartStatus as SmartStatus).temperature.max ? (
                                        <Typography variant="caption" color="text.secondary">
                                            {(smartStatus as SmartStatus).temperature.min && `Min: ${(smartStatus as SmartStatus).temperature.min}°C`}
                                            {(smartStatus as SmartStatus).temperature.min && (smartStatus as SmartStatus).temperature.max && " / "}
                                            {(smartStatus as SmartStatus).temperature.max && `Max: ${(smartStatus as SmartStatus).temperature.max}°C`}
                                        </Typography>
                                    ) : null}
                                </Stack>
                            </Box>
                        )}

                        {/* Power Stats Section */}
                        {!smartStatusIsLoading && smartStatus && (smartStatus as SmartStatus)?.power_on_hours && (
                            <Stack spacing={1}>
                                <Typography variant="subtitle2" color="text.secondary">
                                    Power Statistics
                                </Typography>
                                <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
                                    {(smartStatus as SmartStatus).power_on_hours && (
                                        <Box sx={{ flex: 1 }}>
                                            <Typography variant="caption" color="text.secondary">
                                                Power-On Hours
                                            </Typography>
                                            <Typography variant="body2">
                                                {(smartStatus as SmartStatus).power_on_hours.value.toLocaleString()} hours
                                            </Typography>
                                        </Box>
                                    )}
                                    {(smartStatus as SmartStatus).power_cycle_count && (
                                        <Box sx={{ flex: 1 }}>
                                            <Typography variant="caption" color="text.secondary">
                                                Power Cycles
                                            </Typography>
                                            <Typography variant="body2">
                                                {(smartStatus as SmartStatus).power_cycle_count.value.toLocaleString()} cycles
                                            </Typography>
                                        </Box>
                                    )}
                                </Stack>
                            </Stack>
                        )}

                        {/* Health Status Section */}
                        {(smartStatus as SmartStatus) && (
                            <Box>
                                <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                    Health Check
                                </Typography>
                                <Stack spacing={1}>
                                    <Chip
                                        label={(smartStatus as SmartStatus).is_test_passed ? "All attributes healthy" : "Issues detected"}
                                        color={(smartStatus as SmartStatus).is_test_passed ? "success" : "error"}
                                        size="small"
                                    />
                                    {!(smartStatus as SmartStatus).is_test_passed && (smartStatus as SmartStatus).others && Object.keys((smartStatus as SmartStatus).others || {}).length > 0 && (
                                        <Box>
                                            <Typography variant="caption" color="error.main" sx={{ display: "block", mb: 0.5 }}>
                                                Failing Attributes:
                                            </Typography>
                                            <Stack direction="row" spacing={0.5} flexWrap="wrap" sx={{ gap: 0.5 }}>
                                                {(smartStatus as SmartStatus).others && Object.entries((smartStatus as SmartStatus).others || {}).map((attr) => (
                                                    <Chip
                                                        key={attr[0]}
                                                        label={JSON.stringify(attr)}
                                                        size="small"
                                                        variant="outlined"
                                                        color="error"
                                                    />
                                                ))}
                                            </Stack>
                                        </Box>
                                    )}
                                </Stack>
                            </Box>
                        )}

                        {/* Self-Test Status Section */}
                        {!smartTestStatusLoading && (
                            <Box>
                                <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                    Self-Test Status
                                </Typography>
                                <Stack spacing={1}>
                                    <Stack direction="row" spacing={2} alignItems="center" justifyContent="space-between">
                                        <Stack direction="row" spacing={1} alignItems="center" sx={{ flex: 1 }}>
                                            <Chip
                                                label={smartTestStatus?.status}
                                                color={getTestStatusColor()}
                                                size="small"
                                            />
                                            {smartTestStatus?.test_type && (
                                                <Typography variant="caption" color="text.secondary">
                                                    ({smartTestStatus.test_type})
                                                </Typography>
                                            )}
                                        </Stack>
                                        {(smartTestStatus?.running) && smartTestStatus.percent_complete !== undefined && (
                                            <Box sx={{ position: 'relative', display: 'inline-flex' }}>
                                                <CircularProgress
                                                    variant="determinate"
                                                    value={smartTestStatus.percent_complete}
                                                    size={40}
                                                />
                                                <Box
                                                    sx={{
                                                        top: 0,
                                                        left: 0,
                                                        bottom: 0,
                                                        right: 0,
                                                        position: 'absolute',
                                                        display: 'flex',
                                                        alignItems: 'center',
                                                        justifyContent: 'center',
                                                    }}
                                                >
                                                    <Typography variant="caption" component="div" color="text.secondary">
                                                        {`${Math.round(smartTestStatus.percent_complete)}%`}
                                                    </Typography>
                                                </Box>
                                            </Box>
                                        )}
                                    </Stack>
                                    {smartTestStatus?.lba_of_first_error && (
                                        <Typography variant="caption" color="error">
                                            Error at LBA: {smartTestStatus.lba_of_first_error}
                                        </Typography>
                                    )}
                                </Stack>
                            </Box>
                        )}

                        {/* Control Buttons */}

                        <Box>
                            <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                                Actions
                            </Typography>
                            <Stack direction={{ xs: "column", sm: "row" }} spacing={1} sx={{ flexWrap: "wrap" }}>
                                <Button
                                    size="small"
                                    variant="outlined"
                                    onClick={() => setShowStartTestDialog(true)}
                                    disabled={(smartTestStatus?.running) || smartOperationLoading || isReadOnlyMode}
                                    title={
                                        (smartTestStatus?.running)
                                            ? "Test already running"
                                            : "Start SMART self-test"
                                    }
                                >
                                    Start Test
                                </Button>
                                <Button
                                    size="small"
                                    variant="outlined"
                                    color="warning"
                                    onClick={abortSelfTest}
                                    disabled={!(smartTestStatus?.running) || smartOperationLoading || isReadOnlyMode}
                                    title={
                                        !(smartTestStatus?.running)
                                            ? "No test running"
                                            : "Abort running self-test"
                                    }
                                >
                                    Abort Test
                                </Button>
                                <Button
                                    size="small"
                                    variant="outlined"
                                    onClick={enableSmart}
                                    disabled={smartOperationLoading || smartStatusIsLoading || ((smartStatus as SmartStatus)?.enabled ?? false) || isReadOnlyMode}
                                    title={
                                        (smartStatus as SmartStatus)?.enabled ?? false
                                            ? "SMART already enabled"
                                            : "Enable SMART monitoring"
                                    }
                                >
                                    Enable SMART
                                </Button>
                                <Button
                                    size="small"
                                    variant="outlined"
                                    color="error"
                                    onClick={disableSmart}
                                    disabled={smartOperationLoading || smartStatusIsLoading || !((smartStatus as SmartStatus)?.enabled ?? false) || isReadOnlyMode}
                                    title={
                                        !((smartStatus as SmartStatus)?.enabled ?? false)
                                            ? "SMART already disabled"
                                            : "Disable SMART monitoring"
                                    }
                                >
                                    Disable SMART
                                </Button>
                            </Stack>
                        </Box>

                        { /*End Control Buttons  */}
                    </Stack>
                </CardContent>
            </Collapse>

            {/* Start Test Dialog */}
            <Dialog open={showStartTestDialog} onClose={() => setShowStartTestDialog(false)} maxWidth="sm" fullWidth>
                <DialogTitle>Start SMART Self-Test</DialogTitle>
                <DialogContent sx={{ pt: 2 }}>
                    <FormControl fullWidth>
                        <InputLabel>Test Type</InputLabel>
                        <Select
                            value={selectedTestType}
                            label="Test Type"
                            onChange={(e) => setSelectedTestType(e.target.value as SmartTestType)}
                        >
                            <MenuItem value="short">Short (2-5 minutes)</MenuItem>
                            <MenuItem value="long">Long (hours - full disk scan)</MenuItem>
                            <MenuItem value="conveyance">Conveyance (test for transport damage)</MenuItem>
                        </Select>
                    </FormControl>
                    <Typography variant="caption" color="text.secondary" sx={{ display: "block", mt: 2 }}>
                        The selected test type will be executed on the disk. Running tests may impact disk performance.
                    </Typography>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setShowStartTestDialog(false)}>Cancel</Button>
                    <Button
                        onClick={handleStartTest}
                        variant="contained"
                        disabled={smartOperationLoading}
                    >
                        Start
                    </Button>
                </DialogActions>
            </Dialog>

            {/* Preview Dialog */}
            <PreviewDialog
                open={showPreviewDialog}
                onClose={() => setShowPreviewDialog(false)}
                objectToDisplay={{
                    smartStatus: smartStatus,
                    smartInfo: smartInfo,
                    testStatus: smartTestStatus,
                }}
                title="SMART Status Details"
            />
        </Card>
    );
}
