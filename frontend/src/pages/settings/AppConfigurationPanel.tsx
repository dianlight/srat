import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Divider from "@mui/material/Divider";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { MuiChipsInput } from "mui-chips-input";
import { useEffect, useMemo } from "react";
import { Controller, SwitchElement, TextFieldElement, useForm } from "react-hook-form-mui";
import {
    type AppConfigData,
    type AppConfigSchema,
    type AppConfigSchemaField,
    type AppConfigUpdateRequest,
    useGetApiSettingsAppConfigQuery,
    useGetApiSettingsAppConfigSchemaQuery,
    usePutApiSettingsAppConfigMutation,
} from "../../store/sratApi";

type AppConfigFormValues = Record<string, unknown>;

type AppConfigurationPanelProps = {
    readOnly: boolean;
};

type ConstraintKind = "boolean" | "integer" | "number" | "list" | "password" | "text";

type RenderableField = {
    name: string;
    constraint: string;
    description?: string;
    optional?: boolean;
    options?: string[];
};

function humanizeName(name: string): string {
    return name
        .split(/[_-]/)
        .filter(Boolean)
        .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
        .join(" ");
}

function normalizeFormValues(options: Record<string, unknown> | undefined): AppConfigFormValues {
    if (!options) {
        return {};
    }

    return Object.fromEntries(
        Object.entries(options).map(([key, value]) => {
            if (Array.isArray(value)) {
                return [key, value.map((item) => String(item))];
            }
            if (typeof value === "boolean" || typeof value === "number" || typeof value === "string") {
                return [key, value];
            }
            if (value == null) {
                return [key, ""];
            }
            return [key, JSON.stringify(value)];
        }),
    );
}

function isOptionalConstraint(constraint: string): boolean {
    return constraint.trim().endsWith("?");
}

function normalizeConstraint(constraint: string): string {
    return constraint.trim().replace(/\?$/, "");
}

function getConstraintKind(constraint: string): ConstraintKind {
    const normalized = normalizeConstraint(constraint);
    if (normalized === "bool" || normalized === "boolean") {
        return "boolean";
    }
    if (normalized === "int" || normalized === "integer") {
        return "integer";
    }
    if (normalized === "float" || normalized === "number") {
        return "number";
    }
    if (normalized === "password") {
        return "password";
    }
    if (normalized.startsWith("list(")) {
        return "list";
    }
    return "text";
}

function convertListValues(rawValues: unknown[], itemConstraint: string): unknown[] {
    const normalizedItemConstraint = normalizeConstraint(itemConstraint);
    if (normalizedItemConstraint === "int") {
        return rawValues
            .map((value) => Number.parseInt(String(value), 10))
            .filter((value) => !Number.isNaN(value));
    }
    if (normalizedItemConstraint === "float") {
        return rawValues
            .map((value) => Number.parseFloat(String(value)))
            .filter((value) => !Number.isNaN(value));
    }
    return rawValues.map((value) => String(value));
}

function coerceConstraintValue(value: unknown, field: RenderableField): unknown {
    const kind = getConstraintKind(field.constraint);
    const optional = isOptionalConstraint(field.constraint);

    if ((value === "" || value == null) && optional && kind !== "boolean" && kind !== "list") {
        return undefined;
    }

    switch (kind) {
        case "boolean":
            return Boolean(value);
        case "integer": {
            const parsed = Number.parseInt(String(value), 10);
            return Number.isNaN(parsed) ? undefined : parsed;
        }
        case "number": {
            const parsed = Number.parseFloat(String(value));
            return Number.isNaN(parsed) ? undefined : parsed;
        }
        case "list": {
            const listValues = Array.isArray(value) ? value : [];
            const match = normalizeConstraint(field.constraint).match(/^list\((.*)\)$/);
            return convertListValues(listValues, match?.[1] ?? "str");
        }
        default:
            return value == null ? "" : String(value);
    }
}

function buildRequestOptions(
    currentOptions: Record<string, unknown> | undefined,
    values: AppConfigFormValues,
    fields: RenderableField[],
): AppConfigUpdateRequest["options"] {
    const nextOptions: Record<string, unknown> = { ...(currentOptions ?? {}) };

    for (const field of fields) {
        const nextValue = coerceConstraintValue(values[field.name], field);
        if (nextValue === undefined && isOptionalConstraint(field.constraint)) {
            delete nextOptions[field.name];
            continue;
        }
        nextOptions[field.name] = nextValue;
    }

    return nextOptions;
}

function optionsFromConstraint(constraint: string): string[] {
    const normalized = normalizeConstraint(constraint);

    const directEnumMatch = normalized.match(/^(?:str|enum|select)\(([^)]+)\)$/);
    if (directEnumMatch?.[1]) {
        return directEnumMatch[1].split("|").map((item) => item.trim()).filter(Boolean);
    }

    if (normalized.startsWith("match(")) {
        const inner = normalized.slice("match(".length, -1);
        const groupMatch = inner.match(/\(([^)]+)\)/);
        if (groupMatch?.[1]) {
            return groupMatch[1].split("|").map((item) => item.trim()).filter(Boolean);
        }
    }

    return [];
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

function deepEqualValues(left: unknown, right: unknown): boolean {
    if (left === right) {
        return true;
    }

    if (Array.isArray(left) && Array.isArray(right)) {
        if (left.length !== right.length) {
            return false;
        }

        for (let index = 0; index < left.length; index += 1) {
            if (!deepEqualValues(left[index], right[index])) {
                return false;
            }
        }

        return true;
    }

    if (isRecord(left) && isRecord(right)) {
        const leftKeys = Object.keys(left);
        const rightKeys = Object.keys(right);

        if (leftKeys.length !== rightKeys.length) {
            return false;
        }

        for (const key of leftKeys) {
            if (!(key in right)) {
                return false;
            }
            if (!deepEqualValues(left[key], right[key])) {
                return false;
            }
        }

        return true;
    }

    return false;
}

function buildRenderableFields(
    schemaFields: AppConfigSchemaField[],
    options: Record<string, unknown> | undefined,
): RenderableField[] {
    const inferConstraintFromOptionValue = (value: unknown): string => {
        if (typeof value === "boolean") {
            return "bool";
        }
        if (typeof value === "number") {
            return Number.isInteger(value) ? "int" : "float";
        }
        if (Array.isArray(value)) {
            const first = value[0];
            if (typeof first === "number") {
                return Number.isInteger(first) ? "list(int)" : "list(float)";
            }
            return "list(str)";
        }
        return "str";
    };

    if (schemaFields.length > 0) {
        return schemaFields
            .map((field) => {
                const recordField = field as unknown as Record<string, unknown>;
                const explicitOptions = Array.isArray(recordField.options)
                    ? recordField.options.map((option) => String(option)).filter(Boolean)
                    : [];
                const constraintOptions = optionsFromConstraint(field.constraint);

                return {
                    name: field.name,
                    constraint: field.constraint,
                    description: field.description,
                    optional: typeof recordField.optional === "boolean" ? recordField.optional : isOptionalConstraint(field.constraint),
                    options: explicitOptions.length > 0 ? explicitOptions : constraintOptions,
                };
            })
            .sort((a, b) => a.name.localeCompare(b.name));
    }

    if (options && Object.keys(options).length > 0) {
        return Object.entries(options)
            .map(([name, value]) => {
                const inferredConstraint = inferConstraintFromOptionValue(value);
                return {
                    name,
                    constraint: inferredConstraint,
                    options: optionsFromConstraint(inferredConstraint),
                };
            })
            .sort((a, b) => a.name.localeCompare(b.name));
    }

    return Object.keys(options ?? {})
        .map((name) => ({ name, constraint: "str", options: optionsFromConstraint("str") }))
        .sort((a, b) => a.name.localeCompare(b.name));
}

export function AppConfigurationPanel({ readOnly }: AppConfigurationPanelProps) {
    const { data: appConfig, isLoading: isConfigLoading, isFetching: isConfigFetching } = useGetApiSettingsAppConfigQuery();
    const { data: schema, isLoading: isSchemaLoading, isFetching: isSchemaFetching } = useGetApiSettingsAppConfigSchemaQuery();
    const [updateAppConfig, updateState] = usePutApiSettingsAppConfigMutation();
    const appConfigData = appConfig && "options" in appConfig ? (appConfig as AppConfigData) : undefined;
    const schemaData = schema && "fields" in schema ? (schema as AppConfigSchema) : undefined;

    const schemaFields = useMemo(() => schemaData?.fields ?? [], [schemaData?.fields]);
    const fields = useMemo(
        () => buildRenderableFields(schemaFields, appConfigData?.options),
        [schemaFields, appConfigData?.options],
    );

    const showRuntimeConfiguration = useMemo(() => {
        if (!appConfigData?.runtime_config || Object.keys(appConfigData.runtime_config).length === 0) {
            return false;
        }

        return !deepEqualValues(appConfigData.options, appConfigData.runtime_config);
    }, [appConfigData?.options, appConfigData?.runtime_config]);

    const initialValues = useMemo(() => normalizeFormValues(appConfigData?.options), [appConfigData?.options]);

    const {
        control,
        handleSubmit,
        reset,
        formState,
    } = useForm<AppConfigFormValues>({
        mode: "onBlur",
        defaultValues: initialValues,
        disabled: readOnly,
    });

    useEffect(() => {
        reset(initialValues);
    }, [initialValues, reset]);

    const onSubmit = handleSubmit(async (values) => {
        const options = buildRequestOptions(appConfigData?.options, values, fields);
        await updateAppConfig({ appConfigUpdateRequest: { options } }).unwrap();
    });

    if (isConfigLoading || isSchemaLoading) {
        return (
            <Stack spacing={2} sx={{ alignItems: "center", py: 6 }}>
                <CircularProgress size={28} />
                <Typography color="text.secondary">Loading app configuration…</Typography>
            </Stack>
        );
    }

    return (
        <Stack spacing={3}>
            <Alert severity="warning" variant="outlined">
                Changes require an app restart before they fully take effect.
            </Alert>

            {fields.length === 0 ? (
                <Alert severity="info" variant="outlined">
                    This app does not expose configurable options.
                </Alert>
            ) : (
                <Box id="app-config-form">
                    <Stack spacing={3}>
                        {fields.map((field) => {
                            const label = humanizeName(field.name);
                            const helperText = field.description ?? "";
                            const kind = getConstraintKind(field.constraint);
                            const required = !(field.optional ?? isOptionalConstraint(field.constraint));

                            if (kind === "boolean") {
                                return (
                                    <Stack key={field.name} spacing={0.5}>
                                        <SwitchElement
                                            control={control}
                                            disabled={readOnly}
                                            label={label}
                                            labelPlacement="start"
                                            name={field.name}
                                            switchProps={{
                                                "aria-label": label,
                                                size: "small",
                                            }}
                                        />
                                        {helperText ? (
                                            <Typography color="text.secondary" variant="caption">
                                                {helperText}
                                            </Typography>
                                        ) : null}
                                    </Stack>
                                );
                            }

                            if (kind === "list") {
                                return (
                                    <Controller
                                        key={field.name}
                                        name={field.name}
                                        control={control}
                                        defaultValue={[]}
                                        rules={{
                                            required: required ? `${label} is required.` : false,
                                        }}
                                        render={({ field: controllerField, fieldState }) => (
                                            <MuiChipsInput
                                                {...controllerField}
                                                value={Array.isArray(controllerField.value) ? (controllerField.value as string[]) : []}
                                                size="small"
                                                label={label}
                                                disabled={readOnly}
                                                hideClearAll
                                                error={Boolean(fieldState.error)}
                                                helperText={fieldState.error?.message ?? helperText}
                                            />
                                        )}
                                    />
                                );
                            }

                            if ((field.options?.length ?? 0) > 0) {
                                return (
                                    <Controller
                                        key={field.name}
                                        name={field.name}
                                        control={control}
                                        rules={{
                                            required: required ? `${label} is required.` : false,
                                        }}
                                        render={({ field: controllerField, fieldState }) => (
                                            <TextField
                                                select
                                                size="small"
                                                label={label}
                                                disabled={readOnly}
                                                error={Boolean(fieldState.error)}
                                                helperText={fieldState.error?.message ?? helperText}
                                                value={typeof controllerField.value === "string" ? controllerField.value : ""}
                                                onChange={(event) => controllerField.onChange(event.target.value)}
                                                onBlur={controllerField.onBlur}
                                                name={controllerField.name}
                                                inputRef={controllerField.ref}
                                                fullWidth
                                            >
                                                {field.options?.map((option) => (
                                                    <MenuItem key={option} value={option}>
                                                        {option}
                                                    </MenuItem>
                                                ))}
                                            </TextField>
                                        )}
                                    />
                                );
                            }

                            return (
                                <TextFieldElement
                                    key={field.name}
                                    control={control}
                                    disabled={readOnly}
                                    helperText={helperText}
                                    label={label}
                                    name={field.name}
                                    required={required}
                                    rules={{
                                        required: required ? `${label} is required.` : false,
                                    }}
                                    size="small"
                                    type={kind === "integer" || kind === "number" ? "number" : kind === "password" ? "password" : "text"}
                                    slotProps={kind === "integer" ? { htmlInput: { step: 1 } } : kind === "number" ? { htmlInput: { step: "any" } } : undefined}
                                />
                            );
                        })}

                        <Stack direction={{ xs: "column", sm: "row" }} spacing={2} sx={{ justifyContent: "flex-end" }}>
                            <Button disabled={!formState.isDirty || updateState.isLoading} onClick={() => reset(initialValues)}>
                                Reset
                            </Button>
                            <Button
                                variant="outlined"
                                color="warning"
                                onClick={() => {
                                    void onSubmit();
                                }}
                                disabled={!formState.isDirty || updateState.isLoading || readOnly}
                            >
                                {updateState.isLoading ? "Applying…" : "Apply App Configuration"}
                            </Button>
                        </Stack>
                    </Stack>
                </Box>
            )}

            {showRuntimeConfiguration ? (
                <>
                    <Divider />
                    <Box>
                        <Typography gutterBottom variant="h6">
                            Rendered Runtime Configuration
                        </Typography>
                        <Paper sx={{ p: 2, overflowX: "auto" }} variant="outlined">
                            <TextField
                                fullWidth
                                multiline
                                minRows={6}
                                value={JSON.stringify(appConfigData?.runtime_config ?? {}, null, 2)}
                                InputProps={{ readOnly: true }}
                            />
                        </Paper>
                    </Box>
                </>
            ) : null}

            {updateState.error ? (
                <Alert severity="error" variant="outlined">
                    Unable to save app configuration. Please review the values and try again.
                </Alert>
            ) : null}

            {(isConfigFetching || isSchemaFetching) && !isConfigLoading && !isSchemaLoading ? (
                <Typography color="text.secondary" variant="caption">
                    Refreshing app configuration…
                </Typography>
            ) : null}
        </Stack>
    );
}
