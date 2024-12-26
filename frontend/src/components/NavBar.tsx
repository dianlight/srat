import logo from "../img/logo.png"
import github from "../img/github.svg"
import pkg from '../../package.json'
import { useContext, useEffect, useRef, useState } from "react"
import { apiContext, GithubContext, ModeContext, wsContext } from "../Contexts"
import { MainEventType, type MainHealth, type MainSRATReleaseAsset } from "../srat"
import AppBar from '@mui/material/AppBar';
import Box from '@mui/material/Box';
import Toolbar from '@mui/material/Toolbar';
import IconButton from '@mui/material/IconButton';
import Typography from '@mui/material/Typography';
import Menu from '@mui/material/Menu';
import MenuIcon from '@mui/icons-material/Menu';
import Container from '@mui/material/Container';
import Avatar from '@mui/material/Avatar';
import Button from '@mui/material/Button';
import Tooltip from '@mui/material/Tooltip';
import AdbIcon from '@mui/icons-material/Adb';
import PreviewIcon from '@mui/icons-material/Preview';
import SystemSecurityUpdateIcon from '@mui/icons-material/SystemSecurityUpdate';
import DarkModeIcon from '@mui/icons-material/DarkMode';
import LightModeIcon from '@mui/icons-material/LightMode';
import { useColorScheme } from "@mui/material/styles"
import AutoModeIcon from '@mui/icons-material/AutoMode';
import semver from "semver"
import { CircularProgress, Tab, Tabs, type CircularProgressProps } from "@mui/material"
import { createPortal } from "react-dom"
import { Shares } from "../pages/Shares"
import { SmbConf } from "../pages/SmbConf"
import { Settings } from "../pages/Settings"
import { Volumes } from "../pages/Volumes"
import { Users } from "../pages/Users"
import { green } from "@mui/material/colors"

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
        <Box sx={{ position: 'relative', display: 'inline-flex' }}>
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
                    sx={{ color: 'text.secondary' }}
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

export function NavBar(props: { error: string, bodyRef: React.RefObject<HTMLDivElement | null>, healthData: MainHealth }) {
    const healt = useContext(ModeContext);
    const [updateAssetStatus, setUpdateAssetStatus] = useState<MainSRATReleaseAsset>({});
    const ws = useContext(wsContext);
    const api = useContext(apiContext);
    const { mode, setMode } = useColorScheme();
    const [update, setUpdate] = useState<string | undefined>()
    const [value, setValue] = useState(() => {
        return Number.parseInt(localStorage.getItem("srat_tab") || "0");
    });

    if (!mode) {
        return null;
    }

    const handleChange = (event: React.SyntheticEvent, newValue: number) => {
        setValue(newValue);
        localStorage.setItem("srat_tab", "" + newValue)

    };

    function onLoadHandler() {
        ws.subscribe<MainSRATReleaseAsset>(MainEventType.EventUpdate, (data) => {
            // console.log("Got update", data)
            setUpdateAssetStatus(data);
        })
    }

    const current = pkg.version;
    //console.log("Latest version", props.healthData?.last_release, "Current version", current)

    // Normalize Version Strings
    const currentVersion = semver.clean(current.replace(".dev", "-dev")) || "0.0.0"
    const latestVersion = semver.clean((updateAssetStatus.last_release?.tag_name || "0.0.0").replace(".dev", "-dev")) || "0.0.0"

    if (updateAssetStatus.last_release && update !== latestVersion && semver.compare(latestVersion, currentVersion) == 1) {
        setUpdate(latestVersion)
    }

    useEffect(() => {

    }, [])

    return (<>
        <AppBar position="static" onLoad={onLoadHandler}>
            <Container maxWidth="xl">
                <Toolbar disableGutters>
                    <img id="logo-container" className="brand-logo" alt="SRAT -- Samba Rest Adminitration Tool" src={logo} />
                    <Tabs
                        sx={{ flexGrow: 1, display: { xs: 'flex', md: 'flex' } }}
                        value={value}
                        onChange={handleChange}
                        indicatorColor="secondary"
                        textColor="inherit"
                        variant="fullWidth"
                        aria-label="full width tabs example"
                    >
                        <Tab label="Shares" {...a11yProps(0)} />
                        <Tab label="Volumes" {...a11yProps(1)} />
                        <Tab label="Users" {...a11yProps(2)} />
                        <Tab label="Settings" {...a11yProps(3)} />
                        <Tab label="smb.conf (ro)" {...a11yProps(4)} />
                    </Tabs>
                    <Box sx={{ flexGrow: 0 }}>
                        {healt.read_only &&
                            <IconButton>
                                <Tooltip title="ReadOnly Mode" arrow>
                                    <PreviewIcon sx={{ color: 'white' }} />
                                </Tooltip>
                            </IconButton>
                        }
                        {update && updateAssetStatus.update_status == -1 &&
                            <IconButton onClick={() => api.update.updateUpdate()}>
                                <Tooltip title={`Update ${update} available`} arrow>
                                    <SystemSecurityUpdateIcon sx={{ color: 'white' }} />
                                </Tooltip>
                            </IconButton>
                        }
                        {updateAssetStatus.update_status && updateAssetStatus.update_status != -1 &&
                            <CircularProgressWithLabel
                                value={updateAssetStatus.update_status}
                                size={68}
                                sx={{
                                    color: green[500],
                                    position: 'absolute',
                                    top: -6,
                                    left: -6,
                                    zIndex: 1,
                                }}
                            />
                        }
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
                    </Box>
                </Toolbar>
            </Container>
        </AppBar>
        {props.bodyRef.current && createPortal(<>
            <TabPanel value={value} index={0}>
                <Shares />
            </TabPanel>
            <TabPanel value={value} index={1}>
                <Volumes />
            </TabPanel>
            <TabPanel value={value} index={2}>
                <Users />
            </TabPanel>
            <TabPanel value={value} index={3}>
                <Settings />
            </TabPanel>
            <TabPanel value={value} index={4}>
                <SmbConf />
            </TabPanel></>,
            props.bodyRef.current /*document.getElementById('mainarea')!*/
        )}
    </>
    )
}