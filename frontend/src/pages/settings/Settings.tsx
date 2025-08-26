import AutorenewIcon from "@mui/icons-material/Autorenew"; // Icon for fetching hostname
import PlaylistAddIcon from "@mui/icons-material/PlaylistAdd"; // Import an icon for the button
import { CircularProgress, IconButton, Stack, Typography } from "@mui/material";
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
import { useEffect } from "react";
import { InView } from "react-intersection-observer";
import { useReadOnly } from "../../hooks/readonlyHook";
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
	usePutApiSettingsMutation,
	Telemetry_mode,
} from "../../store/sratApi";

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

export function Settings() {
	const read_only = useReadOnly();
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
		disabled: read_only,
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
		if (read_only || isHostnameFetching) return;
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

	// Tour event handlers
	useEffect(() => {
		const handleSettingsStep3 = () => {
			// Focus on hostname field when step 3 is triggered
			const hostnameField = document.querySelector('input[name="hostname"]') as HTMLInputElement;
			if (hostnameField) {
				hostnameField.focus();
			}
		};

		const handleSettingsStep5 = () => {
			// Focus on allow hosts field when step 5 is triggered
			const allowHostsField = document.querySelector('input[placeholder*="Allow Hosts"]') as HTMLInputElement;
			if (allowHostsField) {
				allowHostsField.focus();
			}
		};

		const handleSettingsStep8 = () => {
			// Focus on interfaces field when step 8 is triggered
			const interfacesField = document.querySelector('input[placeholder*="Interfaces"]') as HTMLInputElement;
			if (interfacesField) {
				interfacesField.focus();
			}
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
			<br />
			<Stack
				spacing={2}
				sx={[
					(theme) => ({

						backgroundColor: theme.vars?.palette.background.default,
					}),
					(theme) =>
						theme.applyStyles('dark', {
							backgroundColor: theme.vars?.palette.grey[900],
						}),
				]}
				data-tutor={`reactour__tab${TabIDs.SETTINGS}__step0`}
			>
				<Divider data-tutor={`reactour__tab${TabIDs.SETTINGS}__step2`} />
				<form
					id="settingsform"
					onSubmit={handleSubmit(handleCommit)}
					noValidate
					autoComplete="off"
				>
					<Grid container spacing={2}>
						<Grid
							size={{ xs: 12, md: 4 }}
							data-tutor={`reactour__tab${TabIDs.SETTINGS}__step2`}>
							<AutocompleteElement
								label="Update Channel"
								name="update_channel"
								loading={isChLoading}
								autocompleteProps={{
									size: "small",
									disabled: read_only || process.env.NODE_ENV === "production",
									contentEditable: false, // Prevent manual input in production
									disableClearable: true
								}}
								options={(updateChannels as string[]) || []}
								control={control}
							/>
						</Grid>
						<Grid size={{ xs: 12, md: 4 }}>
							<AutocompleteElement
								label="Telemetry Mode"
								name="telemetry_mode"
								required
								loading={isTelemetryLoading}
								autocompleteProps={{
									size: "small",
									disabled: read_only || isInternetLoading || !internetConnection,
									contentEditable: false, // Prevent manual input in production
									disableClearable: true,
									autoComplete: false,
								}}
								textFieldProps={{
									autoComplete: "off",
								}}
								options={
									(telemetryModes as string[])?.filter(mode => mode !== Telemetry_mode.Ask) || []
								}
								control={control}
							/>
							{!internetConnection && !isInternetLoading && (
								<Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
									Internet connection required for telemetry settings
								</Typography>
							)}
						</Grid>
						<Grid size={{ xs: 12, md: 4 }}>
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
								<SwitchElement
									switchProps={{
										"aria-label": "Export Stats to HA",
										size: "small",
									}}
									sx={{ display: "flex" }}
									name="export_stats_to_ha"
									label="Export Stats to HA"
									control={control}
									disabled={read_only}
								/>
							</Tooltip>
						</Grid>

						<Grid size={12}>
							<Divider />
						</Grid>
						<Grid size={{ xs: 12, md: 4 }} data-tutor={`reactour__tab${TabIDs.SETTINGS}__step3`}>
							<TextFieldElement
								size="small"
								sx={{ display: "flex" }}
								name="hostname"
								label="Hostname"
								required
								control={control}
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
								disabled={read_only}
								slotProps={{
									input: {
										endAdornment: (
											<InputAdornment position="end">
												<Tooltip title="Fetch current system hostname">
													{/* Span needed for tooltip when IconButton is disabled */}
													<span data-tutor={`reactour__tab${TabIDs.SETTINGS}__step4`}>
														<IconButton
															aria-label="fetch system hostname"
															onClick={handleFetchHostname}
															edge="end"
															disabled={read_only || isHostnameFetching}
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
							/>
						</Grid>
						<Grid size={{ xs: 12, md: 2 }}>
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
								<SwitchElement
									switchProps={{
										"aria-label": "Local Master",
										size: "small",
									}}
									sx={{ display: "flex" }}
									name="local_master"
									label="Local Master"
									control={control}
									disabled={read_only}
								/>
							</Tooltip>
						</Grid>
						<Grid size={{ xs: 12, md: 2 }} data-tutor={`reactour__tab${TabIDs.SETTINGS}__step7`}>
							<SwitchElement
								switchProps={{
									'aria-label': 'Compatibility Mode',
									size: "small",
								}}
								id="compatibility_mode"
								label="Compatibility Mode"
								name="compatibility_mode"
								control={control}
								disabled={read_only}
							/>
						</Grid>
						<Grid size={{ xs: 12, md: 4 }}>
							<TextFieldElement
								size="small"
								sx={{ display: "flex" }}
								name="workgroup"
								label="Workgroup"
								required
								control={control}
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
								disabled={read_only}
							/>
						</Grid>
						<Grid size={{ xs: 12, md: 12 }} data-tutor={`reactour__tab${TabIDs.SETTINGS}__step5`}>
							<Controller
								name="allow_hosts"
								control={control}
								defaultValue={[]}
								disabled={read_only}
								rules={{
									required: "Allow Hosts cannot be empty.",
									validate: (chips: string[] | undefined) => {
										if (!chips || chips.length === 0) return true; // Handled by 'required'

										for (const chip of chips) {
											// Ensure chip is a string before validation
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
														{!read_only && (
															<Tooltip title="Add suggested default Allow Hosts">
																<IconButton
																	aria-label="add suggested default allow hosts"
																	onClick={() => {
																		const currentAllowHosts: string[] =
																			getValues("allow_hosts") || [];
																		const defaultAllowHosts: string[] =
																			default_json.allow_hosts || [];
																		// Only add default hosts that are valid IP addresses or CIDR
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
																	data-tutor={`reactour__tab${TabIDs.SETTINGS}__step6`}
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
						</Grid>
						<Grid size={{ xs: 12, md: 2 }} data-tutor={`reactour__tab${TabIDs.SETTINGS}__step7`}>
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
								<SwitchElement
									switchProps={{
										"aria-label": "Multi Channel Mode",
										size: "small",
									}}
									id="multi_channel"
									label="Multi Channel Mode"
									name="multi_channel"
									control={control}
									disabled={read_only}
								/>
							</Tooltip>
						</Grid>
						<Grid size={{ xs: 12, md: 4 }} data-tutor={`reactour__tab${TabIDs.SETTINGS}__step8`}>
							<SwitchElement
								switchProps={{
									'aria-label': 'Bind All Interfaces',
									size: "small",
								}}
								id="bind_all_interfaces"
								label="Bind All Interfaces"
								name="bind_all_interfaces"
								control={control}
								disabled={read_only}
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
								control={control}
								loading={inLoadinf}
								autocompleteProps={{
									size: "small",
									disabled: bindAllWatch || read_only,
								}}
							/>
						</Grid>
					</Grid>
				</form>
				<Divider />
				<Stack
					direction="row"
					spacing={2}
					sx={{
						justifyContent: "flex-end",
						alignItems: "center",
					}}
					data-tutor={`reactour__tab${TabIDs.SETTINGS}__step9`}
				>
					<Button onClick={() => reset()} disabled={!formState.isDirty}>
						Reset
					</Button>
					<Button
						type="submit"
						form="settingsform"
						disabled={!formState.isDirty}
					>
						Apply
					</Button>
				</Stack>
			</Stack>
			{/*   <DevTool control={control} />  set up the dev tool */}
		</InView>
	);
}
