import { useVolume } from "../../hooks/volumeHook";
import { type Partition } from "../../store/sratApi";
import { Accordion, AccordionDetails, AccordionSummary, Box, Button, CircularProgress, List, ListItem, ListItemText, Typography, Alert } from "@mui/material";
import { useNavigate } from "react-router-dom";
import { TabIDs, type LocationState } from "../../store/locationState";
import { useReadOnly } from "../../hooks/readonlyHook";
import { useMemo } from "react";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';

// A simplified version of what's in Volumes.tsx
function decodeEscapeSequence(source: string) {
    if (typeof source !== 'string') return '';
    return source.replace(/\\x([0-9A-Fa-f]{2})/g, function (_match, group1) {
        return String.fromCharCode(parseInt(String(group1), 16));
    });
};

export function DashboardActions() {
    const { disks, isLoading, error } = useVolume();
    const read_only = useReadOnly();
    const navigate = useNavigate();

    const actionablePartitions = useMemo(() => {
        const partitions: { partition: Partition, action: 'mount' | 'share' }[] = [];
        if (disks && !read_only) {
            for (const disk of disks) { // disks type should be inferred from useVolume
                for (const partition of disk.partitions || []) {
                    // Filter out system/host-mounted partitions
                    if (partition.system || partition.name?.startsWith('hassos-') || (partition.host_mount_point_data && partition.host_mount_point_data.length > 0)) {
                        continue;
                    }

                    const isMounted = partition.mount_point_data?.some(mpd => mpd.is_mounted);
                    const hasShares = partition.mount_point_data?.some(mpd => mpd.shares?.some(share => !share.disabled));
                    const firstMountPath = partition.mount_point_data?.[0]?.path;

                    if (!isMounted) {
                        partitions.push({ partition, action: 'mount' });
                    } else if (!hasShares && firstMountPath?.startsWith("/mnt/")) {
                        partitions.push({ partition, action: 'share' });
                    }
                }
            }
        }
        return partitions;
    }, [disks, read_only]);

    const handleMount = (_partition: Partition) => {
        // Navigate to the volumes tab. A more advanced implementation could pass state
        // to automatically open the mount dialog for the specific partition.
        navigate('/', { state: { tabId: TabIDs.VOLUMES } as LocationState });
    };

    const handleCreateShare = (partition: Partition) => {
        const firstMountPointData = partition.mount_point_data?.[0];
        if (firstMountPointData) {
            navigate('/', { state: { tabId: TabIDs.SHARES, newShareData: firstMountPointData } as LocationState });
        }
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
            return <Alert severity="error">Could not load volume information.</Alert>;
        }

        if (actionablePartitions.length === 0) {
            return <Typography>No actionable items at the moment.</Typography>;
        }

        return (
            <>
                <Typography variant="body2" sx={{ mb: 2 }}>
                    You have volumes that are ready for use. Take action to make them available to the system.
                </Typography>
                <List dense>
                    {actionablePartitions.map(({ partition, action }) => (
                        <ListItem key={partition.id || partition.device} secondaryAction={<Button variant="contained" size="small" onClick={() => action === 'mount' ? handleMount(partition) : handleCreateShare(partition)}>{action === 'mount' ? 'Mount' : 'Create Share'}</Button>}>
                            <ListItemText primary={decodeEscapeSequence(partition.name || partition.device || 'Unknown Partition')} secondary={action === 'mount' ? 'This partition is not mounted.' : 'This partition is mounted but not shared.'} />
                        </ListItem>
                    ))}
                </List>
            </>
        );
    };

    return (
        <Accordion
            key={isLoading ? 'loading' : 'loaded'}
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
                {renderContent()}
            </AccordionDetails>
        </Accordion>
    );
}