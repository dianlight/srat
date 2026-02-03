import {
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	Typography,
	Box,
} from "@mui/material";
import BugReportIcon from "@mui/icons-material/BugReport";
import { useMemo } from "react";
import { Problem_type,
	usePostApiIssuesReportMutation, type IssueReportRequest, type IssueReportResponse } from "../store/sratApi";
import { toast } from "react-toastify";
import { addMessage } from "../store/errorSlice";
import { useAppDispatch } from "../store/store";
import { FormContainer,
	SelectElement,
	SwitchElement,
	TextFieldElement,
	TextareaAutosizeElement,
	useForm } from "react-hook-form-mui";
import { useIssueTemplate } from "../hooks/useIssueTemplate";

interface ReportIssueDialogProps {
	open: boolean;
	onClose: () => void;
}


export function ReportIssueDialog({ open, onClose }: ReportIssueDialogProps) {
	const { template, isLoading: templateLoading } = useIssueTemplate();

	// Define problemTypeLabels first, before using it in useMemo
	const problemTypeLabels: Record<Problem_type, string> = {
		frontend_ui: "Frontend UI Problem",
		ha_integration: "Home Assistant Integration Problem",
		addon: "Addon Problem",
		samba: "Samba Problem",
	};

	const formContext = useForm<IssueReportRequest>({
		defaultValues: {
			problem_type: Problem_type.FrontendUi,
			title: "",
			description: "",
			reproducing_steps: "",
			include_context_data: true,
			include_addon_logs: false,
			include_addon_config: false,
			include_srat_config: false,
			include_database_dump: false,
		},
	});
	const dispatch = useAppDispatch();
	const [postApiIssuesReport] = usePostApiIssuesReportMutation();

	// Extract problem types from template
	const problemTypeOptions = useMemo(() => {
		if (!template || !template.body) {
			return Object.entries(problemTypeLabels).map(([value, label]) => ({ id: value, label }));
		}

		const problemTypeField = template.body.find((field: any) => field?.id === "problem_type");
		if (problemTypeField?.attributes?.options) {
			return problemTypeField.attributes.options.map((option: string) => ({
				id: option.toLowerCase().replace(/\s+/g, "_"),
				label: option,
			}));
		}

		return Object.entries(problemTypeLabels).map(([value, label]) => ({ id: value, label }));
	}, [template, problemTypeLabels]);


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

					/*
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
					*/
					// Open GitHub issue creation page
					try {
						let url = new URL(data.github_url);
						console.log("GitHub URL for issue creation:", url, data.github_url.length);
						let result = window.open(url, "_blank");
						if (result === null) {
							// Open a popup blocked dialog with link to the URL
							alert(
								`Popup blocked! Please click the link to create the issue: ${url}`,
							);
						}
						toast.info(
							`Issue ${data.issue_title} created successfully.`,
						);
					} catch (error) {
						console.error(error);
						dispatch(addMessage(JSON.stringify(error)));
						toast.error(
							`Unable to create issue: ${error?.toString() || "Unknown error"}`,
							{
								autoClose: false,
								type: "error",
								data: error,
							}
						);
					}

					// Close dialog
					onClose();

					return res;
				})
				.catch((err) => {
					dispatch(addMessage(JSON.stringify(err)));
					toast.error(
						`Unable to create issue: ${err?.toString() || "Unknown error"}`,
						{
							autoClose: false,
							type: "error",
							data: err,
						}
					);
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

	return (
		<Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
			<DialogTitle>
				<Box display="flex" alignItems="center" gap={1}>
					<BugReportIcon />
					<Typography variant="h6">
						{template?.name || "Report Issue on GitHub"}
					</Typography>
				</Box>
			</DialogTitle>
			<DialogContent>
				{templateLoading ? (
					<Box sx={{ display: "flex", justifyContent: "center", p: 3 }}>
						<Typography>Loading template...</Typography>
					</Box>
				) : (
					<FormContainer
						formContext={formContext}
						onSuccess={handleSubmit}
						mode="onChange"
						FormProps={{
							id: "report-issue-form"
						}}
					>
						<Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
							<TextFieldElement
								label="Title"
								name="title"
								fullWidth
								required
							/>

							{/* Problem Type Selector */}
							<SelectElement
								label="Problem Type"
								name="problem_type"
								fullWidth
								options={problemTypeOptions}
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

							<TextareaAutosizeElement
								label="Reproducing Steps"
								name="reproducing_steps"
								resizeStyle="both"
								rows={3}
								fullWidth
								minRows={4}
								placeholder="List the steps needed to reproduce the issue."
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
									label="Addon logs (from last boot)"
								/>

								<SwitchElement
									name="include_addon_config"
									label="Addon configuration (sanitized - passwords removed)"
								/>


								<SwitchElement
									name="include_srat_config"
									label="SRAT configuration (sanitized - passwords removed)"
								/>

								<SwitchElement
									name="include_database_dump"
									label="Database dump (sanitized - passwords removed)"
								/>

							</Box>

							<Typography variant="caption" color="text.secondary">
								Note: When you click "Create Issue", diagnostic files will be
								downloaded if requested, and a new GitHub issue page will open with
								pre-filled information. You'll need to manually attach the
								downloaded files to the issue.
							</Typography>
						</Box>
					</FormContainer>)}			</DialogContent>
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
