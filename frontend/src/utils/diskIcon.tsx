import ComputerIcon from "@mui/icons-material/Computer";
import EjectIcon from "@mui/icons-material/Eject";
import SdStorageIcon from "@mui/icons-material/SdStorage";
import UsbIcon from "@mui/icons-material/Usb";
import type { Disk } from "../store/sratApi";

interface DiskIconProps {
  disk: Disk;
  /** VolumesTreeView uses the default color; VolumeDetailsPanel passes "primary". */
  color?: "primary";
}

/**
 * Single shared disk-connection icon used by VolumesTreeView and
 * VolumeDetailsPanel. Keeps the USB/SD/removable/default mapping in one place.
 */
export function DiskIcon({ disk, color }: DiskIconProps) {
  const iconProps = color ? { color } : {};
  switch (disk.connection_bus?.toLowerCase()) {
    case "usb":
      return <UsbIcon {...iconProps} />;
    case "sdio":
    case "mmc":
      return <SdStorageIcon {...iconProps} />;
  }
  if (disk.removable) {
    return <EjectIcon {...iconProps} />;
  }
  return <ComputerIcon {...iconProps} />;
}
