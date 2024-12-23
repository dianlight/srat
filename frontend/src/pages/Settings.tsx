import { Controller, useForm } from "react-hook-form";
import { DevTool } from "@hookform/devtools";
import { useContext, useRef } from "react";
import { apiContext, ModeContext } from "../Contexts";
import { InView } from "react-intersection-observer";
import Grid from "@mui/material/Grid2";
import Button from "@mui/material/Button";
import useSWR from "swr";
import type { ConfigUser, MainGlobalConfig } from "../srat";
import { AutocompleteElement, CheckboxElement, TextFieldElement } from "react-hook-form-mui";
import { MuiChipsInput } from 'mui-chips-input'
import Stack from "@mui/material/Stack";
import Divider from "@mui/material/Divider";

export function Settings() {
    const api = useContext(apiContext);
    const mode = useContext(ModeContext);
    const globalConfig = useSWR<MainGlobalConfig>('/global', () => api.global.globalList().then(res => res.data));
    const { control, handleSubmit, reset, watch, formState } = useForm({
        // mode: "onChange",
        values: globalConfig.data,
        disabled: mode.read_only
    });
    const bindAllWatch = watch("bind_all_interfaces")

    function handleCommit(data: MainGlobalConfig) {
        console.log(data);
        api.global.globalUpdate(data).then(res => {
            console.log(res)
            globalConfig.mutate()
        }).catch(err => console.log(err))
    }

    return (
        <InView>
            <Stack spacing={2}>
                <form id="settingsform" onSubmit={handleSubmit(handleCommit)} noValidate>
                    <Grid container spacing={2}>
                        <Grid size={4}>
                            <TextFieldElement sx={{ display: "flex" }} name="workgroup" label="Workgroup" required control={control} disabled={mode.read_only} />
                        </Grid>
                        <Grid size={8}>
                            <Controller
                                name="allow_hosts"
                                control={control}
                                disabled={mode.read_only}
                                render={({ field }) => <MuiChipsInput label="Allow Hosts" {...field} />}
                            />
                        </Grid>
                        <Grid size={4}>
                            <CheckboxElement label="Compatibility Mode" name="compatibility_mode" control={control} disabled={mode.read_only} />
                            <CheckboxElement label="Multi Channel Mode" name="multi_channel" control={control} disabled={mode.read_only} />
                            <CheckboxElement label="RecycleBin" name="recyle_bin_enabled" control={control} disabled={mode.read_only} />
                        </Grid>
                        <Grid size={8}>
                            <Controller
                                name="veto_files"
                                control={control}
                                disabled={mode.read_only}
                                render={({ field }) => <MuiChipsInput label="Veto Files" {...field} />}
                            />
                        </Grid>
                        <Grid size={4}>
                            <CheckboxElement label="Bind All Interfaces" name="bind_all_interfaces" control={control} disabled={mode.read_only} />
                        </Grid>
                        <Grid size={8}>
                            <Controller
                                name="interfaces"
                                disabled={bindAllWatch || mode.read_only}
                                control={control}
                                render={({ field }) => <MuiChipsInput label="Interfaces" {...field} />}
                            />
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
            <DevTool control={control} /> {/* set up the dev tool */}
        </InView >
    );
}