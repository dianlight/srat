import { Fragment, useContext, useEffect, useState } from "react";
import { apiContext, ModeContext, wsContext as ws } from "../Contexts";
import { DtoEventType, DtoMounDataFlag, type DtoBlockInfo, type DtoBlockPartition, type DtoMountPointData } from "../srat";
import { InView } from "react-intersection-observer";
import { ObjectTable, PreviewDialog } from "../components/PreviewDialog";
import Fab from "@mui/material/Fab";
import List from "@mui/material/List";
import { ListItemButton, ListItem, IconButton, ListItemAvatar, Avatar, ListItemText, Divider, Stack, Typography, Tooltip, Dialog, Button, DialogActions, DialogContent, DialogContentText, DialogTitle, Grid2, Autocomplete } from "@mui/material";
import AddIcon from '@mui/icons-material/Add';
import EjectIcon from '@mui/icons-material/Eject';
import StorageIcon from '@mui/icons-material/Storage';
import CreditScoreIcon from '@mui/icons-material/CreditScore';
import { useConfirm } from "material-ui-confirm";
import { filesize } from "filesize";
import { faHardDrive, faPlug, faPlugCircleCheck, faPlugCircleXmark, faPlugCircleMinus } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeSvgIcon } from "../components/FontAwesomeSvgIcon";
import { Api } from "@mui/icons-material";
import { AutocompleteElement, PasswordElement, PasswordRepeatElement, TextFieldElement, useForm } from "react-hook-form-mui";
import useSWR from "swr";
import { toast } from "react-toastify";


export function Volumes() {
    const mode = useContext(ModeContext);
    const [showPreview, setShowPreview] = useState<boolean>(false);
    const [showMount, setShowMount] = useState<boolean>(false);


    const [status, setStatus] = useState<DtoBlockInfo>({});
    const [selected, setSelected] = useState<DtoBlockPartition | undefined>(undefined);
    const confirm = useConfirm();


    useEffect(() => {
        const vol = ws.subscribe<DtoBlockInfo>(DtoEventType.EventVolumes, (data) => {
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

    function onSubmitMountVolume(data?: DtoMountPointData) {
        if (!data || !data.name) return
        console.log("Mount", data)
        apiContext.volume.mountCreate(data.name, {}).then((res) => {
            toast.info(`Volume ${res.data.label} mounted successfully.`);
            setSelected(undefined);
        }).catch(err => {
            console.error(err);
            toast.error(`Erroe mountig ${data.label}: ${err}`, { data: { error: err } });
            //setErrorInfo(JSON.stringify(err));
        })
    }

    function onSubmitUmountVolume(data: DtoBlockPartition, force = false) {
        console.log("Umount", data)
        confirm({
            title: `Umount ${data.label}?`,
            description: `Do you really want umount ${force ? "forcefull" : ""} the Volume ${data.name}?`
        })
            .then(() => {
                if (!data.name) return
                apiContext.volume.mountDelete(data.name, {
                    force,
                    lazy: !force,
                }).then((res) => {
                    setSelected(undefined);
                    toast.info(`Volume ${data.label} umounted successfully.`);
                }).catch(err => {
                    console.error(err);
                    toast.error(`Erroe umountig ${data.label}: ${err}`, { data: { error: err } });
                    //setErrorInfo(JSON.stringify(err));
                })
            })
            .catch(() => {
                /* ... */
            });
    }


    return <InView>
        <VolumeMountDialog objectToEdit={selected} open={showMount} onClose={(data) => { setSelected({}); onSubmitMountVolume(data); setShowMount(false) }} />
        <PreviewDialog title={selected?.name || ""} objectToDisplay={selected} open={showPreview} onClose={() => { setSelected(undefined); setShowPreview(false) }} />
        <br />
        <List dense={true}>
            <Divider />
            {status.partitions?.map((partition, idx) =>
                <Fragment key={idx}>
                    <ListItemButton key={idx}>
                        <ListItem
                            secondaryAction={!mode.read_only && <>
                                {partition.mount_point === "" &&
                                    <Tooltip title="Mount disk">
                                        <IconButton onClick={() => { setSelected(partition); setShowMount(true) }} edge="end" aria-label="delete">
                                            <FontAwesomeSvgIcon icon={faPlug} />
                                        </IconButton>
                                    </Tooltip>
                                }
                                {partition.mount_point !== "" && partition.mount_point?.startsWith("/mnt/") && <>
                                    <Tooltip title="Unmount disk">
                                        <IconButton onClick={() => onSubmitUmountVolume(partition, false)} edge="end" aria-label="delete">
                                            <FontAwesomeSvgIcon icon={faPlugCircleMinus} />
                                        </IconButton>
                                    </Tooltip>
                                    <Tooltip title="Force unmounting disk">

                                        <IconButton onClick={() => onSubmitUmountVolume(partition, true)} edge="end" aria-label="delete">
                                            <FontAwesomeSvgIcon icon={faPlugCircleXmark} />
                                        </IconButton>
                                    </Tooltip>
                                </>
                                }
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
                                    {partition.mount_point !== '' && <Typography variant="caption">MountPath: {partition.mount_point}</Typography>}
                                    {partition.uuid !== 'unknown' && <Typography variant="caption">UUID: {partition.uuid}</Typography>}
                                    <Typography variant="caption">Dev: {partition.name}</Typography>
                                </Stack>}
                            />
                        </ListItem>
                    </ListItemButton>

                    <Divider />

                </Fragment>
            )}
        </List>
    </InView >
}


function VolumeMountDialog(props: { open: boolean, onClose: (data?: DtoMountPointData) => void, objectToEdit?: DtoBlockPartition }) {
    const mountpointData: DtoMountPointData = {}
    const { control, handleSubmit, watch, formState: { errors } } = useForm<DtoMountPointData>(
        {
            values: {
                name: props.objectToEdit?.name,
                fstype: props.objectToEdit?.type,
                flags: props.objectToEdit?.partition_flags,
                data: props.objectToEdit?.mount_data,
            }
        },
    );
    const filesystems = useSWR<string[]>('/filesystems', () => apiContext.filesystems.filesystemsList().then(res => res.data));

    function handleCloseSubmit(data?: DtoMountPointData) {
        props.onClose(data)
    }

    //console.log("MountpointData", props.objectToEdit)

    return (
        <Fragment>
            <Dialog
                open={props.open}
                onClose={() => handleCloseSubmit()}
            >
                <DialogTitle>
                    Mount Disk {props.objectToEdit?.label} ({props.objectToEdit?.name})
                </DialogTitle>
                <DialogContent>
                    <Stack spacing={2}>
                        <DialogContentText>
                            Plese enter the mount point for the disk or change default configuration settings
                        </DialogContentText>
                        <form id="mountvolumeform" onSubmit={handleSubmit(handleCloseSubmit)} noValidate>
                            <Grid2 container spacing={2}>
                                <Grid2 size={6}>
                                    <AutocompleteElement name="fstype" label="File System Type"
                                        required control={control}
                                        options={filesystems.data || []}
                                    />
                                </Grid2>
                                <Grid2 size={6}>
                                    <AutocompleteElement
                                        multiple
                                        name="flags"
                                        label="Mount Flags"
                                        options={Object.values(DtoMounDataFlag).filter((v) => typeof v === "string") as string[]}
                                        control={control}
                                    />
                                </Grid2>
                                <Grid2 size={12}>
                                    <TextFieldElement name="data" label="Additional Mount Data" control={control} sx={{ display: "flex" }} />
                                </Grid2>

                            </Grid2>
                        </form>
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => handleCloseSubmit(mountpointData)}>Cancel</Button>
                    <Button type="submit" form="mountvolumeform">Mount</Button>
                </DialogActions>
            </Dialog>
        </Fragment>
    );
}
