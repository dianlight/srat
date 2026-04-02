import { SparkLineChart as MuiSparkLineChart } from "@mui/x-charts";
import React from "react";

// HMR-safe key: changes only when this module is reloaded (e.g., on HMR),
// causing a full remount of the chart to avoid stale internal hooks.
const HMR_SAFE_KEY =
  typeof Date !== "undefined" ? `spark-${Date.now()}` : undefined;

export type SafeSparkLineChartProps = React.ComponentProps<
  typeof MuiSparkLineChart
>; // passthrough to MUI SparkLineChart

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
