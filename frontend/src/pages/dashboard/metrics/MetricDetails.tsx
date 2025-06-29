import { useState } from "react";
import type { HealthPing } from "../../../store/sratApi";
import { DiskHealthMetricsAccordion } from "./DiskHealthMetricsAccordion";
import { NetworkHealthMetricsAccordion } from "./NetworkHealthMetricsAccordion";
import { ProcessMetricsAccordion } from "./ProcessMetricsAccordion";
import { SambaStatusMetricsAccordion } from "./SambaStatusMetricsAccordion";
import { SystemMetricsAccordion } from "./SystemMetricsAccordion";
import type { ProcessStatus } from "./types";

interface MetricDetailsProps {
	health: HealthPing | null;
	isLoading: boolean;
	error: Error | null | undefined | {};
	processData: ProcessStatus[];
	cpuHistory: Record<string, number[]>;
	memoryHistory: Record<string, number[]>;
	connectionsHistory: Record<string, number[]>;
}

export function MetricDetails({
	health,
	isLoading,
	error,
	processData,
	cpuHistory,
	memoryHistory,
	connectionsHistory,
}: MetricDetailsProps) {
	const [expandedAccordion, setExpandedAccordion] = useState<string | false>(
		"system-metrics-details",
	);

	const handleAccordionChange =
		(panel: string) => (_event: React.SyntheticEvent, isExpanded: boolean) => {
			setExpandedAccordion(isExpanded ? panel : false);
		};

	const handleDetailClick = (metricId: string) => {
		setExpandedAccordion(metricId);
	};

	return (
		<>
			<SystemMetricsAccordion
				health={health}
				isLoading={isLoading}
				error={error}
				expandedAccordion={expandedAccordion}
				onAccordionChange={handleAccordionChange}
				onDetailClick={handleDetailClick}
			/>
			<ProcessMetricsAccordion
				processData={processData}
				cpuHistory={cpuHistory}
				memoryHistory={memoryHistory}
				connectionsHistory={connectionsHistory}
				expanded={expandedAccordion === "processMetrics"}
				onChange={handleAccordionChange("processMetrics")}
			/>
			<DiskHealthMetricsAccordion
				diskHealth={health?.disk_health}
				expanded={expandedAccordion === "diskHealthMetrics"}
				onChange={handleAccordionChange("diskHealthMetrics")}
			/>
			<NetworkHealthMetricsAccordion
				networkHealth={health?.network_health}
				expanded={expandedAccordion === "networkHealthMetrics"}
				onChange={handleAccordionChange("networkHealthMetrics")}
			/>
			<SambaStatusMetricsAccordion
				sambaStatus={health?.samba_status}
				expanded={expandedAccordion === "sambaStatusMetrics"}
				onChange={handleAccordionChange("sambaStatusMetrics")}
			/>
		</>
	);
}
