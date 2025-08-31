import { Download } from "@mui/icons-material";
import AutoModeIcon from "@mui/icons-material/AutoMode";
import BugReportIcon from "@mui/icons-material/BugReport"; // Import the BugReportIcon
import DarkModeIcon from "@mui/icons-material/DarkMode";
import HelpIcon from "@mui/icons-material/Help";
import HelpOutlineIcon from "@mui/icons-material/HelpOutline";
import LightModeIcon from "@mui/icons-material/LightMode";
import LockIcon from "@mui/icons-material/Lock";
import LockOpenIcon from "@mui/icons-material/LockOpen";
import MenuIcon from "@mui/icons-material/Menu";
import PreviewIcon from "@mui/icons-material/Preview";
import ReportProblemIcon from "@mui/icons-material/ReportProblem";
import SaveIcon from "@mui/icons-material/Save";
import SystemSecurityUpdateIcon from "@mui/icons-material/SystemSecurityUpdate";
import UndoIcon from "@mui/icons-material/Undo";
import {
	CircularProgress,
	type CircularProgressProps,
	List,
	ListItem,
	ListItemText,
	ListSubheader,
	Menu,
	MenuItem,
	Tab,
	Tabs,
	useMediaQuery,
	useTheme,
} from "@mui/material";
//import { DirtyDataContext, ModeContext } from "../Contexts"
import AppBar from "@mui/material/AppBar";
import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import IconButton from "@mui/material/IconButton";
import { useColorScheme } from "@mui/material/styles";
import Toolbar from "@mui/material/Toolbar";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useConfirm } from "material-ui-confirm";
import { useEffect, useMemo, useState } from "react"; // Added useMemo
import { createPortal } from "react-dom";
import { useLocation } from "react-router";
import { toast } from "react-toastify";
import pkg from "../../package.json";
import github from "../img/github.svg";
import icon from "../img/icon.png";
import logo from "../img/logo.png";
import { Dashboard } from "../pages/dashboard/Dashboard";
import { Settings } from "../pages/settings/Settings";
import { Shares } from "../pages/shares/Shares";
import { SmbConf } from "../pages/SmbConf";
import { Swagger } from "../pages/Swagger";
import { Users } from "../pages/users/Users";
import { Volumes } from "../pages/volumes/Volumes";
import { type LocationState, TabIDs } from "../store/locationState";
import {
	type HealthPing,
	Update_process_state,
	usePutApiSambaApplyMutation,
	usePutApiUpdateMutation,
} from "../store/sratApi";
import { ErrorBoundary } from "./ErrorBoundary";
import { NotificationCenter } from "./NotificationCenter";
import { useTour, type StepType } from '@reactour/tour'
import { DashboardSteps } from "../pages/dashboard/DashboardTourStep";
import { SharesSteps } from "../pages/shares/SharesTourStep";
import { VolumesSteps } from "../pages/volumes/VolumesTourStep";
import { SettingsSteps } from "../pages/settings/SettingsTourStep";
import { UsersSteps } from "../pages/users/UsersSteps";
import { useGetServerEventsQuery } from "../store/sseApi";

// Define tab configurations
interface TabConfig {
	id: TabIDs;
	label: string;
	component: React.ReactNode;
	isDevelopmentOnly?: boolean;
	actualIndex?: number; // Will be populated after filtering
	tutorialSteps?: StepType[]; // Optional tutorial steps for this tab
}

const NoTutorialSteps: StepType[] = [
	{
		selector: '[data-tutor="reactour__step1"]',
		content: 'Not yet implemented',
	},
];

const ALL_TAB_CONFIGS: TabConfig[] = [
	{ id: TabIDs.DASHBOARD, label: "Dashboard", component: <Dashboard />, tutorialSteps: DashboardSteps },
	{ id: TabIDs.VOLUMES, label: "Volumes", component: <Volumes />, tutorialSteps: VolumesSteps },
	{ id: TabIDs.SHARES, label: "Shares", component: <Shares />, tutorialSteps: SharesSteps },
	{ id: TabIDs.USERS, label: "Users", component: <Users />, tutorialSteps: UsersSteps },
	{ id: TabIDs.SETTINGS, label: "Settings", component: <Settings />, tutorialSteps: SettingsSteps },
	{
		id: TabIDs.SMB_FILE_CONFIG,
		label: "smb.conf",
		component: <SmbConf />,
		isDevelopmentOnly: true,
		tutorialSteps: NoTutorialSteps,
	},
	{
		id: TabIDs.API_OPENDOC,
		label: "API Docs",
		component: <Swagger />,
		isDevelopmentOnly: true,
		tutorialSteps: NoTutorialSteps,
	},
];

// Helper to get icon based on TabID and health
const getTabIcon = (tab: TabConfig, healthData: HealthPing | undefined) => {
	// Priority 1: Dirty state
	if (healthData?.dirty_tracking) {
		const dirtyMap: Partial<
			Record<TabIDs, keyof HealthPing["dirty_tracking"]>
		> = {
			[TabIDs.SHARES]: "shares",
			[TabIDs.VOLUMES]: "volumes",
			[TabIDs.USERS]: "users",
			[TabIDs.SETTINGS]: "settings",
		};
		const dirtyKey = dirtyMap[tab.id];
		if (dirtyKey && healthData.dirty_tracking[dirtyKey]) {
			return (
				<Tooltip title="Changes not yet applied!">
					<ReportProblemIcon sx={{ color: "white" }} />
				</Tooltip>
			);
		}
	}

	// Priority 2: Development only tab
	if (tab.isDevelopmentOnly) {
		return (
			<Tooltip title="Development Only Tab">
				<BugReportIcon sx={{ color: "orange" }} />
			</Tooltip>
		);
	}
	return undefined;
};
function a11yProps(index: number) {
	return {
		id: `full-width-tab-${index}`,
		"aria-controls": `full-width-tabpanel-${index}`,
	};
}

interface TabPanelProps {
	children?: React.ReactNode;
	index: number;
	value: number;
	tutorialSteps?: StepType[];
}

function CircularProgressWithLabel(
	props: CircularProgressProps & { value: number },
) {
	return (
		<Box
			sx={{
				position: "relative",
				display: "inline-flex",
				verticalAlign: "middle",
			}}
		>
			<CircularProgress variant="determinate" {...props} />
			<Box
				sx={{
					top: 0,
					left: 0,
					bottom: 0,
					right: 0,
					position: "absolute",
					display: "flex",
					alignItems: "center",
					justifyContent: "center",
				}}
			>
				<Typography
					variant="caption"
					component="div"
					sx={{ color: "primary" }}
				>{`${Math.round(props.value)}%`}</Typography>
			</Box>
		</Box>
	);
}

function TabPanel(props: TabPanelProps) {
	const { children, value, index, tutorialSteps, ...other } = props;
	const { setIsOpen: setTourOpen, isOpen: isTourOpen, setSteps } = useTour();

	useEffect(() => {
		if (value === index && isTourOpen && tutorialSteps && setSteps) {
			setSteps(tutorialSteps);
		}
	}, [isTourOpen, tutorialSteps, value, setSteps]);

	return (
		<div
			role="tabpanel"
			hidden={value !== index}
			id={`full-width-tabpanel-${index}`}
			aria-labelledby={`full-width-tab-${index}`}
			data-tutor="reactour__step1"
			{...other}
		>
			{value === index && <Box sx={{ p: 3 }}>{children}</Box>}
		</div>
	);
}

export function NavBar(props: {
	error: string;
	bodyRef: React.RefObject<HTMLDivElement | null>;
}) {
	const location = useLocation();
	const { setIsOpen: setTourOpen, isOpen: isTourOpen } = useTour();
	//const _navigate = useNavigate();

	const visibleTabs = useMemo(() => {
		return ALL_TAB_CONFIGS.filter(
			(tab) =>
				!(
					tab.isDevelopmentOnly &&
					process.env.NODE_ENV === "production" &&
					!pkg.version.match(/dev/)
				),
		).map((tab, index) => ({
			...tab,
			actualIndex: index,
		}));
	}, []); // process.env.NODE_ENV is a build-time constant

	const { data: evdata, error: everror, isLoading: evloading, fulfilledTimeStamp: evfulfilledTimeStamp } = useGetServerEventsQuery();

	const [doUpdate] = usePutApiUpdateMutation();
	const [restartSamba] = usePutApiSambaApplyMutation();

	const { mode, setMode } = useColorScheme();
	//const [update, setUpdate] = useState<string | undefined>()
	const [value, setValue] = useState<number>(0); // Active tab index, default to 0
	const confirm = useConfirm();
	//const [tabId, setTabId] = useState<string>(() => uuidv4())
	const theme = useTheme();
	const [isLogoHovered, setIsLogoHovered] = useState(false);
	const matches = useMediaQuery(theme.breakpoints.up("sm"));
	const [anchorElNav, setAnchorElNav] = useState<null | HTMLElement>(null);

	if (!mode) {
		return null;
	}

	const handleOpenNavMenu = (event: React.MouseEvent<HTMLElement>) => {
		setAnchorElNav(event.currentTarget);
	};

	const handleCloseNavMenu = () => {
		setAnchorElNav(null);
	};

	const handleMenuItemClick = (index: number) => {
		setValue(index);
		localStorage.setItem("srat_tab", index.toString());
		handleCloseNavMenu();
	};

	useEffect(() => {
		// Determine the target tab index based on priority: location.state > localStorage > default (0)
		let targetIndex = 0;
		const state = location.state as LocationState | undefined;

		if (visibleTabs.length === 0) {
			if (value !== 0) setValue(0);
			localStorage.setItem("srat_tab", "0");
			return;
		}

		if (state?.tabId !== undefined) {
			const indexFromState = visibleTabs.findIndex(
				(tab) => tab.id === state.tabId,
			);
			if (indexFromState !== -1) {
				targetIndex = indexFromState;
			} else {
				// Tab from state not found, try localStorage
				const storedIndex = parseInt(
					localStorage.getItem("srat_tab") || "0",
					10,
				);
				if (storedIndex >= 0 && storedIndex < visibleTabs.length) {
					targetIndex = storedIndex;
				} else {
					targetIndex = 0; // Default to first visible tab
				}
			}
		} else {
			const storedIndex = parseInt(localStorage.getItem("srat_tab") || "0", 10);
			if (storedIndex >= 0 && storedIndex < visibleTabs.length) {
				targetIndex = storedIndex;
			} else {
				targetIndex = 0; // Default to first visible tab
			}
		}

		if (value !== targetIndex) {
			setValue(targetIndex);
		}
		localStorage.setItem("srat_tab", targetIndex.toString());
	}, [location.state, visibleTabs, value]); // `value` is intentionally omitted to avoid loops on its own change

	const handleChange = (_event: React.SyntheticEvent, newValue: number) => {
		setValue(newValue);
		localStorage.setItem("srat_tab", newValue.toString());
	};

	function handleDoUpdate() {
		console.log("Doing update");
		confirm({
			title: `Update to ${evdata?.updating?.last_release}?`,
			description:
				"If you proceed the new version is downloaded and installed.",
		}).then(({ confirmed, reason }) => {
			if (confirmed) {
				doUpdate()
					.unwrap()
					.then((_res) => {
						//updateStatus.update_status = (res as UpdateProgress).update_status;
						//users.mutate();
					})
					.catch((err) => {
						console.error(err);
					});
			} else if (reason === "cancel") {
				console.log("cancel");
			}
		});
	}

	function handleRestartNow() {
		console.log("Doing restart");
		restartSamba();
	}

	return (
		<>
			<AppBar position="static">
				<Container maxWidth="xl">
					<Toolbar
						disableGutters
						sx={{
							minHeight: '56px !important',
							height: '56px'
						}}
					>
						{matches && (
							<img
								id="logo-container"
								className="brand-logo"
								alt="SRAT -- Samba Rest Adminitration Tool"
								src={isLogoHovered ? icon : logo}
								onMouseEnter={() => setIsLogoHovered(true)}
								onMouseLeave={() => setIsLogoHovered(false)}
								style={{ height: '40px' }}
							/>
						)}
						{matches ? (
							<Tabs
								sx={{ flexGrow: 1, maxHeight: "48px" }} // display: flex is default for Tabs root, flexGrow is key
								value={value}
								onChange={handleChange}
								indicatorColor="secondary"
								textColor="inherit"
								variant="scrollable"
								aria-label="Section Tabs"
								allowScrollButtonsMobile
								scrollButtons
							>
								{visibleTabs.map((tab) => (
									<Tab
										key={tab.id}
										data-tutor={`reactour__tab${tab.id}__step1`}
										label={tab.label}
										{...a11yProps(tab.actualIndex as number)}
										icon={getTabIcon(tab, evdata?.heartbeat)}
										iconPosition="end"
										sx={{ maxHeight: "48px", minHeight: "48px" }}
									/>
								))}
							</Tabs>
						) : (
							<Box sx={{ flexGrow: 1 }}>
								<IconButton
									size="large"
									aria-label="navigation menu"
									aria-controls="menu-appbar"
									aria-haspopup="true"
									onClick={handleOpenNavMenu}
									color="inherit"
								>
									<MenuIcon />
								</IconButton>
								<Menu
									id="menu-appbar"
									anchorEl={anchorElNav}
									open={Boolean(anchorElNav)}
									onClose={handleCloseNavMenu}
									keepMounted
									anchorOrigin={{ vertical: "bottom", horizontal: "left" }}
									transformOrigin={{ vertical: "top", horizontal: "left" }}
								>
									{visibleTabs.map((tab) => (
										<MenuItem
											key={tab.id}
											onClick={() =>
												handleMenuItemClick(tab.actualIndex as number)
											}
										>
											<Typography textAlign="center">{tab.label}</Typography>
										</MenuItem>
									))}
								</Menu>
							</Box>
						)}

						<Box sx={{ flexGrow: 0, display: "flex", alignItems: "center" }}>
							{/*
							{Object.values(evdata.health.dirty_tracking || {}).reduce(
								(acc, value) => acc + (value ? 1 : 0),
								0,
							) > 0 && (
									<Tooltip title="Restart Samba demon now!" arrow>
										<IconButton onClick={handleRestartNow} size="small">
											<RestartAltIcon sx={{ color: "white" }} />
											<CircularProgress
												size={32}
												sx={{
													color: "blueviolet",
													position: "absolute",
													zIndex: 1,
												}}
											/>
										</IconButton>
									</Tooltip>
								)}
								*/}
							{process.env.NODE_ENV !== "production" && (
								<IconButton size="small">
									<Tooltip title={
										<List
											dense
											subheader={
												<ListSubheader id="nested-list-subheader">
													Development Environment Debug
												</ListSubheader>
											}
										>
											<ListItem>
												<ListItemText
													primary="Protected Mode"
													secondary={evdata?.hello?.protected_mode ? "Enabled" : "Disabled"}
												/>
											</ListItem>
										</List>
									} arrow>
										<BugReportIcon sx={{ color: "orange" }} />
									</Tooltip>
								</IconButton>
							)}
							{!evdata?.hello?.secure_mode ? (
								<IconButton size="small">
									<Tooltip
										title="Secure Mode Disabled"
										arrow
									>
										<LockOpenIcon sx={{ color: "red" }} />
									</Tooltip>
								</IconButton>
							) : (
								<IconButton size="small">
									<Tooltip
										title="Secure Mode Enabled"
										arrow
									>
										<LockIcon sx={{ color: "white" }} />
									</Tooltip>
								</IconButton>
							)}
							{evdata?.hello?.read_only && (
								<IconButton size="small">
									<Tooltip title="ReadOnly Mode" arrow>
										<PreviewIcon sx={{ color: "white" }} />
									</Tooltip>
								</IconButton>
							)}
							{evdata?.updating?.last_release !== undefined && (
								<IconButton onClick={handleDoUpdate} size="small">
									<Tooltip
										title={`Update ${evdata.updating.last_release} available`}
										arrow
									>
										{((update_status) => {
											switch (update_status.update_process_state) {
												case Update_process_state.Checking:
													return <UndoIcon sx={{ color: "white" }} />;
												case Update_process_state.Downloading:
													return <SaveIcon sx={{ color: "white" }} />;
												case Update_process_state.Installing:
													return (
														<SystemSecurityUpdateIcon sx={{ color: "white" }} />
													);
												case Update_process_state.Error:
													toast.error("Error during update", {
														data: { error: evdata.updating.error_message },
													});
													return <BugReportIcon sx={{ color: "red" }} />;
												default:
													return <Download sx={{ color: "white" }} />;
											}
										})(evdata.updating)}
									</Tooltip>
								</IconButton>
							)}
							{evdata?.updating?.progress !== undefined ? (
								<CircularProgressWithLabel
									value={evdata.updating.progress}
									color="success"
								/>
							) : (
								<></>
							)}
							<IconButton size="small" onClick={() => setTourOpen(!isTourOpen)}>
								<Tooltip
									title={isTourOpen ? "Close Guided Tour" : "Start Guided Tour"}
									arrow
								>
									{isTourOpen ? (
										<HelpIcon sx={{ color: "white" }} />
									) : (
										<HelpOutlineIcon sx={{ color: "white" }} />
									)}
								</Tooltip>
							</IconButton>
							<IconButton
								size="small"
								onClick={() => {
									mode === "light"
										? setMode("dark")
										: mode === "dark"
											? setMode("system")
											: setMode("light");
								}}
							>
								<Tooltip title={`Switch Mode ${mode}`} arrow>
									{mode === "light" ? (
										<LightModeIcon sx={{ color: "white" }} />
									) : mode === "dark" ? (
										<DarkModeIcon sx={{ color: "white" }} />
									) : (
										<AutoModeIcon sx={{ color: "white" }} />
									)}
								</Tooltip>
							</IconButton>
							<IconButton
								sx={{ display: { xs: "none", sm: "inline-flex" } }}
								size="small"
								onClick={() => {
									window.open(pkg.repository.url);
								}}
							>
								<Tooltip title="Support project!" arrow>
									<img src={github} style={{ height: "20px" }} />
								</Tooltip>
							</IconButton>
							<NotificationCenter />
						</Box>
					</Toolbar>
				</Container>
			</AppBar>
			{props.bodyRef.current &&
				createPortal(
					visibleTabs.map((tab) => (
						<TabPanel
							key={tab.id}
							value={value}
							index={tab.actualIndex as number}
							tutorialSteps={tab.tutorialSteps}
						>
							<ErrorBoundary>{tab.component}</ErrorBoundary>
						</TabPanel>
					)),
					props.bodyRef.current /*document.getElementById('mainarea')!*/,
				)}
		</>
	);
}
