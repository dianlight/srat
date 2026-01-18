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
import { Problem_type,
	usePostApiIssuesReportMutation, type IssueReportRequest, type IssueReportResponse } from "../store/sratApi";
import { toast } from "react-toastify";
import { addMessage } from "../store/errorSlice";
import { useAppDispatch } from "../store/store";
import { FormContainer,
	SelectElement,
	SwitchElement,
	TextareaAutosizeElement,
	useForm } from "react-hook-form-mui";

interface ReportIssueDialogProps {
	open: boolean;
	onClose: () => void;
}


export function ReportIssueDialog({ open, onClose }: ReportIssueDialogProps) {
	const formContext = useForm<IssueReportRequest>({
		defaultValues: {
			problem_type: Problem_type.FrontendUi,
			description: "",
			include_context_data: true,
			include_addon_logs: false,
			include_srat_config: false,
		},
	});
	const dispatch = useAppDispatch();
	const [postApiIssuesReport] = usePostApiIssuesReportMutation();


	const handleSubmit = async (formData: IssueReportRequest) => {
		// Collect browser context data
		const contextData = {
			current_url: window.location.href,
			navigation_history: getNavigationHistory(),
			browser_info: navigator.userAgent,
			console_errors: getConsoleErrors()
		};

		// Prepare request payload
		const requestPayload = {
			...formData,
			...(formData.include_context_data ? contextData : {}),
		} as IssueReportRequest;
		
		try {
			postApiIssuesReport({ issueReportRequest: requestPayload }).unwrap()
				.then((res) => {

					const data = res as IssueReportResponse 

					toast.info(
						`Issue ${data.issue_title} created successfully.`,
					);

					// Download attachments if requested
					if (formData.include_srat_config && data.sanitized_config) {
						downloadFile(
							data.sanitized_config,
							"srat-config.json",
							"application/json",
						);
					}

					if (formData.include_addon_logs && data.addon_logs) {
						downloadFile(data.addon_logs, "addon-logs.txt", "text/plain");
					}

					// Open GitHub issue creation page
					window.open(data.github_url, "_blank");

					// Close dialog
					onClose();

					return res;
				})
				.catch((err) => {
					dispatch(addMessage(JSON.stringify(err)));
				});


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

	const problemTypeLabels: Record<Problem_type, string> = {
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
				<FormContainer
					formContext={formContext}
					onSuccess={handleSubmit}
					mode="onChange"
					FormProps={{


						id: "report-issue-form"

					}}
				>
					<Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
						{/* Problem Type Selector */}
						<SelectElement
							label="Problem Type"
							name="problem_type"
							fullWidth
							options={Object.entries(problemTypeLabels).map(([value, label]) => {
								return ({ id: value, label });
							}
							)}
							required
						/>

						{/* Description */}
						<TextareaAutosizeElement
							label="Description"
							name="description"
							resizeStyle="both"
							rows={3}
							fullWidth
							minRows={6}
							placeholder="Describe the issue in detail. You can use Markdown formatting."
							required
						/>

						{/* Include Options */}
						<Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>

							<Typography variant="subtitle2">Include in Report:</Typography>

							<SwitchElement
								name="include_context_data"
								label="Contextual data (URL, navigation, browser info, console errors)"
							/>

							<SwitchElement
								name="include_addon_logs"
								label="Addon config and logs (from last boot)"
							/>


							<SwitchElement
								name="include_srat_config"
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
				</FormContainer>
			</DialogContent>
			<DialogActions>
				<Button onClick={onClose}>Cancel</Button>
				<Button
					type="submit"
					form="report-issue-form"
					variant="contained"
					color="primary"
					disabled={!formContext.formState.isDirty}
				>
					Create Issue
				</Button>
			</DialogActions>
		</Dialog >
	);
}
