import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Box,
	FormControlLabel,
	Switch,
	Typography,
} from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useReadOnly } from "../../hooks/readonlyHook";
import { useVolume } from "../../hooks/volumeHook";
import { useGetApiIssuesQuery, type Issue, type Partition } from "../../store/sratApi";
import { ActionableItemsList } from "./components/ActionableItemsList";
import IssueCard from "../../components/IssueCard";

export function DashboardActions() {
	const { disks, isLoading, error } = useVolume();
	const read_only = useReadOnly();
	const _navigate = useNavigate();
	const [expanded, setExpanded] = useState(false);
	const [showIgnored, setShowIgnored] = useState(false);

	const { data: issues, isLoading: is_inLoading } = useGetApiIssuesQuery();

	const actionablePartitions = useMemo(() => {
		const partitions: { partition: Partition; action: "mount" | "share" }[] =
			[];
		if (disks && !read_only) {
			for (const disk of disks) {
				// disks type should be inferred from useVolume
				for (const partition of disk.partitions || []) {
					// Filter out system/host-mounted partitions
					if (
						partition.system ||
						partition.name?.startsWith("hassos-") ||
						(partition.host_mount_point_data &&
							partition.host_mount_point_data.length > 0)
					) {
						continue;
					}

					const isMounted = partition.mount_point_data?.some(
						(mpd) => mpd.is_mounted,
					);
					const hasShares = partition.mount_point_data?.some((mpd) =>
						mpd.shares?.some((share) => !share.disabled),
					);
					const firstMountPath = partition.mount_point_data?.[0]?.path;

					if (!isMounted) {
						partitions.push({ partition, action: "mount" });
					} else if (!hasShares && firstMountPath?.startsWith("/mnt/")) {
						partitions.push({ partition, action: "share" });
					}
				}
			}
		}
		return partitions;
	}, [disks, read_only]);

	function handleResolveIssue(id: number): void {
		throw new Error("Function not implemented.");
	}

	// Set initial expanded state based on content
	useEffect(() => {
		if (!isLoading && !error && (actionablePartitions.length + (Array.isArray(issues) ? issues.length : 0) > 0)) {
			setExpanded(true);
		}
	}, [isLoading, error, actionablePartitions.length, issues]);

	const handleAccordionChange = (_event: React.SyntheticEvent, isExpanded: boolean) => {
		setExpanded(isExpanded);
	};

	return (
		<Accordion
			expanded={expanded}
			onChange={handleAccordionChange}
		>
			<AccordionSummary
				expandIcon={<ExpandMoreIcon />}
				aria-controls="actions-content"
				id="actions-header"
			>
				<Box sx={{ display: 'flex', width: '100%', justifyContent: 'space-between', alignItems: 'center' }}>
					<Typography variant="h6">Actionable Items</Typography>
					<FormControlLabel
						onClick={(e) => e.stopPropagation()}
						onFocus={(e) => e.stopPropagation()}
						control={
							<Switch
								size="small"
								checked={showIgnored}
								onChange={(e) => {
									e.stopPropagation();
									setShowIgnored(e.target.checked);
								}}
							/>
						}
						label="Show Ignored"
						sx={{ mr: 1 }}
					/>
				</Box>
			</AccordionSummary>
			<AccordionDetails>
				{!is_inLoading && issues && Array.isArray(issues) &&
					issues.map((issue) => (
						<IssueCard
							key={issue.id}
							issue={issue}
							onResolve={handleResolveIssue}
							showIgnored={showIgnored}
						/>
					))
				}
				<ActionableItemsList
					actionablePartitions={actionablePartitions}
					isLoading={isLoading}
					error={error}
					showIgnored={showIgnored}
				/>
			</AccordionDetails>
		</Accordion>
	);
}
