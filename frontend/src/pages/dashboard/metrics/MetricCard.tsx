import InfoIcon from "@mui/icons-material/Info";
import {
	Alert,
	Box,
	Card,
	CardContent,
	CardHeader,
	CircularProgress,
	IconButton,
	Tooltip,
	Typography,
	useTheme,
	type SxProps,
	type Theme,
} from "@mui/material";
import { SparkLineChart } from "@mui/x-charts/SparkLineChart";

const MAX_HISTORY_LENGTH = 10;

export interface MetricCardProps {
	title: string;
	subheader?: string;
	value: string;
	history?: number[];
	isLoading: boolean;
	error: boolean;
	historyType?: "line" | "bar";
	children?: React.ReactNode;
	detailMetricId?: string;
	onDetailClick?: (metricId: string) => void;
	sx?: SxProps<Theme>;
}

export function MetricCard({
	title,
	subheader,
	value,
	history,
	isLoading,
	error,
	historyType = "line",
	children,
	detailMetricId,
	onDetailClick,
	sx,
}: MetricCardProps) {
	const theme = useTheme();

	const handleDetailClick = () => {
		if (onDetailClick && detailMetricId) {
			onDetailClick(detailMetricId);
		}
	};

	const renderHistory = () => {
		if (history && history.length <= 1) {
			return <Typography variant="caption">gathering data...</Typography>;
		} else if (!history) {
			return "";
		}

		const chartData = history.slice(-MAX_HISTORY_LENGTH);

		if (historyType === "bar") {
			return (
				<SparkLineChart
					data={chartData}
					width={100}
					height={40}
					plotType="bar"
					color={theme.palette.info.main}
					showTooltip
					showHighlight
				/>
			);
		}

		return (
			<SparkLineChart
				data={chartData}
				width={100}
				height={40}
				color={theme.palette.primary.main}
				showTooltip
				showHighlight
			/>
		);
	};

	const renderContent = () => {
		if (isLoading) {
			return (
				<Box
					sx={[{
						display: "flex",
						justifyContent: "center",
						alignItems: "center",
					},
					...(Array.isArray(sx) ? sx : [sx])
					]}
				>
					<CircularProgress />
				</Box>
			);
		}

		if (error) {
			return <Alert severity="warning">{title} data not available.</Alert>;
		}

		return (
			<>
				<Box
					sx={[{
						display: "flex",
						flexDirection: { xs: "column", sm: "row" },
						alignItems: { xs: "stretch", sm: "center" },
						justifyContent: "space-between",
						mb: history ? { xs: 0.5, sm: 1 } : 0,
						gap: history ? { xs: 1, sm: 2 } : 0,
					}
					]}
				>
					<Typography
						variant="h4"
						component="div"
						sx={{
							textAlign: { xs: "center", sm: "left" }
						}}
					>
						{value}
					</Typography>
					<Box sx={{
						width: { xs: "100%", sm: "40%" },
						height: history ? 40 : 0,
						minWidth: history ? 100 : 'auto',
						display: history ? 'block' : 'none'
					}}>
						{renderHistory()}
					</Box>
				</Box>
				{children}
			</>
		);
	};

	return (
		<Card sx={[{ p: { xs: 0.5, sm: 1 } }, ...(Array.isArray(sx) ? sx : [sx])]}>
			<CardHeader
				title={title}
				subheader={subheader}
				sx={[{
					p: { xs: 1, sm: 1.5 },
					"& .MuiCardHeader-content": {
						overflow: "hidden",
						my: { xs: -0.5, sm: 0 }
					},
					"& .MuiCardHeader-title": {
						fontSize: { xs: "1rem", sm: "1.25rem" },
						whiteSpace: "nowrap",
						overflow: "hidden",
						textOverflow: "ellipsis"
					},
					"& .MuiCardHeader-subheader": {
						fontSize: { xs: "0.75rem", sm: "0.875rem" },
						whiteSpace: "nowrap",
						overflow: "hidden",
						textOverflow: "ellipsis"
					}
				}, ...(Array.isArray(sx) ? sx : [sx])]}
				action={
					detailMetricId &&
					onDetailClick && (
						<Tooltip title="Show Details">
							<IconButton
								onClick={handleDetailClick}
								size="small"
								sx={{ ml: { xs: 0.5, sm: 1 } }}
							>
								<InfoIcon sx={{ fontSize: { xs: "1.25rem", sm: "1.5rem" } }} />
							</IconButton>
						</Tooltip>
					)
				}
			/>
			<CardContent sx={[{
				p: { xs: 1, sm: 2 },
				"&:last-child": {
					pb: { xs: 1, sm: 2 }
				}
			}, ...(Array.isArray(sx) ? sx : [sx])]}>
				{renderContent()}
			</CardContent>
		</Card>
	);
}
