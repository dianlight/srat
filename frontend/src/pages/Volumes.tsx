import { Fragment, useContext, useEffect, useState } from "react";
import { apiContext, ModeContext, wsContext } from "../Contexts";
import type { MainVolume } from "../srat";
import { InView } from "react-intersection-observer";
import { ObjectTable, PreviewDialog } from "../components/PreviewDialog";
import Fab from "@mui/material/Fab";
import List from "@mui/material/List";
import { ListItemButton, ListItem, IconButton, ListItemAvatar, Avatar, ListItemText, Divider, Stack, Typography } from "@mui/material";
import AddIcon from '@mui/icons-material/Add';
import SettingsIcon from '@mui/icons-material/Settings';
import EjectIcon from '@mui/icons-material/Eject';
import StorageIcon from '@mui/icons-material/Storage';
import CreditScoreIcon from '@mui/icons-material/CreditScore';
import { useConfirm } from "material-ui-confirm";


export function Volumes() {
    const api = useContext(apiContext);
    const mode = useContext(ModeContext);
    const [showPreview, setShowPreview] = useState<boolean>(false);


    const [status, setStatus] = useState<MainVolume[]>([]);
    const [selected, setSelected] = useState<MainVolume | null>(null);
    const ws = useContext(wsContext);
    const confirm = useConfirm();


    useEffect(() => {
        ws.subscribe<MainVolume[]>('volumes', (data) => {
            console.log("Got volumes", data)
            setStatus(data);
        })
    }, [])

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
        <PreviewDialog title={selected?.label || ""} objectToDisplay={selected} open={showPreview} onClose={() => { setSelected(null); setShowPreview(false) }} />
        {mode.read_only || <Fab color="primary" aria-label="add" sx={{
            position: 'fixed',
            top: 70,
            right: 16
        }} size="small"
            onClick={() => { setSelected(null); /*setShowEdit(true)*/ }}
        >
            <AddIcon />
        </Fab>}

        <List dense={true}>
            {status.filter((vol => vol.fstype !== 'erofs')).map((volume, idx) =>
                <Fragment key={idx}>
                    <ListItemButton>
                        <ListItem
                            secondaryAction={!mode.read_only && <>
                                <IconButton onClick={() => { setSelected(volume);/* setShowEdit(true) */ }} edge="end" aria-label="settings">
                                    <SettingsIcon />
                                </IconButton>
                                <IconButton onClick={() => onSubmitEjectVolume(volume.device)} edge="end" aria-label="delete" disabled={!volume.lsbk?.rm}>
                                    <EjectIcon />
                                </IconButton>
                            </>
                            }
                        >
                            <ListItemAvatar>
                                <Avatar>
                                    {volume.label === 'hassos-data' ? <CreditScoreIcon /> : <StorageIcon />}
                                </Avatar>
                            </ListItemAvatar>
                            <ListItemText
                                primary={volume.label + " (" + volume.fstype + ")"}
                                onClick={() => { setSelected(volume); setShowPreview(true) }}
                                disableTypography
                                secondary={<Stack spacing={2} direction="row">
                                    <Typography variant="caption">MountPath: {volume.mountpoint}</Typography>
                                    <Typography variant="caption">SN: {volume.serial_number}</Typography>
                                    <Typography variant="caption">Dev: {volume.device}</Typography>
                                </Stack>}
                            />

                        </ListItem>
                    </ListItemButton>
                    <Divider component="li" />
                </Fragment>
            )}
        </List>
    </InView>


    return <>
        <div id="volume" className="modal">
            <div className="modal-content">
                <h4>{selected?.label} ({selected?.device})</h4>
                <p>Volume Attributes:</p>
                <ObjectTable object={selected} />
            </div>
            <div className="modal-footer">
                <a href="#!" className="modal-close waves-effect btn-flat">Close</a>
            </div>
        </div>
        <ul className="collection" >
            {status.filter((vol => vol.fstype !== 'erofs')).map((volume, idx) =>
                < li className="collection-item avatar" key={idx} >
                    <i className="material-icons circle green">{volume.label === 'hassos-data' ? "lock" : "hard_disk"}</i>
                    <span className="title"><a href="#volume" onClick={() => setSelected(volume)} className="modal-trigger">{volume.label}</a></span>
                    <div className="row" >
                        <p className="col s1">FS: {volume.fstype}</p>
                        <p className="col s1">MountPath: {volume.mountpoint}</p>
                        <p className="col s1">SN: {volume.serial_number}</p>
                        <p className="col s1">Dev: {volume.device}</p>
                    </div>
                    {mode.read_only ||
                        <div className="row secondary-content">
                            <div className="col offset-s9 s1"><a href="#edituser" className="btn-floating blue waves-light red modal-trigger"> <i className="material-icons"> share </i></a></div>
                            {volume.lsbk?.rm ? <div className="col s1"><a href="#edituser" className="btn-floating blue waves-light red modal-trigger"> <i className="material-icons"> eject </i></a></div> : ""}
                            <div className="col s1"><a href="#deluser" className="btn-floating waves-effect waves-light red modal-trigger"> <i className="material-icons">share</i></a></div>
                        </div>
                    }
                </li>
            )}
        </ul>
    </>
}