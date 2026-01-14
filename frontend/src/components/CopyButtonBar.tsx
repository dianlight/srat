import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import CodeIcon from "@mui/icons-material/Code";
import Box from "@mui/material/Box";
import { toast } from "react-toastify";

export interface CopyButtonBarProps {
	/**
	 * Content to copy as plain text
	 */
	plainTextContent: string;
	
	/**
	 * Content to copy as markdown (optional, if not provided uses plainTextContent)
	 */
	markdownContent?: string;
	
	/**
	 * Title to prepend to markdown content (optional)
	 */
	markdownTitle?: string;
	
	/**
	 * Show only icon buttons (compact mode)
	 */
	compact?: boolean;
	
	/**
	 * Additional CSS styles
	 */
	sx?: any;
}

/**
 * Reusable component for copy functionality
 * Provides buttons to copy content as plain text or markdown
 */
export function CopyButtonBar(props: CopyButtonBarProps) {
	const { plainTextContent, markdownContent, markdownTitle, compact = false, sx = {} } = props;
	
	const handleCopyPlainText = async () => {
		try {
			await navigator.clipboard.writeText(plainTextContent);
			toast.success("Copied as plain text to clipboard");
		} catch (error) {
			toast.error("Failed to copy to clipboard");
		}
	};

	const handleCopyMarkdown = async () => {
		try {
			let markdown = markdownContent || plainTextContent;
			if (markdownTitle) {
				markdown = `## ${markdownTitle}\n\n${markdown}`;
			}
			await navigator.clipboard.writeText(markdown);
			toast.success("Copied as markdown to clipboard");
		} catch (error) {
			toast.error("Failed to copy to clipboard");
		}
	};

	if (compact) {
		// Icon buttons only for title bars
		return (
			<Box sx={{ display: 'flex', gap: 1, ...sx }}>
				<Tooltip title="Copy as plain text">
					<IconButton
						size="small"
						onClick={handleCopyPlainText}
						aria-label="copy as plain text"
					>
						<ContentCopyIcon fontSize="small" />
					</IconButton>
				</Tooltip>
				<Tooltip title="Copy as markdown">
					<IconButton
						size="small"
						onClick={handleCopyMarkdown}
						aria-label="copy as markdown"
					>
						<CodeIcon fontSize="small" />
					</IconButton>
				</Tooltip>
			</Box>
		);
	}

	// Full buttons for action bars
	return (
		<Box sx={{ display: 'flex', gap: 1, ...sx }}>
			<Tooltip title="Copy as plain text">
				<Button
					onClick={handleCopyPlainText}
					variant="outlined"
					startIcon={<ContentCopyIcon />}
				>
					Copy
				</Button>
			</Tooltip>
			<Tooltip title="Copy as markdown for GitHub issues">
				<Button
					onClick={handleCopyMarkdown}
					variant="outlined"
					startIcon={<CodeIcon />}
				>
					Copy as Markdown
				</Button>
			</Tooltip>
		</Box>
	);
}
