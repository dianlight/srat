
import { Card, CardHeader, CardContent, Typography, Box, CircularProgress, Alert, useTheme } from "@mui/material";
import { Sparklines, SparklinesLine, SparklinesSpots, SparklinesBars } from 'react-sparklines';

export interface MetricCardProps {
    title: string;
    subheader?: string;
    value: string;
    history?: number[];
    isLoading: boolean;
    error: boolean;
    historyType?: 'line' | 'bar';
    children?: React.ReactNode;
}

export function MetricCard({ title, subheader, value, history, isLoading, error, historyType = 'line', children }: MetricCardProps) {
    const theme = useTheme();

    const renderHistory = () => {
        if (history && history.length <= 1) {
            return <Typography variant="caption">gathering data...</Typography>;
        } else if (!history) {
            return "";
        }

        if (historyType === 'bar') {
            return (
                <Sparklines data={history} limit={10} width={100} height={40}>
                    <SparklinesBars style={{ fill: theme.palette.info.main, fillOpacity: ".5" }} />
                </Sparklines>
            );
        }

        return (
            <Sparklines data={history} limit={10} width={100} height={40}>
                <SparklinesLine color={theme.palette.primary.main} />
                <SparklinesSpots />
            </Sparklines>
        );
    };

    const renderContent = () => {
        if (isLoading) {
            return (
                <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                    <CircularProgress />
                </Box>
            );
        }

        if (error) {
            return <Alert severity="warning">{title} data not available.</Alert>;
        }

        return (
            <>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                    <Typography variant="h4" component="div">
                        {value}
                    </Typography>
                    <Box sx={{ width: '50%', height: 40 }}>
                        {renderHistory()}
                    </Box>
                </Box>
                {children}
            </>
        );
    };

    return (
        <Card>
            <CardHeader title={title} subheader={subheader} />
            <CardContent>
                {renderContent()}
            </CardContent>
        </Card>
    );
}
