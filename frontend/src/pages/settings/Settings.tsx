import AutorenewIcon from "@mui/icons-material/Autorenew"; // Icon for fetching hostname
import PlaylistAddIcon from "@mui/icons-material/PlaylistAdd"; // Import an icon for the button
import SearchIcon from "@mui/icons-material/Search";
import { CircularProgress, IconButton, Stack, Typography, TextField, Box, Paper } from "@mui/material";
import Button from "@mui/material/Button";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import InputAdornment from "@mui/material/InputAdornment";
import Tooltip from "@mui/material/Tooltip";
import { MuiChipsInput } from "mui-chips-input";
import {
	AutocompleteElement,
	CheckboxElement,
	Controller,
	SwitchElement,
	TextFieldElement,
	useForm,
} from "react-hook-form-mui";
import { useEffect, useState, useMemo } from "react";
import { InView } from "react-intersection-observer";
import default_json from "../../json/default_config.json";
import { TabIDs } from "../../store/locationState";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import {
	type InterfaceStat,
	type Settings,
	useGetApiHostnameQuery,
	useGetApiNicsQuery,
	useGetApiSettingsQuery,
	useGetApiUpdateChannelsQuery,
	useGetApiTelemetryModesQuery,
	useGetApiTelemetryInternetConnectionQuery,
	useGetApiCapabilitiesQuery,
	usePutApiSettingsMutation,
	Telemetry_mode,
} from "../../store/sratApi";
import { useGetServerEventsQuery } from "../../store/sseApi";
import { getNodeEnv } from "../../macro/Environment" with { type: 'macro' };
import { SimpleTreeView } from "@mui/x-tree-view/SimpleTreeView";
import { TreeItem } from "@mui/x-tree-view/TreeItem";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import ChevronRightIcon from "@mui/icons-material/ChevronRight";

// --- IP Address and CIDR Validation Helpers ---
// Matches IPv4 address or IPv4 CIDR (e.g., 192.168.1.1 or 192.168.1.0/24)
// Mask range /0 to /32
const IPV4_OR_CIDR_REGEX =
	/^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\/(?:[0-9]|[12][0-9]|3[0-2]))?$/;

// Comprehensive IPv6 regex (source: https://stackoverflow.com/a/17871737/796832), modified to also accept CIDR notation.
// Covers various forms like ::1, fe80::%scope, IPv4-mapped, and their CIDR versions (e.g., 2001:db8::/32).
// Mask range /0 to /128
const IPV6_OR_CIDR_REGEX = new RegExp(
	"^(" +
	"([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|" + // 1:2:3:4:5:6:7:8
	"([0-9a-fA-F]{1,4}:){1,7}:|" + // 1::                                 1:2:3:4:5:6:7::
	"([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|" + // 1::8               1:2:3:4:5:6::8   1:2:3:4:5:6::8
	"([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|" + // 1::7:8             1:2:3:4:5::7:8   1:2:3:4:5::8
	"([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|" + // 1::6:7:8           1:2:3:4::6:7:8   1:2:3:4::8
	"([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|" + // 1::5:6:7:8         1:2:3::5:6:7:8   1:2:3::8
	"([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|" + // 1::4:5:6:7:8       1:2::4:5:6:7:8   1:2::8
	"[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|" + // 1::3:4:5:6:7:8     1::3:4:5:6:7:8   1::8
	":((:[0-9a-fA-F]{1,4}){1,7}|:)|" + // ::2:3:4:5:6:7:8    ::2:3:4:5:6:7:8  ::8       ::
	"fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|" + // fe80::7:8%eth0     fe80::7:8%1  (link-local IPv6 addresses with zone index)
	"::(ffff(:0{1,4}){0,1}:){0,1}" +
	"((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}" +
	"(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|" + // ::255.255.255.255  ::ffff:255.255.255.255  ::ffff:0:255.255.255.255 (IPv4-mapped IPv6 addresses and IPv4-translated addresses)
	"([0-9a-fA-F]{1,4}:){1,4}:" +
	"((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}" + // 2001:db8:3:4::192.0.2.33  64:ff9b::192.0.2.33 (IPv4-Embedded IPv6 Address)
	"(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])" +
	")(\/(?:[0-9]|[1-9][0-9]|1[01][0-9]|12[0-8]))?$", // Optional CIDR mask /0 to /128
);

function isValidIpAddressOrCidr(ip: string): boolean {
	if (typeof ip !== "string") return false;
	return IPV4_OR_CIDR_REGEX.test(ip) || IPV6_OR_CIDR_REGEX.test(ip);
}

// --- Hostname Validation Helper ---
// Allows alphanumeric characters and hyphens. Cannot start or end with a hyphen. Length 1-63.
const HOSTNAME_REGEX = /^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/;

// --- Workgroup Validation Helper ---
// Allows alphanumeric characters and hyphens. Cannot start or end with a hyphen. Length 1-15.
const WORKGROUP_REGEX = /^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,13}[a-zA-Z0-9])?$/;

// Tree structure for settings
interface SettingTreeNode {
	id: string;
	label: string;
	children?: SettingTreeNode[];
	settingName?: string;
}

// Define categories structure for dynamic tree building
const categories: { [key: string]: { [key: string]: string[] } | string[] } = {
	'General': ['hostname', 'workgroup', 'local_master', 'compatibility_mode'],
	'Network': {
		'Devices': ['bind_all_interfaces', 'interfaces', 'multi_channel', 'smb_over_quic'],
		'Access Control': ['allow_hosts'],
	},
	'Update': ['update_channel'],
	'Telemetry': ['telemetry_mode'],
	'HomeAssistant': ['export_stats_to_ha'],
	'Power': ['hdidle_enabled', 'hdidle_default_idle_time', 'hdidle_default_command_type', 'hdidle_ignore_spin_down_detection'],
};

// Build tree structure dynamically from categories
const buildSettingsTree = (): SettingTreeNode[] => {
	const tree: SettingTreeNode[] = [];

	Object.entries(categories).forEach(([category, subCategories]) => {
		if (Array.isArray(subCategories)) {
			// For string arrays like General, create a single leaf node
			const leafNode: SettingTreeNode = {
				id: category.toLowerCase(),
				label: category,
				settingName: category.toLowerCase(),
			};
			tree.push(leafNode);
		} else {
			// For objects like Network, create parent node with leaf children
			const categoryNode: SettingTreeNode = {
				id: category.toLowerCase(),
				label: category,
				children: [],
			};

			Object.entries(subCategories).forEach(([subCategory, settings]) => {
				const leafNode: SettingTreeNode = {
					id: `${category.toLowerCase()}_${subCategory.toLowerCase().replace(/\s+/g, '_')}`,
					label: subCategory,
					settingName: subCategory.toLowerCase().replace(/\s+/g, '_'),
				};
				categoryNode.children?.push(leafNode);
			});

			tree.push(categoryNode);
		}
	});

	return tree;
};

export function Settings() {
	const [selectedSetting, setSelectedSetting] = useState<string | null>(null);
	const [searchQuery, setSearchQuery] = useState<string>('');
	const [expandedNodes, setExpandedNodes] = useState<string[]>(['network', 'update', 'telemetry', 'hdidle']);

	const settingsTree = useMemo(() => buildSettingsTree(), []);

	// Filter tree based on search query
	const filteredTree = useMemo(() => {
		if (!searchQuery.trim()) return settingsTree;

		const filterNode = (node: SettingTreeNode): SettingTreeNode | null => {
			const matchesSearch = node.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
				(node.settingName && node.settingName.toLowerCase().includes(searchQuery.toLowerCase()));

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

	const { data: evdata, isLoading: is_evLoading } = useGetServerEventsQuery();

	const {
		data: globalConfig,
		isLoading,
		error,
		refetch,
	} = useGetApiSettingsQuery();
	const { data: nic, isLoading: inLoadinf } = useGetApiNicsQuery();
	const { data: updateChannels, isLoading: isChLoading } =
		useGetApiUpdateChannelsQuery();
	const { data: telemetryModes, isLoading: isTelemetryLoading } =
		useGetApiTelemetryModesQuery();
	const { data: internetConnection, isLoading: isInternetLoading } =
		useGetApiTelemetryInternetConnectionQuery();
	const { data: capabilities, isLoading: isCapabilitiesLoading } =
		useGetApiCapabilitiesQuery();

	const {
		control,
		handleSubmit,
		reset,
		watch,
		setValue,
		getValues,
		formState,
		subscribe,
	} = useForm({
		mode: "onBlur",
		values: globalConfig as Settings,
		disabled: evdata?.hello?.read_only,
	});
	const [update, _updateResponse] = usePutApiSettingsMutation();
	const {
		data: hostname,
		isLoading: isHostnameFetching,
		error: host_error,
		refetch: triggerGetSystemHostname,
	} = useGetApiHostnameQuery();

	const bindAllWatch = watch("bind_all_interfaces");


	function handleCommit(data: Settings) {
		console.log(data);
		update({ settings: data })
			.unwrap()
			.then((res) => {
				//console.log(res)
				reset(res as Settings);
			})
			.catch((err) => {
				console.error(err);
				reset();
			});
	}

	const handleFetchHostname = async () => {
		if (evdata?.hello?.read_only || isHostnameFetching) return;
		try {
			await triggerGetSystemHostname().unwrap();
			setValue("hostname", hostname?.toString(), {
				shouldDirty: true,
				shouldValidate: true,
			});
		} catch (error) {
			console.error("Failed to fetch hostname:", error);
		}
	};

	// Render tree node recursively
	const renderTree = (node: SettingTreeNode) => (
		<TreeItem
			key={node.id}
			itemId={node.id}
			label={node.label}
			onClick={() => node.settingName && setSelectedSetting(node.settingName)}
		>
			{node.children?.map(renderTree)}
		</TreeItem>
	);

	// Render setting field based on setting name
	const renderSettingField = (settingName: string) => {
		const commonProps = {
			control,
			disabled: evdata?.hello?.read_only,
		};

		// Check if this is a composite category (leaf node with multiple fields)
		for (const [category, subCategories] of Object.entries(categories)) {
			if (Array.isArray(subCategories)) {
				// If subCategories is a string array (like General), and settingName matches the category
				if (settingName === category.toLowerCase()) {
					// Render composite panel with all fields in the array
					return (
						<Stack spacing={3}>
							{subCategories.map((field) => (
								<Box key={field}>{renderSettingField(field)}</Box>
							))}
						</Stack>
					);
				}
			} else {
				// If subCategories is an object (like Network), check subcategories
				for (const [subCategory, settings] of Object.entries(subCategories)) {
					const normalizedSubCategory = subCategory.toLowerCase().replace(/\s+/g, '_');
					if (settingName === normalizedSubCategory && Array.isArray(settings) && settings.length > 1) {
						// Render composite panel with all fields in the array
						return (
							<Stack spacing={3}>
								{settings.map((field) => (
									<Box key={field}>{renderSettingField(field)}</Box>
								))}
							</Stack>
						);
					} else if (settingName === normalizedSubCategory && Array.isArray(settings) && settings.length === 1) {
						// Single field category, render the field directly
						return renderSettingField(settings[0] || '');
					}
				}
			}
		}

		// Individual field rendering (existing logic)
		switch (settingName) {

			case 'update_channel':
				return (
					<AutocompleteElement
						label="Update Channel"
						name="update_channel"
						loading={isChLoading}
						autocompleteProps={{
							size: "small",
							disabled: evdata?.hello?.read_only || getNodeEnv() === "production",
							contentEditable: false,
							disableClearable: true
						}}
						options={(updateChannels as string[]) || []}
						{...commonProps}
					/>
				);

			case 'telemetry_mode':
				return (
					<>
						<AutocompleteElement
							label="Telemetry Mode"
							name="telemetry_mode"
							required
							loading={isTelemetryLoading}
							autocompleteProps={{
								size: "small",
								disabled: evdata?.hello?.read_only || isInternetLoading || !internetConnection,
								contentEditable: false,
								disableClearable: true,
								autoComplete: false,
							}}
							textFieldProps={{
								autoComplete: "off",
							}}
							options={
								(telemetryModes as string[])?.filter(mode => mode !== Telemetry_mode.Ask) || []
							}
							{...commonProps}
						/>
						{!internetConnection && !isInternetLoading && (
							<Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
								Internet connection required for telemetry settings
							</Typography>
						)}
					</>
				);

			case 'export_stats_to_ha':
				return (
					<Tooltip
						title={
							<>
								<Typography variant="h6" component="div">
									Export stats to Home Assistant
								</Typography>
								<Typography variant="body2">
									If enabled, the status of disks, volumes and the server will be transmitted to Home Assistant.
								</Typography>
							</>
						}
					>
						<span style={{ display: "inline-block", width: "100%" }}>
							<SwitchElement
								switchProps={{
									"aria-label": "Export Stats to HA",
									size: "small",
								}}
								sx={{ display: "flex" }}
								name="export_stats_to_ha"
								label="Export Stats to HA"
								labelPlacement="start"
								{...commonProps}
							/>
						</span>
					</Tooltip>
				);

			case 'hostname':
				return (
					<TextFieldElement
						size="small"
						sx={{ display: "flex" }}
						name="hostname"
						label="Hostname"
						required
						rules={{
							required: "Hostname is required.",
							pattern: {
								value: HOSTNAME_REGEX,
								message:
									"Invalid hostname. Use alphanumeric characters and hyphens (not at start/end). Max 63 chars.",
							},
							maxLength: {
								value: 63,
								message: "Hostname cannot exceed 63 characters.",
							},
						}}
						slotProps={{
							input: {
								endAdornment: (
									<InputAdornment position="end">
										<Tooltip title="Fetch current system hostname">
											<span>
												<IconButton
													aria-label="fetch system hostname"
													onClick={handleFetchHostname}
													edge="end"
													disabled={evdata?.hello?.read_only || isHostnameFetching}
												>
													{isHostnameFetching ? (
														<CircularProgress size={20} />
													) : (
														<AutorenewIcon />
													)}
												</IconButton>
											</span>
										</Tooltip>
									</InputAdornment>
								),
							},
						}}
						{...commonProps}
					/>
				);

			case 'local_master':
				return (
					<Tooltip
						title={
							<>
								<Typography variant="h6" component="div">
									Enable Local Master
								</Typography>
								<Typography variant="body2">
									This option allows nmbd(8) to try and become a local master
									browser on a subnet. If set to no then nmbd will not
									attempt to become a local master browser on a subnet and
									will also lose in all browsing elections. By default this
									value is set to yes. Setting this value to yes doesn't
									mean that Samba will become the local master browser on a
									subnet, just that nmbd will participate in elections for
									local master browser.
								</Typography>
								<Typography variant="body2">
									Setting this value to no will cause nmbd never to become a
									local master browser.
								</Typography>
							</>
						}
					>
						<span style={{ display: "inline-block", width: "100%" }}>
							<SwitchElement
								switchProps={{
									"aria-label": "Local Master",
									size: "small",
								}}
								sx={{ display: "flex" }}
								name="local_master"
								label="Local Master"
								labelPlacement="start"
								{...commonProps}
							/>
						</span>
					</Tooltip>
				);

			case 'compatibility_mode':
				return (
					<SwitchElement
						switchProps={{
							'aria-label': 'Compatibility Mode',
							size: "small",
						}}
						id="compatibility_mode"
						label="Compatibility Mode"
						labelPlacement="start"
						name="compatibility_mode"
						{...commonProps}
					/>
				);

			case 'workgroup':
				return (
					<TextFieldElement
						size="small"
						sx={{ display: "flex" }}
						name="workgroup"
						label="Workgroup"
						required
						rules={{
							required: "Workgroup is required.",
							pattern: {
								value: WORKGROUP_REGEX,
								message:
									"Invalid workgroup name. Use alphanumeric characters and hyphens (not at start/end). Max 15 chars.",
							},
							maxLength: {
								value: 15,
								message: "Workgroup name cannot exceed 15 characters.",
							},
						}}
						{...commonProps}
					/>
				);

			case 'allow_hosts':
				return (
					<Controller
						name="allow_hosts"
						control={control}
						defaultValue={[]}
						disabled={evdata?.hello?.read_only}
						rules={{
							required: "Allow Hosts cannot be empty.",
							validate: (chips: string[] | undefined) => {
								if (!chips || chips.length === 0) return true;

								for (const chip of chips) {
									if (
										typeof chip !== "string" ||
										!isValidIpAddressOrCidr(chip)
									) {
										return `Invalid entry: "${chip}". Only IPv4/IPv6 addresses or CIDR notation allowed.`;
									}
								}
								return true;
							},
						}}
						render={({ field, fieldState: { error } }) => (
							<MuiChipsInput
								{...field}
								size="small"
								label="Allow Hosts"
								required
								hideClearAll
								validate={(chipValue) =>
									typeof chipValue === "string" &&
									isValidIpAddressOrCidr(chipValue)
								}
								error={!!error}
								helperText={error ? error.message : undefined}
								slotProps={{
									input: {
										endAdornment: (
											<InputAdornment position="end" sx={{ pr: 1 }}>
												{!evdata?.hello?.read_only && (
													<Tooltip title="Add suggested default Allow Hosts">
														<IconButton
															aria-label="add suggested default allow hosts"
															onClick={() => {
																const currentAllowHosts: string[] =
																	getValues("allow_hosts") || [];
																const defaultAllowHosts: string[] =
																	default_json.allow_hosts || [];
																const validDefaultHosts =
																	defaultAllowHosts.filter((host) =>
																		isValidIpAddressOrCidr(host),
																	);
																const newAllowHostsToAdd =
																	validDefaultHosts.filter(
																		(defaultHost) =>
																			!currentAllowHosts.includes(
																				defaultHost,
																			),
																	);
																setValue(
																	"allow_hosts",
																	[
																		...currentAllowHosts,
																		...newAllowHostsToAdd,
																	],
																	{
																		shouldDirty: true,
																		shouldValidate: true,
																	},
																);
															}}
															edge="end"
														>
															<PlaylistAddIcon />
														</IconButton>
													</Tooltip>
												)}
											</InputAdornment>
										),
									},
								}}
								renderChip={(Component, key, props) => {
									const isDefault = default_json.allow_hosts?.includes(
										props.label as string,
									);
									return (
										<Component
											key={key}
											{...props}
											sx={{
												color: isDefault
													? "text.secondary"
													: "text.primary",
											}}
											size="small"
										/>
									);
								}}
							/>
						)}
					/>
				);

			case 'multi_channel':
				return (
					<Tooltip
						title={
							<>
								<Typography variant="h6" component="div">
									Enable Multi Channel Mode
								</Typography>
								<Typography variant="body2">
									This boolean parameter controls whether smbd(8) will support
									SMB3 multi-channel.
								</Typography>
							</>
						}
					>
						<span style={{ display: "inline-block", width: "100%" }}>
							<SwitchElement
								switchProps={{
									"aria-label": "Multi Channel Mode",
									size: "small",
								}}
								id="multi_channel"
								label="Multi Channel Mode"
								name="multi_channel"
								labelPlacement="start"
								{...commonProps}
							/>
						</span>
					</Tooltip>
				);

			case 'smb_over_quic':
				return (
					<>
						<Tooltip
							title={
								<>
									<Typography variant="h6" component="div">
										Enable SMB over QUIC
									</Typography>
									<Typography variant="body2">
										This parameter enables SMB over QUIC transport protocol for improved
										performance and security. Requires Samba 4.23+ and QUIC kernel
										module support.
									</Typography>
									{capabilities && 'supports_quic' in capabilities && !capabilities.supports_quic && 'unsupported_reason' in capabilities && capabilities.unsupported_reason && (
										<Typography variant="body2" sx={{ mt: 1, color: 'warning.light' }}>
											<strong>Not available:</strong> {capabilities.unsupported_reason}
										</Typography>
									)}
								</>
							}
						>
							<span style={{ display: "inline-block", width: "100%" }}>
								<SwitchElement
									switchProps={{
										"aria-label": "SMB over QUIC",
										size: "small",
									}}
									id="smb_over_quic"
									label="SMB over QUIC"
									name="smb_over_quic"
									labelPlacement="start"
									disabled={evdata?.hello?.read_only || isCapabilitiesLoading || !(capabilities && 'supports_quic' in capabilities && capabilities.supports_quic)}
									control={control}
								/>
							</span>
						</Tooltip>
						{capabilities && 'supports_quic' in capabilities && !capabilities.supports_quic && !isCapabilitiesLoading && 'unsupported_reason' in capabilities && capabilities.unsupported_reason && (
							<Typography variant="caption" color="warning.main" sx={{ mt: 0.5, display: 'block' }}>
								{capabilities.unsupported_reason}
							</Typography>
						)}
					</>
				);

			case 'bind_all_interfaces':
				return (
					<>
						<CheckboxElement
							size="small"
							id="bind_all_interfaces"
							label="Bind All Interfaces"
							name="bind_all_interfaces"
							{...commonProps}
						/>
						<AutocompleteElement
							multiple
							label="Interfaces"
							name="interfaces"
							options={
								(nic as InterfaceStat[])
									?.map((nc) => nc.name)
									.filter((name) => name !== "lo" && name !== "hassio") || []
							}
							loading={inLoadinf}
							autocompleteProps={{
								size: "small",
								disabled: bindAllWatch || evdata?.hello?.read_only,
							}}
							control={control}
						/>
					</>
				);

			case 'interfaces':
				// This is handled in bind_all_interfaces case
				return null;

			case 'hdidle_enabled':
				return (
					<Tooltip
						title={
							<>
								<Typography variant="h6" component="div">
									Enable HDIdle Service
								</Typography>
								<Typography variant="body2">
									Automatically spin down idle disks after a configured timeout to reduce
									power consumption and extend disk lifespan.
								</Typography>
							</>
						}
					>
						<span style={{ display: "inline-block", width: "100%" }}>
							<SwitchElement
								name="hdidle_enabled"
								label="Enable Automatic Disk Spin-Down"
								labelPlacement="start"
								switchProps={{
									"aria-label": "Enable HDIdle",
									size: "small",
								}}
								{...commonProps}
							/>
						</span>
					</Tooltip>
				);

			case 'hdidle_default_idle_time':
				return (
					<>
						<TextFieldElement
							name="hdidle_default_idle_time"
							label="Default Idle Time (seconds)"
							type="number"
							required
							disabled={!control._formValues?.hdidle_enabled || evdata?.hello?.read_only}
							slotProps={{
								htmlInput: {
									min: 60,
								},
								input: {
									endAdornment: (
										<InputAdornment position="end">
											seconds
										</InputAdornment>
									),
								},
							}}
							size="small"
							control={control}
						/>
						<Typography variant="caption" color="text.secondary">
							Time before spinning down idle disks (minimum: 60 seconds)
						</Typography>
					</>
				);

			case 'hdidle_default_command_type':
				return (
					<Tooltip
						title={
							<>
								<Typography variant="body2">
									<strong>SCSI:</strong> For most modern SATA/SAS drives
								</Typography>
								<Typography variant="body2">
									<strong>ATA:</strong> For legacy ATA/IDE drives
								</Typography>
							</>
						}
					>
						<span style={{ display: "inline-block", width: "100%" }}>
							<AutocompleteElement
								name="hdidle_default_command_type"
								label="Default Command Type"
								options={["scsi", "ata"]}
								autocompleteProps={{
									size: "small",
									disabled: !control._formValues?.hdidle_enabled || evdata?.hello?.read_only,
									disableClearable: true,
								}}
								control={control}
							/>
						</span>
					</Tooltip>
				);

			case 'hdidle_ignore_spin_down_detection':
				return (
					<Tooltip
						title={
							<Typography variant="body2">
								Force spin down even if disk reports it's already spun down
							</Typography>
						}
					>
						<span style={{ display: "inline-block", width: "100%" }}>
							<CheckboxElement
								name="hdidle_ignore_spin_down_detection"
								label="Ignore Spin Down Detection"
								disabled={!control._formValues?.hdidle_enabled || evdata?.hello?.read_only}
								size="small"
								control={control}
							/>
						</span>
					</Tooltip>
				);

			default:
				return <Typography>Setting not found: {settingName}</Typography>;
		}
	};

	// Tour event handlers
	useEffect(() => {
		const handleSettingsStep3 = () => {
			setSelectedSetting('hostname');
		};

		const handleSettingsStep5 = () => {
			setSelectedSetting('allow_hosts');
		};

		const handleSettingsStep8 = () => {
			setSelectedSetting('interfaces');
		};

		TourEvents.on(TourEventTypes.SETTINGS_STEP_3, handleSettingsStep3);
		TourEvents.on(TourEventTypes.SETTINGS_STEP_5, handleSettingsStep5);
		TourEvents.on(TourEventTypes.SETTINGS_STEP_8, handleSettingsStep8);

		return () => {
			TourEvents.off(TourEventTypes.SETTINGS_STEP_3, handleSettingsStep3);
			TourEvents.off(TourEventTypes.SETTINGS_STEP_5, handleSettingsStep5);
			TourEvents.off(TourEventTypes.SETTINGS_STEP_8, handleSettingsStep8);
		};
	}, []);

	return (
		<InView>
			<Box sx={{ height: 'max', display: 'flex', flexDirection: 'column' }}>
				{/* Search Bar */}
				<Paper sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
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
				</Paper>

				{/* Main Content */}
				<Box sx={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
					{/* Left Panel - Tree View */}
					<Paper
						sx={{
							width: 300,
							borderRight: 1,
							borderColor: 'divider',
							overflow: 'auto',
							flexShrink: 0,
						}}
					>
						<SimpleTreeView
							expandedItems={expandedNodes}
							onExpandedItemsChange={(event, nodeIds) => setExpandedNodes(nodeIds)}
							sx={{ p: 1 }}
						>
							{filteredTree.map(renderTree)}
						</SimpleTreeView>
					</Paper>

					{/* Right Panel - Settings */}
					<Paper sx={{ flex: 1, p: 3, overflow: 'auto' }}>
						<form
							id="settingsform"
							onSubmit={handleSubmit(handleCommit)}
							noValidate
							autoComplete="off"
						>
							{selectedSetting ? (
								<Box>
									<Typography variant="h5" gutterBottom>
										{selectedSetting.split('_').map(word =>
											word.charAt(0).toUpperCase() + word.slice(1)
										).join(' ')}
									</Typography>
									<Divider sx={{ mb: 3 }} />
									<Box sx={{ maxWidth: 600 }}>
										{renderSettingField(selectedSetting)}
									</Box>
								</Box>
							) : (
								<Box sx={{ textAlign: 'center', py: 8 }}>
									<Typography variant="h6" color="text.secondary">
										Select a setting from the tree to configure
									</Typography>
								</Box>
							)}
						</form>
					</Paper>
				</Box>

				{/* Bottom Button Bar */}
				<Paper sx={{ p: 2, borderTop: 1, borderColor: 'divider' }}>
					<Stack
						direction="row"
						spacing={2}
						sx={{
							justifyContent: "flex-end",
							alignItems: "center",
						}}
					>
						<Button onClick={() => reset()} disabled={!formState.isDirty}>
							Reset
						</Button>
						<Button
							type="submit"
							form="settingsform"
							disabled={!formState.isDirty}
							variant="outlined"
							color="success"
						>
							Apply
						</Button>
					</Stack>
				</Paper>
			</Box>
		</InView>
	);
}
