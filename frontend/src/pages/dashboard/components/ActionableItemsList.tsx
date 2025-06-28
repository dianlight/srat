
import { Box, Button, CircularProgress, List, ListItem, ListItemText, Typography, Alert } from "@mui/material";
import { useNavigate } from "react-router-dom";
import { TabIDs, type LocationState } from "../../../store/locationState";
import { type Partition } from "../../../store/sratApi";
import { decodeEscapeSequence } from "../metrics/utils";

interface ActionableItemsListProps {
    actionablePartitions: { partition: Partition, action: 'mount' | 'share' }[];
    isLoading: boolean;
    error: Error | null | undefined | {};
}

export function ActionableItemsList({ actionablePartitions, isLoading, error }: ActionableItemsListProps) {
    const navigate = useNavigate();

    const handleMount = (_partition: Partition) => {
        navigate('/', { state: { tabId: TabIDs.VOLUMES } as LocationState });
    };

    const handleCreateShare = (partition: Partition) => {
        const firstMountPointData = partition.mount_point_data?.[0];
        if (firstMountPointData) {
            navigate('/', { state: { tabId: TabIDs.SHARES, newShareData: firstMountPointData } as LocationState });
        }
    };

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
}
