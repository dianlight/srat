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
import IssueCard from "../../components/IssueCard";
import { useLabFeatures } from "../../hooks/useLabFeatures";
import { useVolume } from "../../hooks/volumeHook";
import { TabIDs } from "../../store/locationState";
import {
  type Partition,
  type Problem,
  Status,
  useDeleteApiProblemsByProblemKeyMutation,
  useGetApiProblemsQuery,
  usePutApiProblemsByProblemKeyMutation,
} from "../../store/sratApi";
import { useGetServerEventsQuery } from "../../store/wsApi";
import { TourEvents, TourEventTypes } from "../../utils/TourEvents";
import { ActionableItemsList } from "./components/ActionableItemsList";

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

  const mergedProblems = useMemo(() => {
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

    if (existingIndex < 0) {
      return [incomingProblem, ...visibleBase];
    }

    return visibleBase.map((problem, index) =>
      index === existingIndex ? incomingProblem : problem,
    );
  }, [problems, evdata?.problem, customComponentLabActive]);

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
      void dismissProblem({ problemKey: id });
    }
  }

  // Permanent ignore: the backend stores the flag and never re-raises the
  // alert (or notifies HA) until it is dismissed/re-enabled.
  function handleIgnoreIssue(issue: Problem): void {
    const key = issue.problem_key;
    if (!key) {
      return;
    }
    void upsertProblem({
      problemKey: key,
      problem: { ...issue, ignored: true, status: Status.Ignored },
    });
  }

  // Re-enable a previously ignored alert. The problem is recreated right
  // away when its condition still holds (e.g. protected mode still on).
  function handleReenableIssue(id: number | string): void {
    if (typeof id !== "string") {
      return;
    }
    const issue = mergedProblems.find((problem) => problem?.problem_key === id);
    if (issue?.problem_key) {
      void upsertProblem({
        problemKey: issue.problem_key,
        problem: { ...issue, ignored: false, status: Status.Created },
      });
    } else {
      void dismissProblem({ problemKey: id });
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
