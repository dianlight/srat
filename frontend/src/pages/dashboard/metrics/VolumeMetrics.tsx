
import { Grid, Card, CardHeader, CardContent, Typography, CircularProgress, Alert } from "@mui/material";
import { PieChart } from '@mui/x-charts/PieChart';
import { type Disk } from "../../../store/sratApi";
import { humanizeBytes, decodeEscapeSequence } from "./utils";

interface VolumeMetricsProps {
    disks: Disk[] | undefined;
    isLoadingVolumes: boolean;
    errorVolumes: Error | null | undefined | {};
}

export function VolumeMetrics({ disks, isLoadingVolumes, errorVolumes }: VolumeMetricsProps) {
    if (isLoadingVolumes) {
        return (
            <Grid size={{ xs: 12, md: 6, lg: 4 }}>
                <CircularProgress />
            </Grid>
        );
    }

    if (errorVolumes) {
        return (
            <Grid size={{ xs: 12, md: 6, lg: 4 }}>
                <Alert severity="error">Could not load disk information.</Alert>
            </Grid>
        );
    }

    const disksWithPartitions = disks?.filter(d => d.partitions && d.partitions.some(p => p.size && p.size > 0)) || [];

    if (disksWithPartitions.length === 0) {
        return <Typography>No partitions with size information found to display.</Typography>;
    }

    return (
        <Grid container spacing={3}>
            {disksWithPartitions.map(disk => {
                const chartData = (disk.partitions || [])
                    .filter(p => p.size && p.size > 0)
                    .map((p) => ({
                        id: p.id || p.device || 'unknown',
                        value: p.size || 0,
                        label: decodeEscapeSequence(p.name || p.device || 'Unknown'),
                    }));

                return (
                    <Grid size={{ xs: 12, md: 6, lg: 4 }} key={disk.id}>
                        <Card sx={{ height: '100%' }}>
                            <CardHeader
                                title={decodeEscapeSequence(disk.id || disk.model || 'Unknown Disk')}
                                slotProps={
                                    {
                                        title: {
                                            variant: 'subtitle2',
                                            noWrap: true,
                                        },
                                    }
                                }
                            />
                            <CardContent sx={{ width: '100%', height: 300, display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                                <PieChart
                                    series={[{
                                        data: chartData,
                                        highlightScope: { fade: 'global', highlight: 'item' },
                                        faded: { innerRadius: 30, additionalRadius: -30, color: 'gray' },
                                        arcLabel: (item) => humanizeBytes(item.value || 0),
                                        arcLabelMinAngle: 25,
                                        valueFormatter: (item) => humanizeBytes(item.value || 0),
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
