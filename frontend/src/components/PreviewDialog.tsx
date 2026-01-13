import Button from "@mui/material/Button";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import { TreeItem } from "@mui/x-tree-view/TreeItem";
import { SimpleTreeView } from '@mui/x-tree-view/SimpleTreeView';
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import ChevronRightIcon from "@mui/icons-material/ChevronRight";
import Box from "@mui/material/Box";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import CodeIcon from "@mui/icons-material/Code";
import { toast } from "react-toastify";

export interface PreviewDialogProps {
	open: boolean;
	onClose: () => void;
	title: string;
	objectToDisplay: any;
}

// Helper functions to generate plain text and markdown
function objectToPlainText(obj: any, indent = 0, label = 'Root'): string {
	if (obj === undefined || obj === null || obj === "") {
		return '';
	}

	const indentStr = '  '.repeat(indent);
	const isSensitive = isSensitiveField(label);
	let result = '';

	if (typeof obj === "string" || typeof obj === "number") {
		const displayValue = isSensitive ? censorValue(obj) : obj;
		result += `${indentStr}${label}: ${displayValue} (${typeof obj}${isSensitive ? ', censored' : ''})\n`;
	} else if (typeof obj === "boolean") {
		result += `${indentStr}${label}: ${obj ? "Yes" : "No"} (boolean)\n`;
	} else if (Array.isArray(obj)) {
		result += `${indentStr}${label} (array[${obj.length}]):\n`;
		obj.forEach((item, index) => {
			result += objectToPlainText(item, indent + 1, `[${index}]`);
		});
	} else if (typeof obj === "object") {
		const keys = Object.getOwnPropertyNames(obj);
		if (label !== 'Root') {
			result += `${indentStr}${label} (object):\n`;
		}
		keys.forEach(key => {
			const value = Object.getOwnPropertyDescriptor(obj, key)?.value;
			result += objectToPlainText(value, indent + (label !== 'Root' ? 1 : 0), key);
		});
	} else {
		result += `${indentStr}${label}: Unknown type (${typeof obj})\n`;
	}

	return result;
}

function objectToMarkdown(obj: any, indent = 0, label = 'Root'): string {
	if (obj === undefined || obj === null || obj === "") {
		return '';
	}

	const indentStr = '  '.repeat(indent);
	const isSensitive = isSensitiveField(label);
	let result = '';

	if (typeof obj === "string" || typeof obj === "number") {
		const displayValue = isSensitive ? censorValue(obj) : obj;
		result += `${indentStr}- **${label}**: \`${displayValue}\` _(${typeof obj}${isSensitive ? ', censored' : ''})_\n`;
	} else if (typeof obj === "boolean") {
		result += `${indentStr}- **${label}**: ${obj ? "Yes" : "No"} _(boolean)_\n`;
	} else if (Array.isArray(obj)) {
		result += `${indentStr}- **${label}** _(array[${obj.length}])_:\n`;
		obj.forEach((item, index) => {
			result += objectToMarkdown(item, indent + 1, `[${index}]`);
		});
	} else if (typeof obj === "object") {
		const keys = Object.getOwnPropertyNames(obj);
		if (label !== 'Root') {
			result += `${indentStr}- **${label}** _(object)_:\n`;
		}
		keys.forEach(key => {
			const value = Object.getOwnPropertyDescriptor(obj, key)?.value;
			result += objectToMarkdown(value, indent + (label !== 'Root' ? 1 : 0), key);
		});
	} else {
		result += `${indentStr}- **${label}**: Unknown type (${typeof obj})\n`;
	}

	return result;
}

export function PreviewDialog(props: PreviewDialogProps) {
	const { onClose, open } = props;

	const handleClose = () => {
		onClose();
	};

	const handleCopyPlainText = async () => {
		try {
			const plainText = objectToPlainText(props.objectToDisplay);
			await navigator.clipboard.writeText(plainText);
			toast.success("Copied as plain text to clipboard");
		} catch (error) {
			toast.error("Failed to copy to clipboard");
		}
	};

	const handleCopyMarkdown = async () => {
		try {
			const markdown = `## ${props.title ?? 'Preview'}\n\n${objectToMarkdown(props.objectToDisplay)}`;
			await navigator.clipboard.writeText(markdown);
			toast.success("Copied as markdown to clipboard");
		} catch (error) {
			toast.error("Failed to copy to clipboard");
		}
	};

	return (
		<Dialog
			sx={{
				"& .MuiDialogContent-root": {
					p: 0,
				},
			}}
			maxWidth="md"
			open={open}
			onClose={handleClose}
			aria-labelledby="alert-dialog-title"
			aria-describedby="alert-dialog-description"
		>
			<DialogTitle id="alert-dialog-title">
				<Box display="flex" alignItems="center" justifyContent="space-between">
					<span>{props.title ?? "Preview"}</span>
					<Box sx={{ display: 'flex', gap: 1 }}>
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
				</Box>
			</DialogTitle>
			<DialogContent>
				<ObjectTree object={props.objectToDisplay} />
			</DialogContent>
			<DialogActions sx={{
				position: 'sticky',
				bottom: 0,
				backgroundColor: 'background.paper',
				borderTop: 1,
				borderColor: 'divider',
				zIndex: 1
			}}>
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
				<Button onClick={handleClose} autoFocus variant="outlined" color="secondary">
					Close
				</Button>
			</DialogActions>
		</Dialog>
	);
}

// Utility functions for privacy and security
const SENSITIVE_KEYWORDS = [
	'password', 'pass', 'pwd', 'secret', 'token', 'key', 'auth', 'credential',
	'private', 'confidential', 'secure', 'api_key', 'apikey', 'access_token',
	'refresh_token', 'jwt', 'bearer', 'authorization', 'salt', 'hash'
];

function isSensitiveField(label: string): boolean {
	const lowerLabel = label.toLowerCase();
	return SENSITIVE_KEYWORDS.some(keyword =>
		lowerLabel.includes(keyword)
	);
}

function censorValue(value: any): string {
	// Use lock emoji to censor sensitive data
	if (typeof value === 'string') {
		return '🔒'.repeat(Math.min(value.length, 8));
	}
	return '🔒'.repeat(8);
}

function ObjectTreeNode(props: { value: any; nodeId: string; label: string }) {
	const { value, nodeId, label } = props;

	if (value === undefined || value === null || value === "") {
		return null;
	}

	const isSensitive = isSensitiveField(label);

	if (typeof value === "string" || typeof value === "number") {
		const displayValue = isSensitive ? censorValue(value) : value;
		const valueColor = isSensitive ? 'error.main' : 'text.primary';

		return (
			<TreeItem
				itemId={nodeId}
				label={
					<Box component="span">
						<Box component="span" sx={{ color: 'primary.main', fontWeight: 'medium' }}>
							{label}
						</Box>
						<Box component="span" sx={{ color: valueColor, fontFamily: isSensitive ? 'monospace' : 'inherit' }}>
							: {displayValue}
						</Box>
						<Box component="span" sx={{ color: 'text.secondary', fontSize: '0.875em' }}>
							{' '}({typeof value}{isSensitive ? ', censored' : ''})
						</Box>
					</Box>
				}
			/>
		);
	}

	if (typeof value === "boolean") {
		return (
			<TreeItem
				itemId={nodeId}
				label={
					<Box component="span">
						<Box component="span" sx={{ color: 'primary.main', fontWeight: 'medium' }}>
							{label}
						</Box>
						<Box component="span" sx={{ color: 'text.primary' }}>
							: {value ? "Yes" : "No"}
						</Box>
						<Box component="span" sx={{ color: 'text.secondary', fontSize: '0.875em' }}>
							{' '}(boolean)
						</Box>
					</Box>
				}
			/>
		);
	}

	if (Array.isArray(value)) {
		return (
			<TreeItem
				itemId={nodeId}
				label={
					<Box component="span">
						<Box component="span" sx={{ color: 'primary.main', fontWeight: 'medium' }}>
							{label}
						</Box>
						<Box component="span" sx={{ color: 'text.secondary', fontSize: '0.875em' }}>
							{' '}(array[{value.length}])
						</Box>
					</Box>
				}
			>
				{value.map((item, index) => (
					<ObjectTreeNode
						key={`${nodeId}.${index}`}
						value={item}
						nodeId={`${nodeId}.${index}`}
						label={`[${index}]`}
					/>
				))}
			</TreeItem>
		);
	}

	if (typeof value === "object") {
		const keys = Object.getOwnPropertyNames(value);
		return (
			<>
				{nodeId === "root" ? keys.map((key, index) => (
					<ObjectTreeNode
						key={`${nodeId}.${index}`}
						value={Object.getOwnPropertyDescriptor(value, key)?.value}
						nodeId={`${nodeId}.${index}`}
						label={key}
					/>
				)) : (
					<TreeItem
						itemId={nodeId}
						label={
							<Box component="span">
								<Box component="span" sx={{ color: 'primary.main', fontWeight: 'medium' }}>
									{label}
								</Box>
								<Box component="span" sx={{ color: 'text.secondary', fontSize: '0.875em' }}>
									{' '}(object)
								</Box>
							</Box>
						}
					>
						{keys.map((key, index) => (
							<ObjectTreeNode
								key={`${nodeId}.${index}`}
								value={Object.getOwnPropertyDescriptor(value, key)?.value}
								nodeId={`${nodeId}.${index}`}
								label={key}
							/>
						))}
					</TreeItem>
				)}
			</>
		);
	}

	return (
		<TreeItem
			itemId={nodeId}
			label={
				<Box component="span">
					<Box component="span" sx={{ color: 'primary.main', fontWeight: 'medium' }}>
						{label}
					</Box>
					<Box component="span" sx={{ color: 'text.secondary', fontSize: '0.875em' }}>
						: Unknown type ({typeof value})
					</Box>
				</Box>
			}
		/>
	);
}

export function ObjectTree(props: {
	object: object | Array<any> | null | undefined;
}) {
	if (!props.object) {
		return <Box p={2}>No data to display</Box>;
	}

	return (
		<SimpleTreeView
			sx={{ flexGrow: 1, maxWidth: '100%', overflowY: 'auto', p: 1 }}
		>
			<ObjectTreeNode
				value={props.object}
				nodeId="root"
				label="Root"
			/>
		</SimpleTreeView>
	);
}
