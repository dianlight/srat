import { Fragment, useContext, useEffect, useState } from "react";
import { ModeContext, wsContext as ws } from "../Contexts";
import { MainEventType, type BlockDisk, type BlockInfo, type BlockPartition } from "../srat";
import { InView } from "react-intersection-observer";
import { ObjectTable, PreviewDialog } from "../components/PreviewDialog";
import Fab from "@mui/material/Fab";
import List from "@mui/material/List";
import { ListItemButton, ListItem, IconButton, ListItemAvatar, Avatar, ListItemText, Divider, Stack, Typography } from "@mui/material";
import AddIcon from '@mui/icons-material/Add';
import EjectIcon from '@mui/icons-material/Eject';
import StorageIcon from '@mui/icons-material/Storage';
import CreditScoreIcon from '@mui/icons-material/CreditScore';
import { useConfirm } from "material-ui-confirm";
import { filesize } from "filesize";
import { faHardDrive, faPlug, faPlugCircleCheck, faPlugCircleXmark } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeSvgIcon } from "../components/FontAwesomeSvgIcon";


export function Volumes() {
    const mode = useContext(ModeContext);
    const [showPreview, setShowPreview] = useState<boolean>(false);


    const [status, setStatus] = useState<BlockInfo>({});
    const [selected, setSelected] = useState<BlockDisk | BlockPartition | null>(null);
    const confirm = useConfirm();


    useEffect(() => {
        const vol = ws.subscribe<BlockInfo>(MainEventType.EventVolumes, (data) => {
            console.log("Got volumes", data)
            setStatus(data);
        })
        return () => {
            ws.unsubscribe(vol);
        };
    }, [])

    function decodeEscapeSequence(source: string) {
        return source.replace(/\\x([0-9A-Fa-f]{2})/g, function () {
            return String.fromCharCode(parseInt(arguments[1], 16));
        });
    };

    function onSubmitEjectVolume(data?: string) {
        console.log("Eject", data)
        if (!data) return
        confirm({
            title: `Eject ${data}?`,
            description: "Do you really want eject the Volume?"
        })
            .then(() => {
                /*
                api.share.shareDelete(data).then((res) => {
                    setSelected(null);
                    //users.mutate();
                }).catch(err => {
                    console.error(err);
                    //setErrorInfo(JSON.stringify(err));
                })
                */
            })
            .catch(() => {
                /* ... */
            });
    }


    return <InView>
        <PreviewDialog title={selected?.name || ""} objectToDisplay={selected} open={showPreview} onClose={() => { setSelected(null); setShowPreview(false) }} />
        {mode.read_only || <Fab color="primary" aria-label="add" sx={{
            float: 'right',
            top: '-20px',
            margin: '-8px'
        }} size="small"
            onClick={() => { setSelected(null); /*setShowEdit(true)*/ }}
        >
            <AddIcon />
        </Fab>}
        <br />
        <List dense={true}>
            {status.disks?.filter((block) => !block.name?.match("z{0,1}ram\\d+")).map((disk, idx) =>
                <Fragment key={idx}>
                    <ListItemButton>
                        <ListItem
                            secondaryAction={!mode.read_only && <>
                                {/*
                                <IconButton onClick={() => { setSelected(volume);setShowEdit(true) }} edge="end" aria-label="settings">
                                    <SettingsIcon />
                                </IconButton>
                            */}
                                <IconButton onClick={() => onSubmitEjectVolume(disk.name)} edge="end" aria-label="delete" disabled={!disk.removable}>
                                    <EjectIcon />
                                </IconButton>
                            </>
                            }
                        >
                            <ListItemAvatar>
                                <Avatar>
                                    <FontAwesomeSvgIcon icon={faHardDrive} />
                                </Avatar>
                            </ListItemAvatar>
                            <ListItemText
                                primary={disk.model}
                                onClick={() => { setSelected(disk); setShowPreview(true) }}
                                disableTypography
                                secondary={<Stack spacing={2} direction="row">
                                    <Typography variant="caption">Size: {(disk.size_bytes && filesize(disk.size_bytes, { round: 0 }))}</Typography>
                                    <Typography variant="caption">Type: {disk.drive_type}</Typography>
                                    <Typography variant="caption">Bus: {disk.storage_controller}</Typography>
                                    <Typography variant="caption">Vendor: {disk.vendor}</Typography>
                                    <Typography variant="caption">SN: {disk.serial_number}</Typography>
                                    <Typography variant="caption">Dev: {disk.name}</Typography>
                                </Stack>}
                            />
                        </ListItem>
                    </ListItemButton>
                    <List disablePadding>
                        {disk.partitions?.filter((part) => !(part.label?.startsWith("hassos-") && part.mount_point === "")).map((partition, idx) =>
                            <ListItemButton sx={{ pl: 4 }} key={idx}>
                                <ListItem
                                    secondaryAction={!mode.read_only && <>
                                        {/*
                                <IconButton onClick={() => { setSelected(volume);setShowEdit(true) }} edge="end" aria-label="settings">
                                    <SettingsIcon />
                                </IconButton>
                            */}
                                        <IconButton onClick={() => onSubmitEjectVolume(partition.name)} edge="end" aria-label="delete" disabled={!disk.removable}>
                                            <FontAwesomeSvgIcon icon={faPlug} />
                                        </IconButton>
                                        <IconButton onClick={() => onSubmitEjectVolume(partition.name)} edge="end" aria-label="delete" disabled={!disk.removable}>
                                            <FontAwesomeSvgIcon icon={faPlugCircleCheck} />
                                        </IconButton>
                                        <IconButton onClick={() => onSubmitEjectVolume(partition.name)} edge="end" aria-label="delete" disabled={!disk.removable}>
                                            <FontAwesomeSvgIcon icon={faPlugCircleXmark} />
                                        </IconButton>
                                    </>
                                    }
                                >
                                    <ListItemAvatar>
                                        <Avatar>
                                            {partition.label === 'hassos-data' ? <CreditScoreIcon /> : <StorageIcon />}
                                        </Avatar>
                                    </ListItemAvatar>
                                    <ListItemText
                                        primary={decodeEscapeSequence((partition.label === "unknown" ? partition.filesystem_label : partition.label) || "unknown")}
                                        onClick={() => { setSelected(partition); setShowPreview(true) }}
                                        disableTypography
                                        secondary={<Stack spacing={2} direction="row">
                                            <Typography variant="caption">Size: {(partition.size_bytes && filesize(partition.size_bytes, { round: 0 }))}</Typography>
                                            <Typography variant="caption">Type: {partition.type}</Typography>
                                            <Typography variant="caption">MountPath: {partition.mount_point}</Typography>
                                            <Typography variant="caption">UUID: {partition.uuid}</Typography>
                                            <Typography variant="caption">Dev: {partition.name}</Typography>
                                        </Stack>}
                                    />
                                </ListItem>
                            </ListItemButton>
                        )}
                        <Divider />
                    </List>
                    <Divider component="li" />
                </Fragment>
            )}
        </List>
    </InView>
}