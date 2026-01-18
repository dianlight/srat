import {
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	FormControl,
	FormControlLabel,
	FormLabel,
	MenuItem,
	Select,
	Switch,
	TextField,
	Typography,
	Box,
} from "@mui/material";
import BugReportIcon from "@mui/icons-material/BugReport";
import { useState } from "react";

export type ProblemType = "frontend_ui" | "ha_integration" | "addon" | "samba";

interface ReportIssueDialogProps {
	open: boolean;
	onClose: () => void;
}

interface IssueReportData {
	problemType: ProblemType;
	description: string;
	includeContextData: boolean;
	includeAddonLogs: boolean;
	includeSRATConfig: boolean;
}

export function ReportIssueDialog({ open, onClose }: ReportIssueDialogProps) {
	const [formData, setFormData] = useState<IssueReportData>({
		problemType: "frontend_ui",
		description: "",
		includeContextData: true,
		includeAddonLogs: false,
		includeSRATConfig: false,
	});

	const handleSubmit = async () => {
		// Collect browser context data
		const contextData = {
			currentURL: window.location.href,
			navigationHistory: getNavigationHistory(),
			browserInfo: navigator.userAgent,
			consoleErrors: getConsoleErrors(),
		};

		// Prepare request payload
		const requestPayload = {
			problem_type: formData.problemType,
			description: formData.description,
			include_context_data: formData.includeContextData,
			include_addon_logs: formData.includeAddonLogs,
			include_srat_config: formData.includeSRATConfig,
			...(formData.includeContextData ? contextData : {}),
		};

		try {
			// Call backend API to generate issue report
			const response = await fetch("/api/issues/report", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify(requestPayload),
			});

			if (!response.ok) {
				throw new Error("Failed to generate issue report");
			}

			const data = await response.json();

			// Download attachments if requested
			if (formData.includeSRATConfig && data.sanitized_config) {
				downloadFile(
					data.sanitized_config,
					"srat-config.json",
					"application/json",
				);
			}

			if (formData.includeAddonLogs && data.addon_logs) {
				downloadFile(data.addon_logs, "addon-logs.txt", "text/plain");
			}

			// Open GitHub issue creation page
			window.open(data.github_url, "_blank");

			// Close dialog
			onClose();
		} catch (error) {
			console.error("Error generating issue report:", error);
			alert("Failed to generate issue report. Please try again.");
		}
	};

	const getNavigationHistory = (): string[] => {
		// Get last 5 entries from browser history (if available)
		// Note: Full history access is restricted, so we can only get current URL
		const history: string[] = [];
		if (window.history.length > 0) {
			history.push(window.location.href);
		}
		return history;
	};

	const getConsoleErrors = (): string[] => {
		// Return captured console errors if available
		// This would need to be implemented with a console error interceptor
		return [];
	};

	const downloadFile = (
		content: string,
		filename: string,
		mimeType: string,
	) => {
		const blob = new Blob([content], { type: mimeType });
		const url = URL.createObjectURL(blob);
		const link = document.createElement("a");
		link.href = url;
		link.download = filename;
		document.body.appendChild(link);
		link.click();
		document.body.removeChild(link);
		URL.revokeObjectURL(url);
	};

	const problemTypeLabels: Record<ProblemType, string> = {
		frontend_ui: "Frontend UI Problem",
		ha_integration: "Home Assistant Integration Problem",
		addon: "Addon Problem",
		samba: "Samba Problem",
	};

	return (
		<Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
			<DialogTitle>
				<Box display="flex" alignItems="center" gap={1}>
					<BugReportIcon />
					<Typography variant="h6">Report Issue on GitHub</Typography>
				</Box>
			</DialogTitle>
			<DialogContent>
				<Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
					{/* Problem Type Selector */}
					<FormControl fullWidth>
						<FormLabel>Problem Type</FormLabel>
						<Select
							value={formData.problemType}
							onChange={(e) =>
								setFormData({
									...formData,
									problemType: e.target.value as ProblemType,
								})
							}
						>
							{Object.entries(problemTypeLabels).map(([value, label]) => (
								<MenuItem key={value} value={value}>
									{label}
								</MenuItem>
							))}
						</Select>
					</FormControl>

					{/* Description */}
					<FormControl fullWidth>
						<FormLabel>Description</FormLabel>
						<TextField
							multiline
							rows={6}
							placeholder="Describe the issue in detail. You can use Markdown formatting."
							value={formData.description}
							onChange={(e) =>
								setFormData({ ...formData, description: e.target.value })
							}
							required
						/>
					</FormControl>

					{/* Include Options */}
					<Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
						<Typography variant="subtitle2">Include in Report:</Typography>

						<FormControlLabel
							control={
								<Switch
									checked={formData.includeContextData}
									onChange={(e) =>
										setFormData({
											...formData,
											includeContextData: e.target.checked,
										})
									}
								/>
							}
							label="Contextual data (URL, navigation, browser info, console errors)"
						/>

						<FormControlLabel
							control={
								<Switch
									checked={formData.includeAddonLogs}
									onChange={(e) =>
										setFormData({
											...formData,
											includeAddonLogs: e.target.checked,
										})
									}
								/>
							}
							label="Addon config and logs (from last boot)"
						/>

						<FormControlLabel
							control={
								<Switch
									checked={formData.includeSRATConfig}
									onChange={(e) =>
										setFormData({
											...formData,
											includeSRATConfig: e.target.checked,
										})
									}
								/>
							}
							label="SRAT configuration (sanitized - passwords removed)"
						/>
					</Box>

					<Typography variant="caption" color="text.secondary">
						Note: When you click "Create Issue", diagnostic files will be
						downloaded if requested, and a new GitHub issue page will open with
						pre-filled information. You'll need to manually attach the
						downloaded files to the issue.
					</Typography>
				</Box>
			</DialogContent>
			<DialogActions>
				<Button onClick={onClose}>Cancel</Button>
				<Button
					onClick={handleSubmit}
					variant="contained"
					color="primary"
					disabled={!formData.description.trim()}
				>
					Create Issue
				</Button>
			</DialogActions>
		</Dialog>
	);
}
