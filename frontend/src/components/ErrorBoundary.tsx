import BugReportIcon from "@mui/icons-material/BugReport";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import {
	Accordion,
	AccordionDetails,
	AccordionSummary,
	Alert,
	AlertTitle,
	Box,
	Button,
	Typography,
} from "@mui/material";
import React, { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
	children: ReactNode;
}

interface State {
	hasError: boolean;
	error: Error | null;
	errorInfo: ErrorInfo | null;
}

export class ErrorBoundary extends Component<Props, State> {
	public state: State = {
		hasError: false,
		error: null,
		errorInfo: null,
	};

	public static getDerivedStateFromError(error: Error): State {
		// Update state so the next render will show the fallback UI.
		return { hasError: true, error, errorInfo: null };
	}

	public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
		// You can also log the error to an error reporting service
		console.error("Uncaught error:", error, errorInfo);
		this.setState({ errorInfo });
	}

	private handleReload = () => {
		window.location.reload();
	};

	public render() {
		if (this.state.hasError) {
			return (
				<Box
					sx={{
						p: 3,
						display: "flex",
						justifyContent: "center",
						alignItems: "center",
						minHeight: "80vh",
					}}
				>
					<Alert
						severity="error"
						icon={<BugReportIcon fontSize="inherit" />}
						action={
							<Button color="inherit" size="small" onClick={this.handleReload}>
								Reload Page
							</Button>
						}
						sx={{ maxWidth: "800px", width: "100%" }}
					>
						<AlertTitle>Oops! Something went wrong.</AlertTitle>
						<Typography variant="body1">
							An unexpected error occurred in this section. You can try
							reloading the page to fix it.
						</Typography>
						{this.state.error && (
							<Accordion
								sx={{
									mt: 2,
									bgcolor: "transparent",
									boxShadow: "none",
									"&:before": { display: "none" },
									"&.Mui-expanded": { margin: 0 },
									"& .MuiAccordionSummary-root": { p: 0, minHeight: "auto" },
									"& .MuiAccordionSummary-content": { m: 0 },
									"& .MuiAccordionDetails-root": { p: 0, pt: 1 },
								}}
							>
								<AccordionSummary expandIcon={<ExpandMoreIcon />}>
									<Typography>Error Details</Typography>
								</AccordionSummary>
								<AccordionDetails>
									<Box
										component="pre"
										sx={{
											whiteSpace: "pre-wrap",
											wordBreak: "break-all",
											fontFamily: "monospace",
											fontSize: "0.875rem",
										}}
									>
										{this.state.error.toString()}
										<br />
										{this.state.errorInfo?.componentStack}
									</Box>
								</AccordionDetails>
							</Accordion>
						)}
					</Alert>
				</Box>
			);
		}

		return this.props.children;
	}
}
