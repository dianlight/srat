import {
  type Dispatch,
  type SetStateAction,
  useCallback,
  useEffect,
  useState,
} from "react";
import type { Disk, Partition } from "../../../store/sratApi";
import { getDiskIdentifier, getPartitionIdentifier } from "../utils";

export interface UseVolumeSelectionParams {
  sourceDisks?: Disk[];
}

export interface UseVolumeSelectionResult {
  selectedDisk: Disk | undefined;
  setSelectedDisk: Dispatch<SetStateAction<Disk | undefined>>;
  selectedPartition: Partition | undefined;
  setSelectedPartition: Dispatch<SetStateAction<Partition | undefined>>;
  selectedPartitionId: string | undefined;
  setSelectedPartitionId: Dispatch<SetStateAction<string | undefined>>;
  expandedDisks: string[];
  setExpandedDisks: Dispatch<SetStateAction<string[]>>;
  hideSystemPartitions: boolean;
  setHideSystemPartitions: Dispatch<SetStateAction<boolean>>;
  showPreview: boolean;
  setShowPreview: Dispatch<SetStateAction<boolean>>;
  showMount: boolean;
  setShowMount: Dispatch<SetStateAction<boolean>>;
  showFilesystemCheckDialog: boolean;
  setShowFilesystemCheckDialog: Dispatch<SetStateAction<boolean>>;
  showFilesystemLabelDialog: boolean;
  setShowFilesystemLabelDialog: Dispatch<SetStateAction<boolean>>;
  showFilesystemFormatDialog: boolean;
  setShowFilesystemFormatDialog: Dispatch<SetStateAction<boolean>>;
  handleDiskSelect: (disk: Disk) => void;
  handlePartitionSelect: (disk: Disk, partition: Partition) => void;
  clearSelection: () => void;
  openDialogForPartition: (
    partition: Partition,
    setDialogOpen: (open: boolean) => void,
  ) => void;
  openPreview: (disk?: Disk, partition?: Partition) => void;
  closePreview: () => void;
  openMount: (partition: Partition) => void;
  closeMount: () => void;
}

export function useVolumeSelection({
  sourceDisks,
}: UseVolumeSelectionParams = {}): UseVolumeSelectionResult {
  const [showPreview, setShowPreview] = useState<boolean>(false);
  const [showMount, setShowMount] = useState<boolean>(false);
  const [showFilesystemCheckDialog, setShowFilesystemCheckDialog] =
    useState(false);
  const [showFilesystemLabelDialog, setShowFilesystemLabelDialog] =
    useState(false);
  const [showFilesystemFormatDialog, setShowFilesystemFormatDialog] =
    useState(false);
  const [hideSystemPartitions, setHideSystemPartitions] = useState<boolean>(
    localStorage.getItem("volumes.hideSystemPartitions") === "true",
  );
  const [selectedDisk, setSelectedDisk] = useState<Disk | undefined>(undefined);
  const [selectedPartition, setSelectedPartition] = useState<
    Partition | undefined
  >(undefined);
  const [selectedPartitionId, setSelectedPartitionId] = useState<
    string | undefined
  >(() => localStorage.getItem("volumes.selectedPartitionId") || undefined);
  const [expandedDisks, setExpandedDisks] = useState<string[]>(() => {
    try {
      const savedExpanded = localStorage.getItem("volumes.expandedDisks");
      if (savedExpanded) {
        const parsed = JSON.parse(savedExpanded);
        if (Array.isArray(parsed)) return parsed as string[];
      }
    } catch {}
    return [];
  });

  const handleDiskSelect = useCallback(
    (disk: Disk) => {
      setSelectedDisk(disk);
      setSelectedPartition(undefined);
      const diskIdx = Math.max(sourceDisks?.indexOf(disk) ?? -1, 0);
      const diskIdentifier = getDiskIdentifier(disk, diskIdx);
      setSelectedPartitionId(diskIdentifier);
      setExpandedDisks((prev) =>
        prev.includes(diskIdentifier) ? prev : [...prev, diskIdentifier],
      );
    },
    [sourceDisks],
  );

  const handlePartitionSelect = useCallback(
    (disk: Disk, partition: Partition) => {
      setSelectedDisk(disk);
      setSelectedPartition(partition);
      const diskIdx = Math.max(sourceDisks?.indexOf(disk) ?? -1, 0);
      const diskIdentifier = getDiskIdentifier(disk, diskIdx);
      const partitionEntries = Object.entries(disk.partitions || {});
      const partitionEntry = partitionEntries.find(
        ([, value]) => value === partition,
      );
      const partitionKey = partitionEntry?.[0];
      const partIdx = Math.max(
        partitionEntry ? partitionEntries.indexOf(partitionEntry) : -1,
        0,
      );
      const partitionId = getPartitionIdentifier(
        diskIdentifier,
        partition,
        partitionKey,
        partIdx,
      );
      setSelectedPartitionId(partitionId);
      setExpandedDisks((prev) => {
        if (prev.includes(diskIdentifier)) return prev;
        return [...prev, diskIdentifier];
      });
    },
    [sourceDisks],
  );

  const clearSelection = useCallback(() => {
    setSelectedPartition(undefined);
    setSelectedDisk(undefined);
    setSelectedPartitionId(undefined);
  }, []);

  const openDialogForPartition = useCallback(
    (partition: Partition, setDialogOpen: (open: boolean) => void) => {
      const activeElement = document.activeElement;
      if (activeElement instanceof HTMLElement) {
        activeElement.blur();
      }
      setSelectedPartition(partition);
      setDialogOpen(true);
    },
    [],
  );

  const openPreview = useCallback((disk?: Disk, partition?: Partition) => {
    if (disk !== undefined) setSelectedDisk(disk);
    if (partition !== undefined) setSelectedPartition(partition);
    setShowPreview(true);
  }, []);

  const closePreview = useCallback(() => {
    clearSelection();
    setShowPreview(false);
  }, [clearSelection]);

  const openMount = useCallback((partition: Partition) => {
    setSelectedPartition(partition);
    setShowMount(true);
  }, []);

  const closeMount = useCallback(() => {
    clearSelection();
    setShowMount(false);
  }, [clearSelection]);

  useEffect(() => {
    try {
      if (selectedPartitionId) {
        localStorage.setItem(
          "volumes.selectedPartitionId",
          selectedPartitionId,
        );
      } else {
        localStorage.removeItem("volumes.selectedPartitionId");
      }
    } catch (err) {
      console.warn("Could not persist selectedPartitionId", err);
    }
  }, [selectedPartitionId]);

  useEffect(() => {
    try {
      if (expandedDisks.length > 0) {
        localStorage.setItem(
          "volumes.expandedDisks",
          JSON.stringify(expandedDisks),
        );
      } else {
        localStorage.removeItem("volumes.expandedDisks");
      }
    } catch (err) {
      console.warn("Could not persist expandedDisks", err);
    }
  }, [expandedDisks]);

  useEffect(() => {
    try {
      localStorage.setItem(
        "volumes.hideSystemPartitions",
        hideSystemPartitions ? "true" : "false",
      );
    } catch (err) {
      console.warn("Could not persist hideSystemPartitions", err);
    }
  }, [hideSystemPartitions]);

  useEffect(() => {
    if (!sourceDisks || sourceDisks.length === 0) return;
    if (!selectedPartitionId) return;

    for (const disk of sourceDisks) {
      const diskIdx = Math.max(sourceDisks.indexOf(disk), 0);
      const diskIdentifier = getDiskIdentifier(disk, diskIdx);
      if (diskIdentifier === selectedPartitionId) {
        setSelectedDisk(disk);
        setSelectedPartition(undefined);
        setExpandedDisks((prev) =>
          prev.includes(diskIdentifier) ? prev : [...prev, diskIdentifier],
        );
        return;
      }
      const partitionEntries = Object.entries(disk.partitions || {});
      if (!partitionEntries || partitionEntries.length === 0) continue;
      for (let partIdx = 0; partIdx < partitionEntries.length; partIdx++) {
        const [partitionKey, partition] = partitionEntries[partIdx] as [
          string,
          Partition,
        ];
        const partitionIdentifier = getPartitionIdentifier(
          diskIdentifier,
          partition,
          partitionKey,
          partIdx,
        );
        if (partitionIdentifier === selectedPartitionId) {
          setSelectedDisk(disk);
          setSelectedPartition(partition);
          return;
        }
      }
    }

    setSelectedPartition(undefined);
    setSelectedDisk(undefined);
    setSelectedPartitionId(undefined);
  }, [sourceDisks, selectedPartitionId]);

  return {
    selectedDisk,
    setSelectedDisk,
    selectedPartition,
    setSelectedPartition,
    selectedPartitionId,
    setSelectedPartitionId,
    expandedDisks,
    setExpandedDisks,
    hideSystemPartitions,
    setHideSystemPartitions,
    showPreview,
    setShowPreview,
    showMount,
    setShowMount,
    showFilesystemCheckDialog,
    setShowFilesystemCheckDialog,
    showFilesystemLabelDialog,
    setShowFilesystemLabelDialog,
    showFilesystemFormatDialog,
    setShowFilesystemFormatDialog,
    handleDiskSelect,
    handlePartitionSelect,
    clearSelection,
    openDialogForPartition,
    openPreview,
    closePreview,
    openMount,
    closeMount,
  };
}
