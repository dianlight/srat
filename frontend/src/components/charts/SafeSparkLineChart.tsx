import React from "react";
import { SparkLineChart as MuiSparkLineChart } from "@mui/x-charts";

// HMR-safe key: changes only when this module is reloaded (e.g., on HMR),
// causing a full remount of the chart to avoid stale internal hooks.
const HMR_SAFE_KEY = typeof Date !== "undefined" ? `spark-${Date.now()}` : undefined;

export type SafeSparkLineChartProps = any; // keep flexible; passthrough to MUI SparkLineChart

export function SafeSparkLineChart(props: SafeSparkLineChartProps) {
    // Defer rendering to after first commit to avoid issues with measurement/ResizeObservers
    const [mounted, setMounted] = React.useState(false);
    React.useEffect(() => {
        setMounted(true);
        return () => setMounted(false);
    }, []);

    if (!mounted) return null;

    return <MuiSparkLineChart key={HMR_SAFE_KEY} {...props} />;
}

export default SafeSparkLineChart;
