import KeyboardArrowLeftIcon from "@mui/icons-material/KeyboardArrowLeft";
import KeyboardArrowRightIcon from "@mui/icons-material/KeyboardArrowRight";
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
	ListItemText,
	Typography,
} from "@mui/material";
import { useEffect, useRef } from "react";
import type { NewsItem } from "../../hooks/githubNewsHook";

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
				<Alert severity="warning" sx={{ mt: 2 }}>
					Could not load project news.
				</Alert>
			);
		}
		if (news.length === 0) {
			return null; // Don't show the news section if there are no recent news
		}

		return (
			<Box>
				<Typography variant="body2" sx={{ mt: 2, mb: 1 }}>
					<strong>Latest News:</strong>
				</Typography>
				<List dense>
					{news.map((item) => (
						<ListItem key={item.id} disablePadding>
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
							/>
						</ListItem>
					))}
				</List>
			</Box>
		);
	};

	return (
		<Card
			sx={{
				height: "100%", // Auto height when collapsed, 100% when expanded
				display: "flex",
				flexDirection: "column",
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
						titleTypographyProps={{ variant: "h6" }}
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
