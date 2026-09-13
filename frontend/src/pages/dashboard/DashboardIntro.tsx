import CampaignOutlinedIcon from "@mui/icons-material/CampaignOutlined";
import KeyboardArrowLeftIcon from "@mui/icons-material/KeyboardArrowLeft";
import KeyboardArrowRightIcon from "@mui/icons-material/KeyboardArrowRight";
import NewReleasesOutlinedIcon from "@mui/icons-material/NewReleasesOutlined";
import {
  Alert,
  Box,
  Card,
  CardContent,
  CardHeader,
  CircularProgress,
  Collapse,
  IconButton,
  Link,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Typography,
} from "@mui/material";
import { useEffect, useRef } from "react";
import type { NewsItem } from "../../hooks/githubNewsHook";
import { GITHUB_ANNOUNCEMENTS_URL } from "../../store/githubRestApi";
import {
  Standard_share_names,
  useGetApiSettingsQuery,
} from "../../store/sratApi";
import { testIds } from "../../testIds";

interface DashboardIntroProps {
  isCollapsed: boolean;
  onToggleCollapse: () => void;
  news: NewsItem[];
  isLoading: boolean;
  error: Error | null;
}

export function DashboardIntro({
  isCollapsed,
  onToggleCollapse,
  news,
  isLoading,
  error,
}: DashboardIntroProps) {
  const initialCheckDone = useRef(false);
  const { data: settings } = useGetApiSettingsQuery();

  useEffect(() => {
    // Once news has loaded, if there are news items, expand the intro panel.
    // This should only happen on the initial load.
    if (!isLoading && !initialCheckDone.current) {
      if (news.length > 0) {
        onToggleCollapse();
      }
      initialCheckDone.current = true;
    }
  }, [news, isLoading, onToggleCollapse]);

  const seeAllNewsLink = (
    <Link
      href={GITHUB_ANNOUNCEMENTS_URL}
      target="_blank"
      rel="noopener noreferrer"
      underline="hover"
      data-testid={testIds.dashboard.newsSeeAll}
    >
      See all news
    </Link>
  );

  const renderNews = () => {
    if (isLoading) {
      return (
        <Box sx={{ display: "flex", justifyContent: "center", mt: 2 }}>
          <CircularProgress size={24} />
        </Box>
      );
    }
    if (error) {
      return (
        <Box>
          <Alert severity="warning" sx={{ mt: 2 }}>
            Could not load project news.
          </Alert>
          <Box sx={{ mt: 1 }}>{seeAllNewsLink}</Box>
        </Box>
      );
    }

    return (
      <Box>
        <Typography variant="body2" sx={{ mt: 2, mb: 1 }}>
          <strong>Latest News:</strong>
        </Typography>
        {news.length > 0 ? (
          <List dense>
            {news.map((item) => (
              <ListItem
                key={item.id}
                disablePadding
                alignItems="flex-start"
                sx={{ gap: 1 }}
              >
                <ListItemIcon sx={{ minWidth: 32, mt: 0.5 }}>
                  {item.type === "release" ? (
                    <NewReleasesOutlinedIcon
                      fontSize="small"
                      data-testid={testIds.dashboard.newsReleaseIcon}
                    />
                  ) : (
                    <CampaignOutlinedIcon
                      fontSize="small"
                      data-testid={testIds.dashboard.newsAnnouncementIcon}
                    />
                  )}
                </ListItemIcon>
                <ListItemText
                  primary={
                    <Link
                      href={item.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      underline="hover"
                    >
                      {item.title}
                    </Link>
                  }
                  secondary={
                    <Typography
                      variant="body2"
                      color="text.secondary"
                      component="span"
                      sx={{ display: "block" }}
                      data-testid={testIds.dashboard.newsAbstract}
                    >
                      {item.abstract}
                    </Typography>
                  }
                />
              </ListItem>
            ))}
          </List>
        ) : null}
        <Box sx={{ mt: 1 }}>{seeAllNewsLink}</Box>
      </Box>
    );
  };

  return (
    <Card
      sx={{
        height: "100%", // Auto height when collapsed, 100% when expanded
        display: "flex",
        flexDirection: "column",
        width: "100%",
      }}
    >
      {isCollapsed ? (
        // Collapsed view
        <Box
          sx={{
            display: "flex",
            flexDirection: "column",
            alignItems: "center", // Center items horizontally within this box
            justifyContent: "flex-start", // Align items to the top
            height: "100%", // This box should also fill the height
            width: "100%", // This box should also fill the width
          }}
        >
          <IconButton
            aria-label="expand"
            onClick={onToggleCollapse}
            sx={{
              mb: 1, // Margin below the button
            }}
          >
            <KeyboardArrowRightIcon />
          </IconButton>
          <Typography
            variant="h6"
            sx={{
              writingMode: "vertical-lr", // Text flows top-to-bottom, then left-to-right columns
              textOrientation: "upright", // Characters are upright
              whiteSpace: "nowrap", // Prevent wrapping
              flexGrow: 1, // Allow it to take available space and push button/content apart
              display: "flex", // Use flexbox for internal centering
              alignItems: "center", // Center vertically within its flex item
              justifyContent: "center", // Center horizontally within its flex item
              fontSize: "1rem", // Adjust font size to fit narrow column
            }}
          >
            Welcome to SRAT
          </Typography>
        </Box>
      ) : (
        // Expanded view
        <>
          <CardHeader
            title="Welcome to SRAT"
            slotProps={{
              Title: {
                sx: { variant: "h6", fontWeight: "bold" },
              },
            }}
            action={
              // Button on the right side of the header
              <IconButton aria-label="collapse" onClick={onToggleCollapse}>
                <KeyboardArrowLeftIcon />
              </IconButton>
            }
          />
          <Collapse in={!isCollapsed}>
            {" "}
            {/* Content collapses/expands */}
            <CardContent
              sx={{ flexGrow: 1, display: "flex", flexDirection: "column" }}
            >
              <Typography variant="body1">
                This is your storage management dashboard. Here you can get a
                quick overview of your system's storage health and perform
                common actions.
              </Typography>
              {settings &&
                "standard_share_names" in settings &&
                settings.standard_share_names === Standard_share_names.Old && (
                  <Alert severity="warning" sx={{ mt: 2 }}>
                    You are using the legacy share names (addons,
                    addon_configs). These names are deprecated and will be
                    removed in a future release. Switch to the new names
                    (local_apps, app_configs) in the Settings page.
                  </Alert>
                )}
              <Box sx={{ flexGrow: 1 }} />{" "}
              {/* Pushes "Latest News" to the bottom */}
              {renderNews()}
            </CardContent>
          </Collapse>
        </>
      )}
    </Card>
  );
}
