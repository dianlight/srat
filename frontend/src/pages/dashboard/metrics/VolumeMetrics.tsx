
import { Grid, Card, CardHeader, CardContent, Typography, CircularProgress, Alert } from "@mui/material";
import { PieChart } from '@mui/x-charts/PieChart';
import { type Disk, type DiskHealth, type HealthPing } from "../../../store/sratApi";
import { humanizeBytes, decodeEscapeSequence } from "./utils";
import WarningIcon from '@mui/icons-material/Warning';

export function VolumeMetrics({ diskHealth }: { diskHealth: DiskHealth }) {

    if (!diskHealth || !diskHealth.per_partition_info || Object.keys(diskHealth.per_partition_info).length === 0) {
        return (
            <Grid container spacing={3}>
                <Grid size={{ xs: 12 }}>
                    <Alert severity="info">
                        No disk health information available.
                    </Alert>
                </Grid>
            </Grid>
        );
    }

    return (
        <Grid container spacing={3}>
            {Object.keys(diskHealth?.per_partition_info).map(diskname => {
                const chartData = (diskHealth?.per_partition_info[diskname] || [])
                    .filter(p => p.total_space_bytes > 0)
                    .map((p) => {
                        const freeSpace = p.free_space_bytes ? humanizeBytes(p.free_space_bytes) : 'N/A';
                        const fsType = p.fstype || 'N/A';
                        const fsckNeeded = p.fsck_needed || false;

                        return {
                            id: p.device || 'unknown',
                            value: p.total_space_bytes || 0,
                            label: `${decodeEscapeSequence(p.device || 'Unknown')} - ${fsType} - Free: ${freeSpace}`,
                            icon: fsckNeeded ? <WarningIcon fontSize="small" /> : null,
                        };
                    });

                return (
                    <Grid size={{ xs: 12, md: 6, lg: 4 }} key={diskname}>
                        <Card sx={{ height: '100%' }}>
                            <CardHeader
                                title={decodeEscapeSequence(diskname || 'Unknown Disk')}
                                slotProps={{
                                    title: {
                                        variant: 'subtitle2',
                                        noWrap: true,
                                    },
                                }}
                            />
                            <CardContent sx={{ width: '100%', height: 300, display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                                <PieChart
                                    series={[{
                                        data: chartData,
                                        highlightScope: { fade: 'global', highlight: 'item' },
                                        faded: { innerRadius: 30, additionalRadius: -30, color: 'gray' },
                                        arcLabel: (item) => humanizeBytes(item.value || 0),
                                        arcLabelMinAngle: 25,
                                        valueFormatter: (item) => `${item.label}`,
                                        innerRadius: 30,
                                        outerRadius: 100,
                                        paddingAngle: 5,
                                        cornerRadius: 15,
                                        cx: 150,
                                        cy: 115,
                                    }]}
                                    width={300}
                                    height={250}
                                    slotProps={{
                                        legend: {
                                            direction: "horizontal",
                                            position: { vertical: 'bottom', horizontal: 'center' },
                                            sx: {
                                                fontSize: '0.75rem',
                                                gap: 1,
                                            },
                                            //itemMarkWidth: 10,
                                            // itemMarkHeight: 10,

                                            //    labelStyle: {
                                            //        display: 'flex',
                                            //        alignItems: 'center',
                                            //    }
                                        },
                                    }}
                                />
                            </CardContent>
                        </Card>
                    </Grid>
                );
            })}
        </Grid>
    );
}

