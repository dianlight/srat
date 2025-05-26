import logo from "../img/logo.png"
import icon from "../img/icon.png"
import github from "../img/github.svg"
import pkg from '../../package.json'
import { useContext, useEffect, useState } from "react"
//import { DirtyDataContext, ModeContext } from "../Contexts"
import AppBar from '@mui/material/AppBar';
import Box from '@mui/material/Box';
import Toolbar from '@mui/material/Toolbar';
import IconButton from '@mui/material/IconButton';
import Typography from '@mui/material/Typography';
import Container from '@mui/material/Container';
import Tooltip from '@mui/material/Tooltip';
import PreviewIcon from '@mui/icons-material/Preview';
import SystemSecurityUpdateIcon from '@mui/icons-material/SystemSecurityUpdate';
import DarkModeIcon from '@mui/icons-material/DarkMode';
import LightModeIcon from '@mui/icons-material/LightMode';
import { useColorScheme } from "@mui/material/styles"
import AutoModeIcon from '@mui/icons-material/AutoMode';
import semver from "semver"
import { CircularProgress, Tab, Tabs, useMediaQuery, useTheme, type CircularProgressProps } from "@mui/material"
import { createPortal } from "react-dom"
import { Shares } from "../pages/Shares"
import { SmbConf } from "../pages/SmbConf"
import { Settings } from "../pages/Settings"
import { Volumes } from "../pages/Volumes"
import { Users } from "../pages/Users"
import SaveIcon from '@mui/icons-material/Save';
import ReportProblemIcon from '@mui/icons-material/ReportProblem';
import UndoIcon from '@mui/icons-material/Undo';
import { useConfirm } from "material-ui-confirm"
import CancelIcon from '@mui/icons-material/Cancel';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import { v4 as uuidv4 } from 'uuid';
import { Swagger } from "../pages/Swagger"
import { NotificationCenter } from "./NotificationCenter"
import { useSSE } from "react-hooks-sse"
import { Supported_events, usePutSambaApplyMutation, usePutUpdateMutation, type HealthPing, type ReleaseAsset, type UpdateProgress } from "../store/sratApi"
import { useHealth } from "../hooks/healthHook";
import { useAppSelector } from "../store/store";
import { useReadOnly } from "../hooks/readonlyHook";
import LockOpenIcon from '@mui/icons-material/LockOpen';
import LockIcon from '@mui/icons-material/Lock';

function a11yProps(index: number) {
    return {
        id: `full-width-tab-${index}`,
        'aria-controls': `full-width-tabpanel-${index}`,
    };
}

interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

function CircularProgressWithLabel(
    props: CircularProgressProps & { value: number },
) {
    return (
        <Box sx={{ position: 'relative', display: 'inline-flex', verticalAlign: 'middle' }}>
            <CircularProgress variant="determinate" {...props} />
            <Box
                sx={{
                    top: 0,
                    left: 0,
                    bottom: 0,
                    right: 0,
                    position: 'absolute',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                }}
            >
                <Typography
                    variant="caption"
                    component="div"
                    sx={{ color: 'primary' }}
                >{`${Math.round(props.value)}%`}</Typography>
            </Box>
        </Box>
    );
}

function TabPanel(props: TabPanelProps) {
    const { children, value, index, ...other } = props;

    return (
        <div
            role="tabpanel"
            hidden={value !== index}
            id={`full-width-tabpanel-${index}`}
            aria-labelledby={`full-width-tab-${index}`}
            {...other}
        >
            {value === index && (
                <Box sx={{ p: 3 }}>
                    {children}
                </Box>
            )}
        </div>
    );
}

export function NavBar(props: { error: string, bodyRef: React.RefObject<HTMLDivElement | null> }) {
    const read_only = useReadOnly();
    const health = useHealth();
    //  const [sse, sseStatus] = useContext(SSEContext);

    //  const [updateAssetStatus, setUpdateAssetStatus] = useState<DtoReleaseAsset>({});

    const updateAssetStatus = useSSE(Supported_events.Update, {} as UpdateProgress, {
        parser(input: any): ReleaseAsset {
            console.log("Got version", input)
            return JSON.parse(input);
        },
    });

    const [doUpdate] = usePutUpdateMutation();
    const [restartSamba] = usePutSambaApplyMutation();


    const { mode, setMode } = useColorScheme();
    const [update, setUpdate] = useState<string | undefined>()
    const [value, setValue] = useState(() => {
        return Number.parseInt(localStorage.getItem("srat_tab") || "0");
    });
    const confirm = useConfirm();
    const [tabId, setTabId] = useState<string>(() => uuidv4())
    const theme = useTheme();
    const [isLogoHovered, setIsLogoHovered] = useState(false);
    const matches = useMediaQuery(theme.breakpoints.up('sm'));


    if (!mode) {
        return null;
    }

    function handleChange(event: React.SyntheticEvent, newValue: number) {
        setValue(newValue);
        localStorage.setItem("srat_tab", "" + newValue)
    };

    function handleDoUpdate() {
        console.log("Doing update")
        confirm({
            title: `Update to ${update}?`,
            description: "If you proceed the new version is downloaded and installed."
        })
            .then(({ confirmed, reason }) => {
                if (confirmed) {
                    doUpdate().unwrap().then((res) => {
                        updateAssetStatus.update_status = (res as UpdateProgress).update_status;
                        //users.mutate();
                    }).catch(err => {
                        console.error(err);
                    })
                } else if (reason === "cancel") {
                    console.log("cancel")
                }
            })
    }

    function handleRestartNow() {
        console.log("Doing restart")
        restartSamba()
    }

    useEffect(() => {
        const current = pkg.version;

        // Normalize Version Strings
        const currentVersion = semver.clean(current.replace(".dev", "-dev")) || "0.0.0"
        const latestVersion = semver.clean((health.health.last_release?.last_release || "0.0.0").replace(".dev", "-dev")) || "0.0.0"

        if (update !== latestVersion && semver.compare(latestVersion, currentVersion) == 1) {
            setUpdate(latestVersion)
        } else {
            setUpdate(undefined)
        }
    }, [health])

    /*
    useEventSourceListener(
        sse,
        [DtoEventType.EventHeartbeat],
        (evt) => {
            //console.log("SSE EventHeartbeat", evt);
            setUpdateAssetStatus(JSON.parse(evt.data));
        },
        [setUpdateAssetStatus],
    );
    */

    return (<>
        <AppBar position="static">
            <Container maxWidth="xl">
                <Toolbar disableGutters>
                    {matches &&
                        <img
                            id="logo-container"
                            className="brand-logo"
                            alt="SRAT -- Samba Rest Adminitration Tool"
                            src={isLogoHovered ? icon : logo}
                            onMouseEnter={() => setIsLogoHovered(true)}
                            onMouseLeave={() => setIsLogoHovered(false)} />
                    }
                    <Tabs
                        sx={{ flexGrow: 1, display: { xs: 'flex', md: 'flex' } }}
                        value={value}
                        onChange={handleChange}
                        indicatorColor="secondary"
                        textColor="inherit"
                        variant="scrollable"
                        aria-label="Section Tabs"
                        allowScrollButtonsMobile
                        scrollButtons
                    >
                        <Tab label="Shares" {...a11yProps(0)} icon={health.health?.dirty_tracking?.shares ? <Tooltip title="Changes not yet applied!"><ReportProblemIcon sx={{ color: 'white' }} /></Tooltip> : undefined} iconPosition="end" />
                        <Tab label="Volumes" {...a11yProps(1)} icon={health.health?.dirty_tracking?.volumes ? <Tooltip title="Changes not yet applied"><ReportProblemIcon sx={{ color: 'white' }} /></Tooltip> : undefined} iconPosition="end" />
                        <Tab label="Users" {...a11yProps(2)} icon={health.health?.dirty_tracking?.users ? <Tooltip title="Changes not yet applied!"><ReportProblemIcon sx={{ color: 'white' }} /></Tooltip> : undefined} iconPosition="end" />
                        <Tab label="Settings" {...a11yProps(3)} icon={health.health?.dirty_tracking?.settings ? <Tooltip title="Changes not yet applied!"><ReportProblemIcon sx={{ color: 'white' }} /></Tooltip> : undefined} iconPosition="end" />
                        <Tab label="smb.conf" {...a11yProps(4)} />
                        <Tab label="API Docs" {...a11yProps(4)} />
                    </Tabs>
                    <Box sx={{ flexGrow: 0 }}>

                        {Object.values(health.health.dirty_tracking || {}).reduce((acc, value) => acc + (value ? 1 : 0), 0) > 0 &&
                            <Tooltip title="Restart Samba demon now!" arrow>
                                <IconButton onClick={handleRestartNow}>
                                    <RestartAltIcon sx={{ color: 'white' }} />
                                    <CircularProgress
                                        size={32}
                                        sx={{
                                            color: "blueviolet",
                                            position: 'absolute',
                                            zIndex: 1,
                                        }}
                                    />
                                </IconButton>
                            </Tooltip>
                        }
                        <IconButton>
                            <Tooltip title={health.health.secure_mode ? "Secure Mode Enabled" : "Secure Mode Disabled"} arrow>
                                {health.health.secure_mode ?
                                    <LockIcon sx={{ color: 'white' }} /> :
                                    <LockOpenIcon sx={{ color: 'red' }} />}
                            </Tooltip>
                        </IconButton>
                        {read_only &&
                            <IconButton>
                                <Tooltip title="ReadOnly Mode" arrow>
                                    <PreviewIcon sx={{ color: 'white' }} />
                                </Tooltip>
                            </IconButton>
                        }
                        {(update && updateAssetStatus.update_status == undefined) ? (
                            <IconButton onClick={handleDoUpdate}>
                                <Tooltip title={`Update ${update} available`} arrow>
                                    <SystemSecurityUpdateIcon sx={{ color: 'white' }} />
                                </Tooltip>
                            </IconButton>
                        ) : (
                            updateAssetStatus.update_status != undefined ?
                                <CircularProgressWithLabel value={updateAssetStatus.update_status} color="success" />
                                :
                                <></>
                        )}
                        <IconButton onClick={() => { mode == 'light' ? setMode('dark') : (mode == 'dark' ? setMode('system') : setMode('light')) }} >
                            <Tooltip title={`Switch Mode ${mode}`} arrow>
                                {mode === 'light' ? <LightModeIcon sx={{ color: 'white' }} /> : mode === 'dark' ? <DarkModeIcon sx={{ color: 'white' }} /> : <AutoModeIcon sx={{ color: 'white' }} />}
                            </Tooltip>
                        </IconButton>
                        <IconButton onClick={() => { window.open(pkg.repository.url) }} >
                            <Tooltip title="Support project!" arrow>
                                <img src={github} style={{ height: "20px" }} />
                            </Tooltip>
                        </IconButton>
                        <NotificationCenter />
                    </Box>
                </Toolbar>
            </Container>
        </AppBar>
        {props.bodyRef.current && createPortal(
            <>
                <TabPanel key={tabId + "0"} value={value} index={0}>
                    <Shares />
                </TabPanel>
                <TabPanel key={tabId + "1"} value={value} index={1}>
                    <Volumes />
                </TabPanel>
                <TabPanel key={tabId + "2"} value={value} index={2}>
                    <Users />
                </TabPanel>
                <TabPanel key={tabId + "3"} value={value} index={3}>
                    <Settings />
                </TabPanel>
                <TabPanel key={tabId + "4"} value={value} index={4}>
                    <SmbConf />
                </TabPanel>
                <TabPanel key={tabId + "5"} value={value} index={5}>
                    <Swagger />
                </TabPanel>
            </>,
            props.bodyRef.current /*document.getElementById('mainarea')!*/
        )}
    </>
    )
}
