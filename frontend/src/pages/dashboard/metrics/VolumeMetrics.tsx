
import { Grid, Card, CardHeader, CardContent, Typography, CircularProgress, Alert, useTheme } from "@mui/material";
import { type Disk, type DiskHealth, type HealthPing } from "../../../store/sratApi";
import { humanizeBytes, decodeEscapeSequence } from "./utils";
import WarningIcon from '@mui/icons-material/Warning';
import { rainbowSurgePalette } from "@mui/x-charts/colorPalettes";
import { color } from "bun";
import {
    pieArcClasses,
    PieChart,
    //   pieClasses,
} from '@mui/x-charts/PieChart';

export function VolumeMetrics({ diskHealth }: { diskHealth: DiskHealth }) {

    const theme = useTheme();
    const palette = rainbowSurgePalette(theme.palette.mode);

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
                const partitionSeriesData = (diskHealth.per_partition_info[diskname] || [])
                    .map((p) => {
                        return {
                            label: p.device || 'unknown',
                            value: p.total_space_bytes || 0,
                        }
                    });

                const freespaceSeriesData = (diskHealth.per_partition_info[diskname] || [])
                    .map((p, index) => {
                        return [{
                            label: p.device || 'unknown',
                            value: (p.total_space_bytes || 0) - (p.free_space_bytes || 0),
                            color: palette[index]
                        }, {
                            label: p.device || 'unknown',
                            value: p.free_space_bytes || 0,
                            color: 'gray',
                        }]
                    }).flat();


                /*






                const chartData = (diskHealth?.per_partition_info[diskname] || [])
                    .filter(p => p.total_space_bytes > 0)
                    .map((p) => {
                        const fsType = p.fstype || 'N/A';
                        const fsckNeeded = p.fsck_needed || false;

                        return {
                            id: p.device || 'unknown',
                            value: p.total_space_bytes || 0,
                            label: `${fsType}`,
                            icon: fsckNeeded ? <WarningIcon fontSize="small" /> : null,
                        };
                    });
                const freeData = (diskHealth?.per_partition_info[diskname] || [])
                    .filter(p => p.total_space_bytes > 0)
                    .map((p) => {
                        const freeSpace = p.free_space_bytes ? humanizeBytes(p.free_space_bytes) : 'N/A';
                        const fsType = p.fstype || 'N/A';
                        const fsckNeeded = p.fsck_needed || false;

                        return {
                            id: p.device || 'unknown',
                            value: p.free_space_bytes || 0,
                            label: `Free: ${freeSpace}`,
                            icon: fsckNeeded ? <WarningIcon fontSize="small" /> : null,
                        };
                    });
                */
                console.log(`Disk: ${diskname}, Chart Data:`, partitionSeriesData, 'Free Data:', freespaceSeriesData);

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
                                    series={[
                                        {
                                            innerRadius: 0,
                                            outerRadius: 80,
                                            data: partitionSeriesData,
                                            highlightScope: { fade: 'global', highlight: 'item' },
                                            valueFormatter: (item) => humanizeBytes(item.value || 0),
                                            // arcLabel: (item) => humanizeBytes(item.value || 0),
                                        },
                                        {
                                            id: 'outer',
                                            innerRadius: 90,
                                            outerRadius: 100,
                                            data: freespaceSeriesData,
                                            highlightScope: { fade: 'global', highlight: 'item' },
                                            //                                            arcLabel: (item) => humanizeBytes(item.value || 0),
                                            valueFormatter: (item) => humanizeBytes(item.value || 0),
                                        },
                                    ]}

                                    hideLegend
                                    width={300}
                                    height={250}
                                    sx={{
                                        [`.${pieArcClasses.series}[data-series="outer"] .${pieArcClasses.root}`]: {
                                            opacity: 0.6,
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

