import { DevTool } from "@hookform/devtools";
import { useContext, useEffect, useRef } from "react";
import { InView } from "react-intersection-observer";
import Grid from "@mui/material/Grid2";
import Button from "@mui/material/Button";
import { AutocompleteElement, CheckboxElement, SelectElement, TextFieldElement, useForm, Controller } from "react-hook-form-mui";
import { MuiChipsInput } from 'mui-chips-input'
import Stack from "@mui/material/Stack";
import Divider from "@mui/material/Divider";
import { useGetSettingsQuery, usePutSettingsMutation, type DtoSettings } from "../store/sratApi";
import { useReadOnly } from "../hooks/readonlyHook";

export function Settings() {
    const read_only = useReadOnly();
    const { data: globalConfig, isLoading, error } = useGetSettingsQuery();
    const { control, handleSubmit, reset, watch, formState } = useForm({
        mode: "onBlur",
        values: globalConfig,
        disabled: read_only,
    });
    const [update, updateResponse] = usePutSettingsMutation();

    const bindAllWatch = watch("bind_all_interfaces")
    let timer: NodeJS.Timer | null = null;

    function handleCommit(data: DtoSettings) {
        console.log(data);
        update({ dtoSettings: data }).unwrap().then(res => {
            console.log(res)
        }).catch(err => console.log(err))
    }

    useEffect(() => {
        // if (!formState.isDirty) return;
        // TypeScript users 
        const subscription = watch(() => {
            if (timer) clearTimeout(timer);
            timer = setTimeout(handleSubmit(handleCommit), 1000)
        })
        //const subscription = watch(handleSubmit(onSubmit));
        return () => subscription.unsubscribe();
    }, [handleSubmit, watch]);

    return (
        <InView>
            <Stack spacing={2}>
                <form id="settingsform" onSubmit={handleSubmit(handleCommit)} noValidate>
                    <Grid container spacing={2}>
                        <Grid size={4}>
                            <SelectElement label="Update Channel" name="update_channel"
                                options={[
                                    {
                                        id: 'none',
                                        label: 'No Update',
                                    },
                                    {
                                        id: 'stable',
                                        label: 'Stable Release',
                                    },
                                    {
                                        id: 'prerelease',
                                        label: 'Beta Release',
                                    }
                                ]} sx={{ display: "flex" }} control={control} disabled={read_only} />
                        </Grid>
                        <Grid size={12}>
                            <Divider />
                        </Grid>
                        <Grid size={4}>
                            <TextFieldElement sx={{ display: "flex" }} name="workgroup" label="Workgroup" required control={control} disabled={read_only} />
                        </Grid>
                        <Grid size={8}>
                            <Controller
                                name="allow_hosts"
                                control={control}
                                disabled={read_only}
                                render={({ field }) => <MuiChipsInput label="Allow Hosts" {...field} />}
                            />
                        </Grid>
                        <Grid size={4}>
                            <CheckboxElement id="compatibility_mode" label="Compatibility Mode" name="compatibility_mode" control={control} disabled={read_only} />
                            <CheckboxElement id="multi_channel" label="Multi Channel Mode" name="multi_channel" control={control} disabled={read_only} />
                            <CheckboxElement id="recyle_bin_enabled" label="RecycleBin" name="recyle_bin_enabled" control={control} disabled={read_only} />
                        </Grid>
                        <Grid size={8}>
                            <Controller
                                name="veto_files"
                                control={control}
                                disabled={read_only}
                                render={({ field }) => <MuiChipsInput label="Veto Files" {...field} />}
                            />
                        </Grid>
                        <Grid size={4}>
                            <CheckboxElement id="bind_all_interfaces" label="Bind All Interfaces" name="bind_all_interfaces" control={control} disabled={read_only} />
                        </Grid>
                        <Grid size={8}>
                            <Controller
                                name="interfaces"
                                disabled={bindAllWatch || read_only}
                                control={control}
                                render={({ field }) => <MuiChipsInput label="Interfaces" {...field} />}
                            />
                        </Grid>
                    </Grid>
                </form>
                {/*
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
                */}
            </Stack>
            {/*   <DevTool control={control} />  set up the dev tool */}
        </InView >
    );
}