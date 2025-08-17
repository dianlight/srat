import { useCallback, useEffect, useRef, useState } from "react";
import type { Disk } from "../../../store/sratApi";

const localStorageKey = "srat_volumes_expanded_accordion";

export function useVolumeAccordion(
	disks: Disk[] | undefined,
	isLoading: boolean,
	hideSystemPartitions: boolean,
) {
	const [expandedAccordion, setExpandedAccordion] = useState<string | false>(
		() => {
			const savedAccordionId = localStorage.getItem(localStorageKey);
			return savedAccordionId || false;
		},
	);
	const initialAutoOpenDone = useRef(false);
	const prevDisksRef = useRef<Disk[] | undefined>(undefined);
	const prevHideSystemPartitionsRef = useRef<boolean | undefined>(undefined);

	const isDiskRendered = useCallback(
		(disk: Disk, hideSystem: boolean): boolean => {
			const filteredPartitions =
				disk.partitions?.filter(
					(p) =>
						!(
							hideSystem &&
							p.system &&
							(p.name?.startsWith("hassos-") ||
								(p.host_mount_point_data && p.host_mount_point_data.length > 0))
						),
				) || [];
			const hasActualPartitions = disk.partitions && disk.partitions.length > 0;
			const allPartitionsAreHiddenByToggle =
				hasActualPartitions && filteredPartitions.length === 0 && hideSystem;
			return !allPartitionsAreHiddenByToggle;
		},
		[],
	);

	useEffect(() => {
		if (
			prevDisksRef.current !== disks ||
			prevHideSystemPartitionsRef.current !== hideSystemPartitions
		) {
			initialAutoOpenDone.current = false;
			prevDisksRef.current = disks;
			prevHideSystemPartitionsRef.current = hideSystemPartitions;
		}

		if (initialAutoOpenDone.current) {
			if (!disks || disks.length === 0) {
				if (expandedAccordion !== false) setExpandedAccordion(false);
			} else if (typeof expandedAccordion === "string") {
				const diskIndex = disks.findIndex(
					(d, idx) => (d.id || `disk-${idx}`) === expandedAccordion,
				);
				if (
					diskIndex === -1 ||
					!isDiskRendered(disks[diskIndex], hideSystemPartitions)
				) {
					setExpandedAccordion(false);
				}
			}
			return;
		}

		if (isLoading || !Array.isArray(disks)) {
			return;
		}

		if (disks.length === 0) {
			if (expandedAccordion !== false) setExpandedAccordion(false);
			initialAutoOpenDone.current = true;
			return;
		}

		let isValidLocalStorageValue = false;
		if (typeof expandedAccordion === "string") {
			const diskIndex = disks.findIndex(
				(d, idx) => (d.id || `disk-${idx}`) === expandedAccordion,
			);
			if (diskIndex !== -1) {
				isValidLocalStorageValue = isDiskRendered(
					disks[diskIndex],
					hideSystemPartitions,
				);
			}
		}

		if (!isValidLocalStorageValue) {
			let firstRenderedDiskIdentifier: string | null = null;
			for (let i = 0; i < disks.length; i++) {
				if (isDiskRendered(disks[i], hideSystemPartitions)) {
					firstRenderedDiskIdentifier = disks[i].id || `disk-${i}`;
					break;
				}
			}
			setExpandedAccordion(firstRenderedDiskIdentifier || false);
		}
		initialAutoOpenDone.current = true;
	}, [
		disks,
		isLoading,
		hideSystemPartitions,
		expandedAccordion,
		isDiskRendered,
	]);

	useEffect(() => {
		if (!initialAutoOpenDone.current) {
			return;
		}

		if (expandedAccordion === false) {
			localStorage.removeItem(localStorageKey);
		} else if (typeof expandedAccordion === "string") {
			let isValidToSave = false;
			if (Array.isArray(disks)) {
				const diskIndex = disks.findIndex(
					(d, idx) => (d.id || `disk-${idx}`) === expandedAccordion,
				);
				if (diskIndex !== -1) {
					isValidToSave = isDiskRendered(
						disks[diskIndex],
						hideSystemPartitions,
					);
				}
			}

			if (isValidToSave) {
				localStorage.setItem(localStorageKey, expandedAccordion);
			} else {
				localStorage.removeItem(localStorageKey);
			}
		}
	}, [expandedAccordion, disks, hideSystemPartitions, isDiskRendered]);

	const handleAccordionChange =
		(panel: string) => (_event: React.SyntheticEvent, isExpanded: boolean) => {
			setExpandedAccordion(isExpanded ? panel : false);
			if (!initialAutoOpenDone.current) {
				initialAutoOpenDone.current = true;
			}
		};

	return {
		expandedAccordion,
		handleAccordionChange,
		isDiskRendered,
		setExpandedAccordion,
	};
}
