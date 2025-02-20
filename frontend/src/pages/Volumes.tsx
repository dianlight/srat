import { Fragment, useContext, useEffect, useRef, useState } from "react";
import { ModeContext } from "../Contexts";
import { InView } from "react-intersection-observer";
import { ObjectTable, PreviewDialog } from "../components/PreviewDialog";
import Fab from "@mui/material/Fab";
import List from "@mui/material/List";
import { ListItemButton, ListItem, IconButton, ListItemAvatar, Avatar, ListItemText, Divider, Stack, Typography, Tooltip, Dialog, Button, DialogActions, DialogContent, DialogContentText, DialogTitle, Grid2, Autocomplete, TextField, Input } from "@mui/material";
import ShareIcon from '@mui/icons-material/Share';
import AddIcon from '@mui/icons-material/Add';
import EjectIcon from '@mui/icons-material/Eject';
import StorageIcon from '@mui/icons-material/Storage';
import CreditScoreIcon from '@mui/icons-material/CreditScore';
import { useConfirm } from "material-ui-confirm";
import { filesize } from "filesize";
import { faHardDrive, faPlug, faPlugCircleCheck, faPlugCircleXmark, faPlugCircleMinus } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeSvgIcon } from "../components/FontAwesomeSvgIcon";
import { Api, DiscFull } from "@mui/icons-material";
import { AutocompleteElement, Controller, PasswordElement, PasswordRepeatElement, TextFieldElement, useForm } from "react-hook-form-mui";
import { toast } from "react-toastify";
import { useSSE } from "react-hooks-sse";
import { useVolume } from "../hooks/volumeHook";
import { useReadOnly } from "../hooks/readonlyHook";
import { DtoMounDataFlag, useDeleteVolumeByIdMountMutation, useGetFilesystemsQuery, usePostVolumeByIdMountMutation, type DtoBlockPartition, type DtoMountPointData } from "../store/sratApi";


export function Volumes() {
    const read_only = useReadOnly();
    const [showPreview, setShowPreview] = useState<boolean>(false);
    const [showMount, setShowMount] = useState<boolean>(false);

    const { volumes, isLoading, error } = useVolume();
    const [selected, setSelected] = useState<DtoBlockPartition | undefined>(undefined);
    const confirm = useConfirm();
    const [mountVolume, mountVolumeResult] = usePostVolumeByIdMountMutation();
    const [umountVolume, umountVolumeResult] = useDeleteVolumeByIdMountMutation();



    function decodeEscapeSequence(source: string) {
        return source.replace(/\\x([0-9A-Fa-f]{2})/g, function () {
            return String.fromCharCode(parseInt(arguments[1], 16));
        });
    };

    function onSubmitMountVolume(data?: DtoMountPointData) {
        console.log("Mount", data)
        if (!data || !data.id) return
        mountVolume({ id: data.id, dtoMountPointData: data }).unwrap().then((res) => {
            toast.info(`Volume ${res.path} mounted successfully.`);
            setSelected(undefined);
        }).catch(err => {
            console.error("Error:", err, err.data);
            toast.error(`${err.data.code}:${err.data.message}`, { data: { error: err } });
        })
    }

    function shareExists(mountPoint: string) {
        //return shares.some(share => share.path === mountPoint);
        return true;
    }

    function handleCreateShare(partition: DtoBlockPartition) {
        //navigate(`/shares/create?path=${partition.mount_point}`);
    }

    function handleGoToShare(partition: DtoBlockPartition) {
    }

    function onSubmitUmountVolume(data: DtoBlockPartition, force = false) {
        console.log("Umount", data)
        confirm({
            title: `Umount ${data.label}?`,
            description: `Do you really want umount ${force ? "forcefull" : ""} the Volume ${data.name}?`
        })
            .then(() => {
                if (!data.mount_point_data?.id) return
                umountVolume({
                    id: data.mount_point_data?.id,
                    force: force,
                    lazy: true,
                }).unwrap().then((res) => {
                    setSelected(undefined);
                    toast.info(`Volume ${data.label} unmounted successfully.`);
                }).catch(err => {
                    console.error(err);
                    toast.error(`Error unmounting ${data.label}: ${err}`, { data: { error: err } });
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
            {volumes.partitions?.map((partition, idx) =>
                <Fragment key={idx}>
                    <ListItemButton key={idx}>
                        <ListItem
                            secondaryAction={!read_only && <>
                                {partition.mount_point === "" &&
                                    <Tooltip title="Mount disk">
                                        <IconButton onClick={() => { setSelected(partition); setShowMount(true) }} edge="end" aria-label="mount">
                                            <FontAwesomeSvgIcon icon={faPlug} />
                                        </IconButton>
                                    </Tooltip>
                                }
                                {partition.mount_point !== "" && partition.mount_point?.startsWith("/mnt/") && <>
                                    <Tooltip title="Unmount disk">
                                        <IconButton onClick={() => onSubmitUmountVolume(partition, false)} edge="end" aria-label="unmount">
                                            <FontAwesomeSvgIcon icon={faPlugCircleMinus} />
                                        </IconButton>
                                    </Tooltip>
                                    <Tooltip title="Force unmounting disk">
                                        <IconButton onClick={() => onSubmitUmountVolume(partition, true)} edge="end" aria-label="force unmount">
                                            <FontAwesomeSvgIcon icon={faPlugCircleXmark} />
                                        </IconButton>
                                    </Tooltip>
                                    {!shareExists(partition.mount_point) && (
                                        <Tooltip title="Create Share">
                                            <IconButton onClick={() => handleCreateShare(partition)} edge="end" aria-label="create share">
                                                <AddIcon />
                                            </IconButton>
                                        </Tooltip>
                                    )}
                                    {shareExists(partition.mount_point) && (
                                        <Tooltip title="Go to Share">
                                            <IconButton onClick={() => handleGoToShare(partition)} edge="end" aria-label="go to share">
                                                <ShareIcon />
                                            </IconButton>
                                        </Tooltip>
                                    )}
                                </>
                                }
                            </>}
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

interface MountPointData extends DtoMountPointData {
    flagsNames?: string[];
}


function VolumeMountDialog(props: { open: boolean, onClose: (data?: DtoMountPointData) => void, objectToEdit?: DtoBlockPartition }) {
    const mountpointData: MountPointData = {}
    const { control, handleSubmit, watch, formState: { errors } } = useForm<MountPointData>(
        {
            values: {
                id: props.objectToEdit?.mount_point_data?.id,
                fstype: props.objectToEdit?.type,
                flags: props.objectToEdit?.mount_point_data?.flags,
                flagsNames: props.objectToEdit?.mount_point_data?.flags?.map(v => DtoMounDataFlag[v]) || [],
            },
        },
    );
    const { data: filesystems, isLoading, error } = useGetFilesystemsQuery()

    function handleCloseSubmit(data?: MountPointData) {
        if (data) {
            data.flags = data.flagsNames?.map(v => Object.values(DtoMounDataFlag)
                .filter(v2 => !(typeof v2 === 'string')).find((v1, ix) => {
                    // console.log(v1, v, DtoMounDataFlag[v1])
                    return DtoMounDataFlag[v1] === v
                })
            ).filter(v3 => v3 !== undefined);
        }
        console.log("Close", data)
        props.onClose(data)
    }

    return (
        <Fragment>
            <Dialog
                open={props.open}
                onClose={() => handleCloseSubmit()}
            >
                <DialogTitle>
                    {props.objectToEdit?.mount_point_data?.id} Mount Disk {props.objectToEdit?.label} ({props.objectToEdit?.name})
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
                                        options={filesystems || []}
                                    />
                                </Grid2>
                                <Grid2 size={6}>
                                    <AutocompleteElement
                                        multiple
                                        name="flagsNames"
                                        label="Mount Flags"
                                        options={Object.values(DtoMounDataFlag).filter((v) => typeof v === "string") as string[]}
                                        control={control}
                                    />
                                </Grid2>
                                {/*
                                <Grid2 size={12}>
                                    <TextFieldElement name="data" label="Additional Mount Data" control={control} sx={{ display: "flex" }} />
                                </Grid2>
                                */}
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
