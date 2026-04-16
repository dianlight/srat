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
import { useVolume } from "../../hooks/volumeHook";
import { TabIDs } from "../../store/locationState";
import {
  type Partition,
  Severity2,
  Status,
  useDeleteApiProblemsByProblemKeyMutation,
  useGetApiProblemsQuery,
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

  const mergedProblems = useMemo(() => {
    const baseProblems = Array.isArray(problems) ? problems : [];

    const incomingProblem = evdata?.problem;
    if (!incomingProblem) {
      return baseProblems;
    }

    const existingIndex = baseProblems.findIndex(
      (problem) => problem?.problem_key === incomingProblem.problem_key,
    );

    const isRemovedStatus =
      incomingProblem.status === Status.Dismissed ||
      incomingProblem.status === Status.Deleted;

    if (isRemovedStatus) {
      if (existingIndex < 0) {
        return baseProblems;
      }

      return baseProblems.filter(
        (problem) => problem?.problem_key !== incomingProblem.problem_key,
      );
    }

    if (existingIndex < 0) {
      return [incomingProblem, ...baseProblems];
    }

    return baseProblems.map((problem, index) =>
      index === existingIndex ? incomingProblem : problem,
    );
  }, [problems, evdata?.problem]);

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
              //console.log("Adding share action for partition", partition.id, partition);
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
        {evdata?.hello?.protected_mode ? (
          <>
            <IssueCard
              key="protected-mode"
              issue={{
                id: -1,
                problem_key: "protected_mode",
                title: "Addon in Protected Mode",
                description:
                  "The addon is currently in protected mode. In this mode, no disks can be mounted to prevent unauthorized access. To disable protected mode, navigate to the addon settings in your Home Assistant interface and toggle the protected mode option off. Ensure you understand the security implications before disabling.",
                severity: Severity2.Error,
                status: Status.Created,
                ignored: false,
                repeating: 0,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
              }}
              showIgnored={false}
            />
            <ActionableItemsList
              actionablePartitions={actionablePartitions}
              isLoading={isLoading}
              error={error}
              showIgnored={showIgnored}
              disabled={true}
            />
          </>
        ) : (
          <>
            {(
              mergedProblems.filter(Boolean) as NonNullable<
                (typeof mergedProblems)[number]
              >[]
            ).map((issue) => (
              <IssueCard
                key={issue.problem_key}
                issue={issue}
                onResolve={handleResolveIssue}
                showIgnored={showIgnored}
              />
            ))}
            <ActionableItemsList
              actionablePartitions={actionablePartitions}
              isLoading={isLoading}
              error={error}
              showIgnored={showIgnored}
            />
          </>
        )}
      </AccordionDetails>
    </Accordion>
  );
}
