import AppsIcon from "@mui/icons-material/Apps";
import DevicesIcon from "@mui/icons-material/Devices";
import HomeIcon from "@mui/icons-material/Home";
import InsightsIcon from "@mui/icons-material/Insights";
import LanIcon from "@mui/icons-material/Lan";
import MenuIcon from "@mui/icons-material/Menu";
import SearchIcon from "@mui/icons-material/Search";
import SecurityIcon from "@mui/icons-material/Security";
import TuneIcon from "@mui/icons-material/Tune";
import {
  Box,
  Drawer,
  IconButton,
  InputAdornment,
  Paper,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import Button from "@mui/material/Button";
import Divider from "@mui/material/Divider";
import { SimpleTreeView } from "@mui/x-tree-view/SimpleTreeView";
import { TreeItem } from "@mui/x-tree-view/TreeItem";
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { InView } from "react-intersection-observer";
import { TabIDs } from "../../store/locationState";
import {
  type Settings as ApiSettings,
  useGetApiSettingsQuery,
  usePutApiSettingsMutation,
} from "../../store/sratApi";
import { useGetServerEventsQuery } from "../../store/wsApi";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import { AppConfigurationPanel } from "./AppConfigurationPanel";
import { GeneralPanel } from "./panels/GeneralPanel";
import { HomeAssistantPanel } from "./panels/HomeAssistantPanel";
import { NetworkAccessControlPanel } from "./panels/NetworkAccessControlPanel";
import { NetworkDevicesPanel } from "./panels/NetworkDevicesPanel";
import { TelemetryPanel } from "./panels/TelemetryPanel";
import { buildSettingsTree, type SettingTreeNode } from "./settingsConfig";

const SETTINGS_STORAGE_KEY = "srat_settings_selected";
const SETTINGS_EXPANDED_KEY = "srat_settings_expanded";
const DEFAULT_EXPANDED = ["network", "update", "telemetry", "hdidle"];

export function Settings() {
  const [selectedSetting, setSelectedSetting] = useState<string | null>(() => {
    return localStorage.getItem(SETTINGS_STORAGE_KEY) || "general";
  });
  const [searchQuery, setSearchQuery] = useState<string>("");
  const [expandedNodes, setExpandedNodes] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(SETTINGS_EXPANDED_KEY);
      return stored ? (JSON.parse(stored) as string[]) : DEFAULT_EXPANDED;
    } catch {
      return DEFAULT_EXPANDED;
    }
  });
  const [mobileTreeOpen, setMobileTreeOpen] = useState<boolean>(false);

  const settingsTree = useMemo(() => buildSettingsTree(), []);

  const filteredTree = useMemo(() => {
    if (!searchQuery.trim()) return settingsTree;

    const filterNode = (node: SettingTreeNode): SettingTreeNode | null => {
      const matchesSearch =
        node.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
        node.settingName?.toLowerCase().includes(searchQuery.toLowerCase());
      if (matchesSearch) return node;
      if (node.children) {
        const filteredChildren = node.children
          .map(filterNode)
          .filter((child): child is SettingTreeNode => child !== null);
        if (filteredChildren.length > 0) {
          return { ...node, children: filteredChildren };
        }
      }
      return null;
    };

    return settingsTree
      .map(filterNode)
      .filter((node): node is SettingTreeNode => node !== null);
  }, [settingsTree, searchQuery]);

  const { data: evdata } = useGetServerEventsQuery();
  const { data: globalConfig } = useGetApiSettingsQuery();
  const readOnly = Boolean(evdata?.hello?.read_only);

  const methods = useForm({
    mode: "onBlur",
    values: globalConfig as ApiSettings,
    disabled: readOnly,
  });
  const { handleSubmit, reset, formState } = methods;
  const [update, _updateResponse] = usePutApiSettingsMutation();

  function handleCommit(data: ApiSettings) {
    console.debug("Settings commit:", data);
    update({ settings: data })
      .unwrap()
      .then((res) => {
        reset(res as ApiSettings);
      })
      .catch((err) => {
        console.error("Settings update error:", err);
        reset();
      });
  }

  const handleSelectSetting = (settingName?: string) => {
    if (!settingName) return;
    setSelectedSetting(settingName);
    localStorage.setItem(SETTINGS_STORAGE_KEY, settingName);
    setMobileTreeOpen(false);
  };

  const renderTreeLabel = (node: SettingTreeNode) => {
    const iconProps = {
      fontSize: "small" as const,
      sx: { color: "text.secondary" },
    };
    let icon: ReactNode = null;
    switch (node.id) {
      case "general":
        icon = <TuneIcon {...iconProps} />;
        break;
      case "network":
        icon = <LanIcon {...iconProps} />;
        break;
      case "network_devices":
        icon = <DevicesIcon {...iconProps} />;
        break;
      case "network_access_control":
        icon = <SecurityIcon {...iconProps} />;
        break;
      case "telemetry":
        icon = <InsightsIcon {...iconProps} />;
        break;
      case "homeassistant":
        icon = <HomeIcon {...iconProps} />;
        break;
      case "app_configuration":
        icon = <AppsIcon {...iconProps} />;
        break;
      default:
        icon = null;
    }
    return (
      <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
        {icon}
        <Typography variant="body2">{node.label}</Typography>
      </Stack>
    );
  };

  const renderTree = (node: SettingTreeNode) => (
    <TreeItem
      key={node.id}
      itemId={node.id}
      label={renderTreeLabel(node)}
      onClick={() => handleSelectSetting(node.settingName)}
    >
      {node.children?.map(renderTree)}
    </TreeItem>
  );

  const renderSettingPanel = (settingName: string) => {
    switch (settingName) {
      case "app_configuration":
        return <AppConfigurationPanel readOnly={readOnly} />;
      case "general":
        return <GeneralPanel readOnly={readOnly} />;
      case "devices":
        return <NetworkDevicesPanel readOnly={readOnly} />;
      case "access_control":
        return <NetworkAccessControlPanel readOnly={readOnly} />;
      case "telemetry":
        return <TelemetryPanel readOnly={readOnly} />;
      case "homeassistant":
        return <HomeAssistantPanel readOnly={readOnly} />;
      default:
        return <Typography>Setting not found: {settingName}</Typography>;
    }
  };

  useEffect(() => {
    const handleSettingsStep3 = () => setSelectedSetting("general");
    const handleSettingsStep5 = () => setSelectedSetting("access_control");
    const handleSettingsStep8 = () => setSelectedSetting("devices");

    TourEvents.on(TourEventTypes.SETTINGS_STEP_3, handleSettingsStep3);
    TourEvents.on(TourEventTypes.SETTINGS_STEP_5, handleSettingsStep5);
    TourEvents.on(TourEventTypes.SETTINGS_STEP_8, handleSettingsStep8);

    return () => {
      TourEvents.off(TourEventTypes.SETTINGS_STEP_3, handleSettingsStep3);
      TourEvents.off(TourEventTypes.SETTINGS_STEP_5, handleSettingsStep5);
      TourEvents.off(TourEventTypes.SETTINGS_STEP_8, handleSettingsStep8);
    };
  }, []);

  const treeView = (
    <SimpleTreeView
      expandedItems={expandedNodes}
      onExpandedItemsChange={(_event, nodeIds) => {
        setExpandedNodes(nodeIds);
        localStorage.setItem(SETTINGS_EXPANDED_KEY, JSON.stringify(nodeIds));
      }}
      sx={{ p: 1 }}
    >
      {filteredTree.map(renderTree)}
    </SimpleTreeView>
  );

  return (
    <InView>
      <FormProvider {...methods}>
        <Box
          sx={{
            height: "100%",
            minHeight: 0,
            display: "flex",
            flexDirection: "column",
          }}
          data-tutor={`reactour__tab${TabIDs.SETTINGS}__step0`}
        >
          {/* Search Bar */}
          <Paper
            sx={{
              p: { xs: 1.5, md: 2 },
              borderBottom: 1,
              borderColor: "divider",
            }}
            data-tutor={`reactour__tab${TabIDs.SETTINGS}__step2`}
          >
            <Stack direction="row" spacing={1.5} sx={{ alignItems: "center" }}>
              <Box sx={{ display: { xs: "flex", md: "none" } }}>
                <IconButton
                  aria-label="open settings navigation"
                  onClick={() => setMobileTreeOpen(true)}
                  size="small"
                >
                  <MenuIcon />
                </IconButton>
              </Box>
              <TextField
                fullWidth
                size="small"
                placeholder="Search settings..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                slotProps={{
                  input: {
                    startAdornment: (
                      <InputAdornment position="start">
                        <SearchIcon />
                      </InputAdornment>
                    ),
                  },
                }}
              />
            </Stack>
          </Paper>

          {/* Mobile Drawer */}
          <Drawer
            anchor="left"
            open={mobileTreeOpen}
            onClose={() => setMobileTreeOpen(false)}
            ModalProps={{ keepMounted: true }}
            sx={{ display: { xs: "block", md: "none" } }}
          >
            <Box
              sx={{
                width: 280,
                height: "100%",
                display: "flex",
                flexDirection: "column",
              }}
              role="presentation"
            >
              <Box sx={{ p: 2, borderBottom: 1, borderColor: "divider" }}>
                <Typography variant="h6">Settings</Typography>
                <Typography
                  variant="body2"
                  sx={{
                    color: "text.secondary",
                  }}
                >
                  Choose a category
                </Typography>
              </Box>
              <Box sx={{ flex: 1, overflow: "auto" }}>{treeView}</Box>
            </Box>
          </Drawer>

          {/* Main Content */}
          <Box
            sx={{
              flex: 1,
              minHeight: 0,
              display: "flex",
              flexDirection: "row",
              overflow: { xs: "auto", md: "hidden" },
            }}
          >
            {/* Left Panel - Tree View */}
            <Paper
              sx={{
                display: { xs: "none", md: "block" },
                width: 300,
                borderRight: 1,
                borderColor: "divider",
                overflow: "auto",
                flexShrink: 0,
              }}
            >
              {treeView}
            </Paper>

            {/* Right Panel - Settings */}
            <Paper
              sx={{
                flex: 1,
                minHeight: 0,
                p: { xs: 2, md: 3 },
                overflow: "auto",
              }}
            >
              <form
                id="settingsform"
                onSubmit={handleSubmit(handleCommit)}
                noValidate
                autoComplete="off"
              >
                {selectedSetting ? (
                  <Box>
                    <Typography variant="h5" gutterBottom>
                      {selectedSetting
                        .split("_")
                        .map(
                          (word) =>
                            word.charAt(0).toUpperCase() + word.slice(1),
                        )
                        .join(" ")}
                    </Typography>
                    <Divider sx={{ mb: 3 }} />
                    <Box sx={{ maxWidth: { xs: "100%", md: 600 } }}>
                      {renderSettingPanel(selectedSetting)}
                    </Box>
                  </Box>
                ) : (
                  <Box sx={{ textAlign: "center", py: 8 }}>
                    <Typography
                      variant="h6"
                      sx={{
                        color: "text.secondary",
                      }}
                    >
                      Select a setting from the tree to configure
                    </Typography>
                  </Box>
                )}
              </form>
            </Paper>
          </Box>

          {/* Bottom Button Bar */}
          {selectedSetting !== "app_configuration" ? (
            <Paper
              sx={{
                p: { xs: 1.5, md: 2 },
                borderTop: 1,
                borderColor: "divider",
              }}
              data-tutor={`reactour__tab${TabIDs.SETTINGS}__step9`}
            >
              <Stack
                direction={{ xs: "column", sm: "row" }}
                spacing={2}
                sx={{
                  justifyContent: { xs: "stretch", sm: "flex-end" },
                  alignItems: { xs: "stretch", sm: "center" },
                }}
              >
                <Button
                  onClick={() => reset()}
                  disabled={!formState.isDirty}
                  fullWidth={true}
                >
                  Reset
                </Button>
                <Button
                  type="submit"
                  form="settingsform"
                  disabled={!formState.isDirty}
                  variant="outlined"
                  color="success"
                  fullWidth={true}
                >
                  Apply
                </Button>
              </Stack>
            </Paper>
          ) : null}
        </Box>
      </FormProvider>
    </InView>
  );
}
