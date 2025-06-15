import { DevTool } from "@hookform/devtools";
import { useContext, useEffect, useRef, useCallback } from "react";
import { InView, useInView } from "react-intersection-observer";
import Grid from "@mui/material/Grid";
import Button from "@mui/material/Button";
import { AutocompleteElement, CheckboxElement, SelectElement, TextFieldElement, useForm, Controller } from "react-hook-form-mui";
import { MuiChipsInput } from 'mui-chips-input'
import { Stack, CircularProgress } from "@mui/material";
import Divider from "@mui/material/Divider";
import { useGetNicsQuery, useGetSettingsQuery, usePutSettingsMutation, type NetworkInfo, type Settings, useGetHostnameQuery, useGetUpdateChannelsQuery, type GetUpdateChannelsApiResponse } from "../store/sratApi";
import { useReadOnly } from "../hooks/readonlyHook";
import debounce from 'lodash.debounce';
import { NIL } from "uuid";
import InputAdornment from "@mui/material/InputAdornment";
import Tooltip from "@mui/material/Tooltip";
import { Chip, IconButton, Typography } from "@mui/material";
import PlaylistAddIcon from '@mui/icons-material/PlaylistAdd'; // Import an icon for the button
import AutorenewIcon from '@mui/icons-material/Autorenew'; // Icon for fetching hostname
import default_json from "../json/default_config.json"
import language from "react-syntax-highlighter/dist/esm/languages/hljs/1c";
import { Label } from "@mui/icons-material";

// --- IP Address and CIDR Validation Helpers ---
// Matches IPv4 address or IPv4 CIDR (e.g., 192.168.1.1 or 192.168.1.0/24)
// Mask range /0 to /32
const IPV4_OR_CIDR_REGEX = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\/(?:[0-9]|[12][0-9]|3[0-2]))?$/;

// Comprehensive IPv6 regex (source: https://stackoverflow.com/a/17871737/796832), modified to also accept CIDR notation.
// Covers various forms like ::1, fe80::%scope, IPv4-mapped, and their CIDR versions (e.g., 2001:db8::/32).
// Mask range /0 to /128
const IPV6_OR_CIDR_REGEX = new RegExp(
    '^(' +
    '([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|' + // 1:2:3:4:5:6:7:8
    '([0-9a-fA-F]{1,4}:){1,7}:|' +              // 1::                                 1:2:3:4:5:6:7::
    '([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|' + // 1::8               1:2:3:4:5:6::8   1:2:3:4:5:6::8
    '([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|' + // 1::7:8             1:2:3:4:5::7:8   1:2:3:4:5::8
    '([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|' + // 1::6:7:8           1:2:3:4::6:7:8   1:2:3:4::8
    '([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|' + // 1::5:6:7:8         1:2:3::5:6:7:8   1:2:3::8
    '([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|' + // 1::4:5:6:7:8       1:2::4:5:6:7:8   1:2::8
    '[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|' +      // 1::3:4:5:6:7:8     1::3:4:5:6:7:8   1::8
    ':((:[0-9a-fA-F]{1,4}){1,7}|:)|' +                   // ::2:3:4:5:6:7:8    ::2:3:4:5:6:7:8  ::8       ::
    'fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|' +    // fe80::7:8%eth0     fe80::7:8%1  (link-local IPv6 addresses with zone index)
    '::(ffff(:0{1,4}){0,1}:){0,1}' +
    '((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}' +
    '(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|' +         // ::255.255.255.255  ::ffff:255.255.255.255  ::ffff:0:255.255.255.255 (IPv4-mapped IPv6 addresses and IPv4-translated addresses)
    '([0-9a-fA-F]{1,4}:){1,4}:' +
    '((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}' + // 2001:db8:3:4::192.0.2.33  64:ff9b::192.0.2.33 (IPv4-Embedded IPv6 Address)
    '(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])' +
    ')(\/(?:[0-9]|[1-9][0-9]|1[01][0-9]|12[0-8]))?$' // Optional CIDR mask /0 to /128
);

function isValidIpAddressOrCidr(ip: string): boolean {
    if (typeof ip !== 'string') return false;
    return IPV4_OR_CIDR_REGEX.test(ip) || IPV6_OR_CIDR_REGEX.test(ip);
}

// --- Veto File Entry Validation Helper ---
// Matches a valid Samba veto file entry:
// - Not empty
// - Does not contain '/' (as it's a separator for the list in smb.conf)
// - Does not contain null byte '\0'
const VETO_FILE_ENTRY_REGEX = /^[^/\0]+$/;

function isValidVetoFileEntry(entry: string): boolean {
    if (typeof entry !== 'string') return false;
    return VETO_FILE_ENTRY_REGEX.test(entry);
}

// --- Hostname Validation Helper ---
// Allows alphanumeric characters and hyphens. Cannot start or end with a hyphen. Length 1-63.
const HOSTNAME_REGEX = /^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/;

// --- Workgroup Validation Helper ---
// Allows alphanumeric characters and hyphens. Cannot start or end with a hyphen. Length 1-15.
const WORKGROUP_REGEX = /^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,13}[a-zA-Z0-9])?$/;

export function Settings() {
    const read_only = useReadOnly();
    const { data: globalConfig, isLoading, error, refetch } = useGetSettingsQuery();
    const { data: nic, isLoading: inLoadinf } = useGetNicsQuery();
    const { data: updateChannels, isLoading: isChLoading } = useGetUpdateChannelsQuery();


    const { control, handleSubmit, reset, watch, setValue, getValues, formState, subscribe } = useForm({
        mode: "onBlur",
        values: globalConfig as Settings,
        disabled: read_only,
    });
    const [update, updateResponse] = usePutSettingsMutation();
    const { data: hostname, isLoading: isHostnameFetching, error: host_error, refetch: triggerGetSystemHostname } = useGetHostnameQuery();

    const bindAllWatch = watch("bind_all_interfaces")

    /*
    const debouncedCommit = debounce((data: Settings) => {
        //console.log("Committing")
        handleCommit(data);
    }, 500, { leading: true });
    */

    function handleCommit(data: Settings) {
        console.log(data);
        update({ settings: data }).unwrap().then(res => {
            //console.log(res)
            reset(res as Settings)
        }).catch(err => {
            console.error(err)
            reset()
        })
    }

    const handleFetchHostname = async () => {
        if (read_only || isHostnameFetching) return;
        try {
            await triggerGetSystemHostname().unwrap();
            setValue("hostname", hostname?.toString(), { shouldDirty: true, shouldValidate: true });
        } catch (error) {
            console.error("Failed to fetch hostname:", error);
        }
    }

    /*
    useEffect(() => {
        // make sure to unsubscribe;
        const callback = subscribe({
            formState: {
                isDirty: true,
            },
            callback: ({ values }) => {
                //console.log(values);
                //console.log(formState.isDirty, formState.isSubmitted, formState.isSubmitting)
                handleSubmit(debouncedCommit)()
            }
        })

        return () => callback();

        // You can also just return the subscribe
        // return subscribe();
    }, [subscribe, handleSubmit])
    */

    return (
        <InView>
            <br />
            <Stack spacing={2} sx={{ p: 2 }}>
                <Divider />
                <form id="settingsform" onSubmit={handleSubmit(handleCommit)} noValidate>
                    <Grid container spacing={2}>
                        <Grid size={4}>
                            <AutocompleteElement
                                label="Update Channel" name="update_channel"
                                loading={isChLoading}
                                autocompleteProps={{
                                    size: "small",
                                    disabled: read_only || process.env.NODE_ENV === 'production',
                                }}
                                options={((updateChannels as string[]) || [])}
                                control={control}
                            />
                        </Grid>
                        <Grid size={12}>
                            <Divider />
                        </Grid>
                        <Grid size={4}>
                            <TextFieldElement
                                size="small"
                                sx={{ display: "flex" }}
                                name="hostname"
                                label="Hostname"
                                required
                                control={control}
                                rules={{
                                    required: 'Hostname is required.',
                                    pattern: {
                                        value: HOSTNAME_REGEX,
                                        message: 'Invalid hostname. Use alphanumeric characters and hyphens (not at start/end). Max 63 chars.'
                                    },
                                    maxLength: {
                                        value: 63,
                                        message: 'Hostname cannot exceed 63 characters.'
                                    }
                                }}
                                disabled={read_only}
                                slotProps={{
                                    input: {
                                        endAdornment: (
                                            <InputAdornment position="end">
                                                <Tooltip title="Fetch current system hostname">
                                                    {/* Span needed for tooltip when IconButton is disabled */}
                                                    <span>
                                                        <IconButton
                                                            aria-label="fetch system hostname"
                                                            onClick={handleFetchHostname}
                                                            edge="end"
                                                            disabled={read_only || isHostnameFetching}
                                                        >
                                                            {isHostnameFetching ? <CircularProgress size={20} /> : <AutorenewIcon />}
                                                        </IconButton>
                                                    </span>
                                                </Tooltip>
                                            </InputAdornment>
                                        )
                                    }
                                }} />
                        </Grid>
                        <Grid size={4}>
                            <TextFieldElement
                                size="small"
                                sx={{ display: "flex" }}
                                name="workgroup"
                                label="Workgroup"
                                required
                                control={control}
                                rules={{
                                    required: 'Workgroup is required.',
                                    pattern: {
                                        value: WORKGROUP_REGEX,
                                        message: 'Invalid workgroup name. Use alphanumeric characters and hyphens (not at start/end). Max 15 chars.'
                                    },
                                    maxLength: {
                                        value: 15,
                                        message: 'Workgroup name cannot exceed 15 characters.'
                                    }
                                }}
                                disabled={read_only} />
                        </Grid>
                        <Grid size={8}>
                            <Controller
                                name="allow_hosts"
                                control={control}
                                defaultValue={[]}
                                disabled={read_only}
                                rules={{
                                    required: 'Allow Hosts cannot be empty.',
                                    validate: (chips: string[] | undefined) => {
                                        if (!chips || chips.length === 0) return true; // Handled by 'required'

                                        for (const chip of chips) { // Ensure chip is a string before validation
                                            if (typeof chip !== 'string' || !isValidIpAddressOrCidr(chip)) {
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
                                        validate={(chipValue) => typeof chipValue === 'string' && isValidIpAddressOrCidr(chipValue)}
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
                                                                        const currentAllowHosts: string[] = getValues("allow_hosts") || [];
                                                                        const defaultAllowHosts: string[] = default_json.allow_hosts || [];
                                                                        // Only add default hosts that are valid IP addresses or CIDR
                                                                        const validDefaultHosts = defaultAllowHosts.filter(host => isValidIpAddressOrCidr(host));
                                                                        const newAllowHostsToAdd = validDefaultHosts.filter(
                                                                            (defaultHost) => !currentAllowHosts.includes(defaultHost)
                                                                        );
                                                                        setValue("allow_hosts", [...currentAllowHosts, ...newAllowHostsToAdd], { shouldDirty: true, shouldValidate: true });
                                                                    }}
                                                                    edge="end"
                                                                >
                                                                    <PlaylistAddIcon />
                                                                </IconButton>
                                                            </Tooltip>
                                                        )}
                                                    </InputAdornment>
                                                ),
                                            }
                                        }}
                                        renderChip={(Component, key, props) => {
                                            const isDefault = default_json.allow_hosts?.includes(props.label as string);
                                            return (
                                                <Component {...props} sx={{ color: isDefault ? 'text.secondary' : 'text.primary' }} size="small" key={key} />
                                            );
                                        }}
                                    />)}
                            />
                        </Grid>
                        <Grid size={4}>
                            <CheckboxElement size="small" id="compatibility_mode" label="Compatibility Mode" name="compatibility_mode" control={control} disabled={read_only} />
                            <CheckboxElement size="small" id="multi_channel" label="Multi Channel Mode" name="multi_channel" control={control} disabled={read_only} />
                        </Grid>
                        <Grid size={8}>
                            <Controller
                                name="veto_files"
                                control={control}
                                defaultValue={[]}
                                disabled={read_only}
                                rules={{
                                    validate: (chips: string[] | undefined) => {
                                        if (!chips || chips.length === 0) return true; // Allow empty list
                                        for (const chip of chips) {
                                            if (!isValidVetoFileEntry(chip)) {
                                                return `Invalid entry: "${chip}". Veto file entries cannot be empty, contain '/' or null characters.`;
                                            }
                                        }
                                        return true;
                                    },
                                }}
                                render={({ field, fieldState: { error } }) => (
                                    <MuiChipsInput
                                        {...field}
                                        size="small"
                                        hideClearAll
                                        label="Veto Files"
                                        validate={(chipValue) => typeof chipValue === 'string' && isValidVetoFileEntry(chipValue)}
                                        error={!!error}
                                        helperText={error ? error.message : "List of files/patterns to hide (e.g., ._* Thumbs.db). Entries cannot contain '/'."}
                                        renderChip={(Component, key, props) => {
                                            const isDefault = default_json.veto_files?.includes(props.label as string);
                                            return (
                                                <Component {...props} sx={{ color: isDefault ? 'text.secondary' : 'text.primary' }} size="small" key={key} />
                                            );
                                        }}
                                        slotProps={{
                                            input: {
                                                endAdornment: (
                                                    <InputAdornment position="end" sx={{ pr: 1 }}>
                                                        {!read_only && (
                                                            <Tooltip title="Add suggested default Veto files">
                                                                <IconButton
                                                                    aria-label="add suggested default veto files"
                                                                    onClick={() => {
                                                                        const currentVetoFiles: string[] = getValues("veto_files") || [];
                                                                        const defaultVetoFiles: string[] = default_json.veto_files || [];
                                                                        const newVetoFilesToAdd = defaultVetoFiles.filter(
                                                                            (defaultFile) => !currentVetoFiles.includes(defaultFile)
                                                                        );
                                                                        setValue("veto_files", [...currentVetoFiles, ...newVetoFilesToAdd], { shouldDirty: true, shouldValidate: true });
                                                                    }}
                                                                    edge="end"
                                                                >
                                                                    <PlaylistAddIcon />
                                                                </IconButton>
                                                            </Tooltip>
                                                        )}
                                                    </InputAdornment>
                                                ),
                                            }
                                        }}

                                    />)}
                            />
                        </Grid>
                        <Grid size={4}>
                            <CheckboxElement size="small" id="bind_all_interfaces" label="Bind All Interfaces" name="bind_all_interfaces" control={control} disabled={read_only} />
                        </Grid>
                        <Grid size={8}>
                            <AutocompleteElement
                                multiple
                                label="Interfaces"
                                name="interfaces"
                                options={(nic as NetworkInfo)?.nics?.map(nc => nc.name) || []}
                                control={control}
                                loading={inLoadinf}
                                autocompleteProps={{
                                    size: "small",
                                    disabled: bindAllWatch || read_only
                                }}
                            />
                        </Grid>
                        <Grid size={4}>
                            <SelectElement label="WSDD Service" name="wsdd"
                                size="small"
                                required
                                options={[
                                    {
                                        id: 'none',
                                        label: 'None',
                                    },
                                    {
                                        id: 'wsdd',
                                        label: 'Wsdd (Alpine apk)',
                                    },
                                    {
                                        id: 'wsdd2',
                                        label: 'Netgear wsdd2',
                                    },
                                ]} sx={{ display: "flex" }} control={control} disabled={read_only} />
                        </Grid>
                    </Grid>
                </form>
                <Divider />
                <Stack direction="row"
                    spacing={2}
                    sx={{
                        justifyContent: "flex-end",
                        alignItems: "center",
                    }} >
                    <Button onClick={() => reset()} disabled={!formState.isDirty}>Reset</Button>
                    <Button type="submit" form="settingsform" disabled={!formState.isDirty}>Apply</Button>
                </Stack>
            </Stack>
            {/*   <DevTool control={control} />  set up the dev tool */}
        </InView >
    );
}
