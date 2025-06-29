import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Typography,
} from "@mui/material";
import { useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { useReadOnly } from "../../hooks/readonlyHook";
import { useVolume } from "../../hooks/volumeHook";
import type { Partition } from "../../store/sratApi";
import { ActionableItemsList } from "./components/ActionableItemsList";

export function DashboardActions() {
	const { disks, isLoading, error } = useVolume();
	const read_only = useReadOnly();
	const navigate = useNavigate();

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

	return (
		<Accordion
			key={isLoading ? "loading" : "loaded"}
			defaultExpanded={!isLoading && !error && actionablePartitions.length > 0}
		>
			<AccordionSummary
				expandIcon={<ExpandMoreIcon />}
				aria-controls="actions-content"
				id="actions-header"
			>
				<Typography variant="h6">Actionable Items</Typography>
			</AccordionSummary>
			<AccordionDetails>
				<ActionableItemsList
					actionablePartitions={actionablePartitions}
					isLoading={isLoading}
					error={error}
				/>
			</AccordionDetails>
		</Accordion>
	);
}
