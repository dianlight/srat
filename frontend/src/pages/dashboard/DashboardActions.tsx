import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  FormControlLabel,
  Switch,
  Typography,
} from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { toast } from "react-toastify";

import IssueCard from "../../components/IssueCard";
import { useLabFeatures } from "../../hooks/useLabFeatures";
import { useSentryTelemetry } from "../../hooks/useSentryTelemetry";
import { useVolume } from "../../hooks/volumeHook";
import { TabIDs } from "../../store/locationState";
import {
  type Partition,
  type Problem,
  sratApi,
  Status,
  useDeleteApiProblemsByProblemKeyMutation,
  useGetApiProblemsQuery,
  usePutApiProblemsByProblemKeyMutation,
} from "../../store/sratApi";
import { useAppDispatch } from "../../store/store";
import { useGetServerEventsQuery } from "../../store/wsApi";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import { ActionableItemsList } from "./components/ActionableItemsList";
import {
  describeProblemActionFailure,
  dismissProblemToast,
  getAuthoritativeProblem,
  getLocalProblemIgnore,
  getProblemIgnoreOverride,
  isInactiveProblem,
  isProblemIgnoreOverrideActive,
  isProblemResponse,
  mergeLiveProblems,
  type ProblemAction,
  type ProblemIgnoreOverride,
  problemToastId,
  reconcileProblemIgnoreMutationResult,
  resolveReenableTarget,
  restoreProblemIgnoreOverride,
  setLocalProblemIgnore,
  showProblemToast,
  useProblemIgnoreOverrides,
  useReconcileProblemIgnoreOverrides,
} from "./problemActionError";

export function DashboardActions() {
  const { disks, isLoading, error } = useVolume();
  const [expanded, setExpanded] = useState(false);
  const [showIgnored, setShowIgnored] = useState(false);
  const { data: evdata } = useGetServerEventsQuery();
  const { data: problems } = useGetApiProblemsQuery();
  const [dismissProblem] = useDeleteApiProblemsByProblemKeyMutation();
  const [upsertProblem] = usePutApiProblemsByProblemKeyMutation();
  const { isAvailable: labFeatureAvailable } = useLabFeatures();
  const customComponentLabActive = labFeatureAvailable("ha_custom_component");
  const { reportError } = useSentryTelemetry();
  const dispatch = useAppDispatch();
  const ignoreOverrides = useProblemIgnoreOverrides();
  useReconcileProblemIgnoreOverrides(problems, evdata?.problem);

  // #1360: parse RTK rejections detail-first (detail, then message, then
  // status) and report serialized { problemKey, status, detail } context so
  // Sentry never sees a bare string with `[Object]` console_args. The toast
  // carries the same detail plus a retry hint for 5xx / fetch failures.
  function reportProblemActionFailure(
    action: ProblemAction,
    problemKey: string,
    err: unknown,
  ): void {
    const failure = describeProblemActionFailure(action, problemKey, err);
    const error = new Error(failure.message);
    (error as { cause?: unknown }).cause = err;
    const context = {
      problemKey: failure.problemKey,
      status: failure.status,
      detail: failure.detail,
      response: failure.response,
    };
    // Log the Error first so the console→Sentry forwarder titles on
    // `Failed to <action> problem <key>: <detail>` (#1354 pattern).
    console.error(error, context);
    reportError(error, context);
    toast.error(failure.toastMessage);
  }

  function patchProblemCache(problemKey: string, problem: Problem): void {
    dispatch(
      sratApi.util.updateQueryData("getApiProblems", undefined, (draft) => {
        if (!Array.isArray(draft)) return;
        const target = draft.find((p) => p?.problem_key === problemKey);
        if (target) {
          Object.assign(target, problem);
        }
      }),
    );
  }

  function applyMutationResult(response: unknown): void {
    if (!isProblemResponse(response)) {
      return;
    }
    // Update the REST cache first: App/NotificationCenter suppress stale WS
    // payloads with this server-sourced response once the override is cleared.
    patchProblemCache(response.problem_key, response);
    reconcileProblemIgnoreMutationResult(response);
  }

  function rollbackProblemAction(
    problemKey: string,
    requestedIgnored: boolean,
    previousProblem: Problem,
    previousOverride: ProblemIgnoreOverride | null | undefined,
  ): void {
    if (getLocalProblemIgnore(problemKey) !== requestedIgnored) {
      return;
    }
    restoreProblemIgnoreOverride(problemKey, previousOverride);
    patchProblemCache(problemKey, previousProblem);

    const shouldShowToast =
      !isInactiveProblem(previousProblem) && previousOverride?.ignored !== true;
    if (shouldShowToast) {
      showProblemToast(previousProblem);
    } else {
      dismissProblemToast(problemToastId(problemKey));
    }
  }

  const serverMergedProblems = useMemo(() => {
    const baseProblems = Array.isArray(problems) ? problems : [];

    const isSuppressed = (problemKey?: string | null) =>
      Boolean(problemKey?.startsWith("custom_component_")) &&
      !customComponentLabActive;

    const visibleBase = customComponentLabActive
      ? baseProblems
      : baseProblems.filter((problem) => !isSuppressed(problem?.problem_key));

    const incomingProblem = evdata?.problem;
    if (!incomingProblem || isSuppressed(incomingProblem.problem_key)) {
      return visibleBase;
    }

    const existingIndex = visibleBase.findIndex(
      (problem) => problem?.problem_key === incomingProblem.problem_key,
    );

    const isRemovedStatus =
      incomingProblem.status === Status.Dismissed ||
      incomingProblem.status === Status.Deleted;

    if (isRemovedStatus) {
      if (existingIndex < 0) {
        return visibleBase;
      }

      return visibleBase.filter(
        (problem) => problem?.problem_key !== incomingProblem.problem_key,
      );
    }

    return mergeLiveProblems(visibleBase, incomingProblem);
  }, [problems, evdata?.problem, customComponentLabActive]);

  // Locally requested ignore/re-enable state wins only while it disagrees with
  // a stale server snapshot. Once REST/WS confirms the click the override is
  // cleared and the card follows the authoritative problem state.
  const mergedProblems = useMemo(() => {
    if (ignoreOverrides.size === 0) {
      return serverMergedProblems;
    }
    return serverMergedProblems.map((problem) => {
      const override = problem?.problem_key
        ? ignoreOverrides.get(problem.problem_key)
        : undefined;
      if (!isProblemIgnoreOverrideActive(override, problem)) {
        return problem;
      }
      return {
        ...problem,
        ignored: override?.ignored === true,
        status: override?.ignored ? Status.Ignored : Status.Created,
      };
    });
  }, [serverMergedProblems, ignoreOverrides]);

  useEffect(() => {
    const handleDashboardStep3 = () => {
      setExpanded(true);
    };

    TourEvents.on(TourEventTypes.DASHBOARD_STEP_3, handleDashboardStep3);

    return () => {
      TourEvents.off(TourEventTypes.DASHBOARD_STEP_3, handleDashboardStep3);
    };
  }, []);

  const actionablePartitions = useMemo(() => {
    const partitions: {
      partition: Partition;
      action: "mount" | "share" | "enable-share";
    }[] = [];
    if (disks && !evdata?.hello?.read_only) {
      for (const disk of disks) {
        // disks type should be inferred from useVolume
        const diskPartitions = Object.values(disk.partitions || {});
        for (const partition of diskPartitions) {
          // Filter out system/host-mounted partitions
          if (
            partition.system ||
            partition.name?.startsWith("hassos-") ||
            (partition.host_mount_point_data &&
              Object.values(partition.host_mount_point_data).length > 0)
          ) {
            continue;
          }

          const mpds = Object.values(partition.mount_point_data || {});
          const isMounted = mpds.some((mpd) => mpd.is_mounted);
          const hasEnabledShare = mpds.some(
            (mpd) => mpd.share && mpd.share.disabled !== true,
          );
          const hasDisabledShare = mpds.some(
            (mpd) => mpd.share && mpd.share.disabled === true,
          );

          const firstMountPath = mpds[0]?.path;

          if (!isMounted) {
            partitions.push({ partition, action: "mount" });
          } else if (!hasEnabledShare && firstMountPath?.startsWith("/mnt/")) {
            if (hasDisabledShare) {
              partitions.push({ partition, action: "enable-share" });
            } else {
              partitions.push({ partition, action: "share" });
            }
          }
        }
      }
    }
    return partitions;
  }, [disks, evdata?.hello?.read_only]);

  function handleResolveIssue(id: number | string): void {
    if (typeof id === "string") {
      dismissProblem({ problemKey: id })
        .unwrap()
        .catch((err) => {
          reportProblemActionFailure("dismiss", id, err);
        });
    }
  }

  // Permanent ignore: the backend stores the flag and never re-raises the
  // alert (or notifies HA) until it is dismissed/re-enabled.
  function handleIgnoreIssue(issue: Problem): void {
    const key = issue?.problem_key?.trim();
    if (!key) {
      console.error("Cannot ignore problem without problem_key", issue);
      return;
    }
    const previousProblem =
      getAuthoritativeProblem(problems, evdata?.problem, key) ?? issue;
    const previousOverride = getProblemIgnoreOverride(key);
    const optimisticProblem: Problem = {
      ...previousProblem,
      problem_key: key,
      ignored: true,
      status: Status.Ignored,
    };
    // Optimistically dismiss the toast immediately so the badge drops
    // and the card is hidden before the WS event reconciles.
    dismissProblemToast(problemToastId(key));
    setLocalProblemIgnore(key, true, previousProblem.updated_at);
    // Keep the REST cache authoritative until the mutation succeeds. The local
    // override hides the card/toast meanwhile; a failed PUT can then roll back
    // the override without the reconciler mistaking our optimistic cache patch
    // for a server confirmation.
    void upsertProblem({ problemKey: key, problem: optimisticProblem })
      .unwrap()
      .then(applyMutationResult)
      .catch((err) => {
        rollbackProblemAction(key, true, previousProblem, previousOverride);
        reportProblemActionFailure("ignore", key, err);
      });
  }

  // Re-enable a previously ignored alert. The problem is recreated right
  // away when its condition still holds (e.g. protected mode still on).
  function handleReenableIssue(id: number | string): void {
    if (typeof id !== "string") {
      return;
    }
    const target = resolveReenableTarget(serverMergedProblems, id);
    if (target.type === "upsert") {
      const previousProblem =
        getAuthoritativeProblem(problems, evdata?.problem, id) ?? target.issue;
      const previousOverride = getProblemIgnoreOverride(id);
      const optimisticProblem: Problem = {
        ...previousProblem,
        problem_key: id,
        ignored: false,
        status: Status.Created,
      };
      setLocalProblemIgnore(id, false, previousProblem.updated_at);
      void upsertProblem({ problemKey: id, problem: optimisticProblem })
        .unwrap()
        .then(applyMutationResult)
        .catch((err) => {
          rollbackProblemAction(id, false, previousProblem, previousOverride);
          reportProblemActionFailure("re-enable", id, err);
        });
    } else {
      dismissProblem({ problemKey: id })
        .unwrap()
        .catch((err) => {
          reportProblemActionFailure("dismiss", id, err);
        });
    }
  }

  // Protected mode now surfaces as a regular backend problem
  // (problem_key "protected_mode") honoring ignores and the Alerts settings
  // category. Partition actions stay disabled while protected.
  const isProtectedMode = evdata?.hello?.protected_mode === true;

  // Set initial expanded state based on content
  useEffect(() => {
    if (
      !isLoading &&
      !error &&
      actionablePartitions.length + mergedProblems.length > 0
    ) {
      setExpanded(true);
    }
  }, [isLoading, error, actionablePartitions.length, mergedProblems.length]);

  const handleAccordionChange = (
    _event: React.SyntheticEvent,
    isExpanded: boolean,
  ) => {
    setExpanded(isExpanded);
  };

  return (
    <Accordion
      data-tutor={`reactour__tab${TabIDs.DASHBOARD}__step3`}
      expanded={expanded}
      onChange={handleAccordionChange}
    >
      <AccordionSummary
        expandIcon={<ExpandMoreIcon />}
        aria-controls="actions-content"
        id="actions-header"
      >
        <Box
          sx={{
            display: "flex",
            width: "100%",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <Typography variant="h6">Actionable Items</Typography>
          <FormControlLabel
            onClick={(e) => e.stopPropagation()}
            onFocus={(e) => e.stopPropagation()}
            control={
              <Switch
                size="small"
                checked={showIgnored}
                onChange={(e) => {
                  e.stopPropagation();
                  setShowIgnored(e.target.checked);
                }}
              />
            }
            label="Show Ignored"
            sx={{ mr: 1 }}
          />
        </Box>
      </AccordionSummary>
      <AccordionDetails>
        {(
          mergedProblems.filter(Boolean) as NonNullable<
            (typeof mergedProblems)[number]
          >[]
        ).map((issue) => (
          <IssueCard
            key={issue.problem_key}
            issue={issue}
            onResolve={handleResolveIssue}
            onIgnore={handleIgnoreIssue}
            onReenable={handleReenableIssue}
            showIgnored={showIgnored}
          />
        ))}
        <ActionableItemsList
          actionablePartitions={actionablePartitions}
          isLoading={isLoading}
          error={error}
          showIgnored={showIgnored}
          disabled={isProtectedMode}
        />
      </AccordionDetails>
    </Accordion>
  );
}
