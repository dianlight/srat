import CloudOutlinedIcon from "@mui/icons-material/CloudOutlined";
import DeleteIcon from "@mui/icons-material/Delete";
import LinkIcon from "@mui/icons-material/Link";
import RefreshIcon from "@mui/icons-material/Refresh";
import ScienceOutlinedIcon from "@mui/icons-material/ScienceOutlined";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Grid,
  Stack,
  Typography,
} from "@mui/material";
import { filesize } from "filesize";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { CheckboxElement, FormContainer } from "react-hook-form-mui";
import { toast } from "react-toastify";
import type { RcloneDiffResult } from "../../../../store/sratApi";
import {
  Direction,
  useDeleteApiRcloneLinkMutation,
  useGetApiRcloneLinkQuery,
  useGetApiRcloneProvidersQuery,
  usePostApiRcloneLinkDiffMutation,
  usePostApiRcloneLinkSyncMutation,
} from "../../../../store/sratApi";
import { extractApiErrorMessage } from "./apiErrors";
import { CloudLinkWizardDialog } from "./CloudLinkWizardDialog";
import { CloudSyncProgressDialog } from "./CloudSyncProgressDialog";
import { isDiffResult, isProvidersResponse, isRcloneLink } from "./typeGuards";

export interface CloudSyncPanelProps {
  targetKind: string;
  targetId: string;
  targetLabel: string;
  readOnly?: boolean;
}

const DIFF_COLUMNS = [
  { key: "local_only", title: "Local only" },
  { key: "remote_only", title: "Remote only" },
  { key: "changed", title: "Changed" },
] as const;

function formatSize(bytes?: number): string {
  if (bytes === undefined || bytes === null) {
    return "";
  }
  return filesize(bytes, { standard: "jedec" }) as string;
}

function formatLastSync(iso?: string | null): string {
  if (!iso) {
    return "Never synced";
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return iso;
  }
  return `Last sync: ${date.toLocaleString()}`;
}

/**
 * Lab-gated cloud link panel for a mounted volume target. Shows the
 * unlinked placeholder with the link wizard, or — once linked — the
 * provider/status cards, the difference table and Push/Pull/Sync actions
 * with live progress and abort.
 */
export function CloudSyncPanel({
  targetKind,
  targetId,
  targetLabel,
  readOnly = false,
}: CloudSyncPanelProps) {
  const [wizardOpen, setWizardOpen] = useState(false);
  const [unlinkConfirmOpen, setUnlinkConfirmOpen] = useState(false);
  const [confirmDirection, setConfirmDirection] = useState<Direction | null>(
    null,
  );
  const [progressDirection, setProgressDirection] = useState<Direction | null>(
    null,
  );
  const [progressDryRun, setProgressDryRun] = useState(false);
  const [progressOpen, setProgressOpen] = useState(false);
  const [diffResult, setDiffResult] = useState<RcloneDiffResult | null>(null);

  const linkQuery = useGetApiRcloneLinkQuery({
    targetKind,
    targetId,
  });
  const providersQuery = useGetApiRcloneProvidersQuery();
  const [deleteLink, { isLoading: isDeleting }] =
    useDeleteApiRcloneLinkMutation();
  const [diff, { isLoading: isDiffing }] = usePostApiRcloneLinkDiffMutation();
  const [startSync, { isLoading: isStarting }] =
    usePostApiRcloneLinkSyncMutation();

  // RTK Query exposes non-2xx via `error`, so any success-shaped data
  // present means linked.
  const link = isRcloneLink(linkQuery.data) ? linkQuery.data : undefined;
  const linked = Boolean(link);

  const providersData = isProvidersResponse(providersQuery.data)
    ? providersQuery.data
    : undefined;

  const providerDisplayName = useMemo(() => {
    const name = link?.provider ?? "";
    return (
      providersData?.providers?.find((p) => p.name === name)?.display_name ??
      name
    );
  }, [providersData, link?.provider]);

  const handleRefreshDiff = async () => {
    try {
      const result = await diff({ targetKind, targetId }).unwrap();
      if (isDiffResult(result)) {
        setDiffResult(result);
      } else {
        throw Object.assign(new Error(), { data: result });
      }
    } catch (err) {
      toast.error(extractApiErrorMessage(err, "Failed to compute differences"));
    }
  };

  const handleUnlink = async () => {
    try {
      await deleteLink({ targetKind, targetId }).unwrap();
      toast.success(`${targetLabel} unlinked from ${providerDisplayName}.`);
      setUnlinkConfirmOpen(false);
      setDiffResult(null);
    } catch (err) {
      toast.error(extractApiErrorMessage(err, "Failed to unlink"));
    }
  };

  const confirmForm = useForm<{ dry_run: boolean }>({
    defaultValues: { dry_run: false },
  });

  const handleStartSync = async (direction: Direction, dryRun: boolean) => {
    try {
      await startSync({
        targetKind,
        targetId,
        rcloneSyncRequest: { direction, dry_run: dryRun },
      }).unwrap();
      setProgressDirection(direction);
      setProgressDryRun(dryRun);
      setProgressOpen(true);
    } catch (err) {
      toast.error(extractApiErrorMessage(err, "Failed to start the sync"));
    } finally {
      setConfirmDirection(null);
    }
  };

  const entries = diffResult?.entries ?? [];
  const totalDifferences = entries.length;

  const directionButtons: Array<{ direction: Direction; label: string }> = [
    { direction: Direction.Push, label: "Push local → remote" },
    { direction: Direction.Pull, label: "Pull remote → local" },
    { direction: Direction.Bidi, label: "Sync both ways" },
  ];

  const confirmCopy: Record<Direction, string> = {
    push: "Copy new and changed local files to the remote folder.",
    pull: "Copy new and changed remote files to this volume.",
    bidi: "Synchronize both directions. Conflicting files are resolved by newest modification time.",
  };

  return (
    <Card>
      <CardContent>
        <Stack spacing={2}>
          <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
            <CloudOutlinedIcon color="primary" />
            <Typography variant="h6">Cloud Sync</Typography>
            <ScienceOutlinedIcon color="warning" fontSize="small" />
          </Stack>

          {!linked ? (
            <>
              <Typography variant="body2" color="text.secondary">
                This volume is not linked to a cloud provider.
              </Typography>
              <Box>
                <Button
                  variant="contained"
                  startIcon={<LinkIcon />}
                  disabled={readOnly}
                  onClick={() => setWizardOpen(true)}
                >
                  Link to cloud…
                </Button>
              </Box>
            </>
          ) : (
            <>
              <Grid container spacing={2}>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Provider
                  </Typography>
                  <Stack
                    direction="row"
                    spacing={1}
                    sx={{ alignItems: "center" }}
                  >
                    <Typography>
                      {providerDisplayName || link?.provider}
                    </Typography>
                    <Chip
                      size="small"
                      color={
                        link?.status === "authorized" ? "success" : "warning"
                      }
                      label={link?.status ?? ""}
                    />
                  </Stack>
                  <Typography variant="body2" color="text.secondary">
                    Remote folder: {link?.remote_path}
                  </Typography>
                </Grid>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    Status
                  </Typography>
                  <Stack
                    direction="row"
                    spacing={1}
                    sx={{ alignItems: "center" }}
                  >
                    <Typography variant="body2">
                      {formatLastSync(link?.last_sync_at)}
                    </Typography>
                    {link?.last_sync_result && (
                      <Chip
                        size="small"
                        color={
                          link.last_sync_result === "success"
                            ? "success"
                            : "error"
                        }
                        label={link.last_sync_result}
                      />
                    )}
                  </Stack>
                  {link?.last_sync_message && (
                    <Typography variant="caption" color="text.secondary">
                      {link.last_sync_message}
                    </Typography>
                  )}
                </Grid>
              </Grid>

              <Divider />
              {diffResult?.warning && (
                <Alert severity="warning">{diffResult.warning}</Alert>
              )}
              {diffResult && totalDifferences === 0 && !diffResult.warning && (
                <Alert severity="success">No differences found.</Alert>
              )}
              <Stack
                direction="row"
                spacing={1}
                sx={{
                  justifyContent: "space-between",
                  alignItems: "center",
                }}
              >
                <Typography variant="subtitle1">
                  Differences ({totalDifferences})
                </Typography>
                <Button
                  size="small"
                  startIcon={<RefreshIcon />}
                  disabled={isDiffing}
                  onClick={() => {
                    void handleRefreshDiff();
                  }}
                >
                  {isDiffing ? "Computing…" : "Refresh diff"}
                </Button>
              </Stack>
              <Grid container spacing={2}>
                {DIFF_COLUMNS.map((column) => {
                  const columnEntries = entries.filter(
                    (entry) => entry.diff_type === column.key,
                  );
                  return (
                    <Grid key={column.key} size={{ xs: 12, md: 4 }}>
                      <Typography
                        variant="subtitle2"
                        color="text.secondary"
                        gutterBottom
                      >
                        {`${column.title} (${columnEntries.length})`}
                      </Typography>
                      {columnEntries.length === 0 ? (
                        <Typography variant="caption" color="text.disabled">
                          None
                        </Typography>
                      ) : (
                        <Stack spacing={0.5}>
                          {columnEntries.map((entry) => (
                            <Typography
                              key={`${column.key}:${entry.path}`}
                              variant="caption"
                              noWrap
                            >
                              {entry.path}
                              {(entry.local_size !== undefined ||
                                entry.remote_size !== undefined) &&
                                ` (${formatSize(
                                  entry.local_size ?? entry.remote_size,
                                )})`}
                            </Typography>
                          ))}
                        </Stack>
                      )}
                    </Grid>
                  );
                })}
              </Grid>
              <Divider />
              <Stack
                direction="row"
                spacing={1}
                sx={{ flexWrap: "wrap" }}
                useFlexGap
              >
                {directionButtons.map(({ direction, label }) => (
                  <Button
                    key={direction}
                    variant="outlined"
                    disabled={readOnly || isStarting}
                    onClick={() => {
                      confirmForm.reset({ dry_run: false });
                      setConfirmDirection(direction);
                    }}
                  >
                    {label}
                  </Button>
                ))}
                <Button
                  color="error"
                  startIcon={<DeleteIcon />}
                  disabled={readOnly || isDeleting}
                  onClick={() => setUnlinkConfirmOpen(true)}
                >
                  Unlink
                </Button>
              </Stack>
            </>
          )}
        </Stack>
      </CardContent>

      <CloudLinkWizardDialog
        open={wizardOpen}
        onClose={() => setWizardOpen(false)}
        targetKind={targetKind}
        targetId={targetId}
        targetLabel={targetLabel}
      />

      <CloudSyncProgressDialog
        open={progressOpen}
        onClose={() => setProgressOpen(false)}
        direction={progressDirection}
        dryRun={progressDryRun}
        targetKind={targetKind}
        targetId={targetId}
        targetLabel={targetLabel}
      />

      {/* Sync confirmation (FormContainer so the dry-run checkbox and the
          submit button share one form; onSuccess receives validated data) */}
      <Dialog
        open={confirmDirection !== null}
        onClose={() => setConfirmDirection(null)}
      >
        <FormContainer
          formContext={confirmForm}
          onSuccess={async ({ dry_run }) => {
            if (confirmDirection) {
              await handleStartSync(confirmDirection, dry_run);
            }
          }}
        >
          <DialogTitle>Confirm synchronization</DialogTitle>
          <DialogContent>
            <Typography variant="body2">
              {confirmDirection ? confirmCopy[confirmDirection] : ""}
            </Typography>
            <CheckboxElement
              name="dry_run"
              label="Dry run (preview only — nothing is transferred)"
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setConfirmDirection(null)}>Cancel</Button>
            <Button
              type="submit"
              variant="contained"
              color={
                confirmDirection === Direction.Bidi ? "warning" : "primary"
              }
              disabled={isStarting}
            >
              Start
            </Button>
          </DialogActions>
        </FormContainer>
      </Dialog>

      {/* Unlink confirmation */}
      <Dialog
        open={unlinkConfirmOpen}
        onClose={() => setUnlinkConfirmOpen(false)}
      >
        <DialogTitle>Unlink from {providerDisplayName}?</DialogTitle>
        <DialogContent>
          <Typography variant="body2">
            The local link configuration will be removed. Remote files are NOT
            deleted.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setUnlinkConfirmOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            color="error"
            disabled={isDeleting}
            onClick={() => {
              void handleUnlink();
            }}
          >
            Unlink
          </Button>
        </DialogActions>
      </Dialog>
    </Card>
  );
}
