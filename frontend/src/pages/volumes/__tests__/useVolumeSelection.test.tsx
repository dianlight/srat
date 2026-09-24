import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useEffect } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { Disk, Partition } from "../../../store/sratApi";
import { getDiskIdentifier, getPartitionIdentifier } from "../utils";
import {
  useVolumeSelection,
  type UseVolumeSelectionResult,
} from "../hooks/useVolumeSelection";

const disk: Disk = {
  id: "disk-1",
  model: "TestDisk",
  partitions: {
    "part-42": {
      id: "part-42",
      name: "TestPartition",
      size: 1024,
      mount_point_data: {},
    } as Partition,
  },
} as Disk;

const partition = Object.values(disk.partitions ?? {})[0] as Partition;

function Harness({
  sourceDisks,
  onResult,
}: {
  sourceDisks?: Disk[];
  onResult?: (result: UseVolumeSelectionResult) => void;
}) {
  const result = useVolumeSelection({ sourceDisks });
  useEffect(() => {
    onResult?.(result);
  }, [result, onResult]);
  return (
    <div>
      <p>selected id: {result.selectedPartitionId ?? "none"}</p>
      <p>selected disk: {result.selectedDisk?.id ?? "none"}</p>
      <p>selected partition: {result.selectedPartition?.id ?? "none"}</p>
      <p>selected partition name: {result.selectedPartition?.name ?? "none"}</p>
      <p>expanded: {result.expandedDisks.join(",") || "none"}</p>
      <p>hide system: {result.hideSystemPartitions ? "yes" : "no"}</p>
      <p>preview dialog: {result.showPreview ? "open" : "closed"}</p>
      <p>mount dialog: {result.showMount ? "open" : "closed"}</p>
      <p>check dialog: {result.showFilesystemCheckDialog ? "open" : "closed"}</p>
      <p>label dialog: {result.showFilesystemLabelDialog ? "open" : "closed"}</p>
      <p>
        format dialog: {result.showFilesystemFormatDialog ? "open" : "closed"}
      </p>
      <button
        type="button"
        onClick={() => result.handleDiskSelect(disk)}
      >
        select disk
      </button>
      <button
        type="button"
        onClick={() => result.handlePartitionSelect(disk, partition)}
      >
        select partition
      </button>
      <button type="button" onClick={() => result.setShowPreview(true)}>
        open preview
      </button>
      <button type="button" onClick={() => result.closePreview()}>
        close preview
      </button>
      <button type="button" onClick={() => result.setShowMount(true)}>
        open mount
      </button>
      <button type="button" onClick={() => result.closeMount()}>
        close mount
      </button>
      <button
        type="button"
        onClick={() =>
          result.openDialogForPartition(
            partition,
            result.setShowFilesystemCheckDialog,
          )
        }
      >
        open check dialog
      </button>
      <button
        type="button"
        onClick={() =>
          result.openDialogForPartition(
            partition,
            result.setShowFilesystemLabelDialog,
          )
        }
      >
        open label dialog
      </button>
      <button
        type="button"
        onClick={() =>
          result.openDialogForPartition(
            partition,
            result.setShowFilesystemFormatDialog,
          )
        }
      >
        open format dialog
      </button>
      <button
        type="button"
        onClick={() => result.setHideSystemPartitions(true)}
      >
        hide system
      </button>
      <button type="button" onClick={() => result.clearSelection()}>
        clear selection
      </button>
      <button
        type="button"
        onClick={() =>
          result.setSelectedPartition((currentPartition) =>
            currentPartition ? { ...currentPartition, name: "Renamed" } : currentPartition,
          )
        }
      >
        rename partition via updater
      </button>
    </div>
  );
}

describe("useVolumeSelection", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  afterEach(() => {
    localStorage.clear();
  });

  test("selecting a disk persists the selection id and expands the disk", async () => {
    const user = userEvent.setup();
    render(<Harness sourceDisks={[disk]} />);

    await user.click(screen.getByRole("button", { name: "select disk" }));

    expect(screen.getByText("selected disk: disk-1")).toBeInTheDocument();
    expect(screen.getByText("selected partition: none")).toBeInTheDocument();
    expect(localStorage.getItem("volumes.selectedPartitionId")).toBe("disk-1");
    expect(screen.getByText("expanded: disk-1")).toBeInTheDocument();
    expect(localStorage.getItem("volumes.expandedDisks")).toBe('["disk-1"]');
  });

  test("selecting a partition persists the composite partition id", async () => {
    const user = userEvent.setup();
    render(<Harness sourceDisks={[disk]} />);

    await user.click(screen.getByRole("button", { name: "select partition" }));

    const expectedId = getPartitionIdentifier(
      getDiskIdentifier(disk, 0),
      partition,
      "part-42",
      0,
    );
    expect(
      screen.getByText(`selected id: ${expectedId}`),
    ).toBeInTheDocument();
    expect(screen.getByText("selected partition: part-42")).toBeInTheDocument();
    expect(localStorage.getItem("volumes.selectedPartitionId")).toBe(
      expectedId,
    );
  });

  test("dialog open and close helpers toggle state", async () => {
    const user = userEvent.setup();
    render(<Harness sourceDisks={[disk]} />);

    expect(screen.getByText("preview dialog: closed")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "open preview" }));
    expect(screen.getByText("preview dialog: open")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "close preview" }));
    expect(screen.getByText("preview dialog: closed")).toBeInTheDocument();
    expect(screen.getByText("selected id: none")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "open mount" }));
    expect(screen.getByText("mount dialog: open")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "close mount" }));
    expect(screen.getByText("mount dialog: closed")).toBeInTheDocument();
  });

  test("openDialogForPartition selects the partition and opens the target dialog", async () => {
    const user = userEvent.setup();
    render(<Harness sourceDisks={[disk]} />);

    await user.click(screen.getByRole("button", { name: "open check dialog" }));
    expect(screen.getByText("check dialog: open")).toBeInTheDocument();
    expect(screen.getByText("selected partition: part-42")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "open label dialog" }));
    expect(screen.getByText("label dialog: open")).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: "open format dialog" }),
    );
    expect(screen.getByText("format dialog: open")).toBeInTheDocument();
  });

  test("restores a persisted partition selection from source disks", async () => {
    const expectedId = getPartitionIdentifier(
      getDiskIdentifier(disk, 0),
      partition,
      "part-42",
      0,
    );
    localStorage.setItem("volumes.selectedPartitionId", expectedId);

    render(<Harness sourceDisks={[disk]} />);

    expect(
      await screen.findByText("selected partition: part-42"),
    ).toBeInTheDocument();
    expect(screen.getByText("selected disk: disk-1")).toBeInTheDocument();
  });

  test("restores a persisted disk selection and expands the disk", async () => {
    localStorage.setItem("volumes.selectedPartitionId", "disk-1");

    render(<Harness sourceDisks={[disk]} />);

    expect(await screen.findByText("selected disk: disk-1")).toBeInTheDocument();
    expect(screen.getByText("selected partition: none")).toBeInTheDocument();
    expect(screen.getByText("expanded: disk-1")).toBeInTheDocument();
  });

  test("clears stale persisted selection ids that match nothing", async () => {
    localStorage.setItem("volumes.selectedPartitionId", "disk-1::ghost");

    render(<Harness sourceDisks={[disk]} />);

    expect(await screen.findByText("selected id: none")).toBeInTheDocument();
    expect(localStorage.getItem("volumes.selectedPartitionId")).toBeNull();
  });

  test("persists expanded disks and hide-system preference", async () => {
    const user = userEvent.setup();
    localStorage.setItem("volumes.expandedDisks", JSON.stringify(["disk-1"]));
    render(<Harness sourceDisks={[disk]} />);

    expect(screen.getByText("expanded: disk-1")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "hide system" }));
    expect(screen.getByText("hide system: yes")).toBeInTheDocument();
    expect(localStorage.getItem("volumes.hideSystemPartitions")).toBe("true");

    await user.click(screen.getByRole("button", { name: "clear selection" }));
    expect(localStorage.getItem("volumes.selectedPartitionId")).toBeNull();
  });

  test("supports functional updates to the selected partition", async () => {
    const user = userEvent.setup();
    render(<Harness sourceDisks={[disk]} />);

    await user.click(screen.getByRole("button", { name: "select partition" }));
    await user.click(
      screen.getByRole("button", { name: "rename partition via updater" }),
    );

    expect(
      screen.getByText("selected partition name: Renamed"),
    ).toBeInTheDocument();
    expect(localStorage.getItem("volumes.selectedPartitionId")).toBeTruthy();
  });

  test("survives invalid persisted expanded disks JSON", async () => {
    localStorage.setItem("volumes.expandedDisks", "invalid-json");
    render(<Harness sourceDisks={[disk]} />);

    expect(screen.getByText("expanded: none")).toBeInTheDocument();
  });
});
