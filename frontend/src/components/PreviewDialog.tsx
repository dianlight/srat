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
import { CopyButtonBar } from "./CopyButtonBar";
import { 
	isSensitiveField, 
	censorValue, 
	objectToPlainText, 
	objectToMarkdown 
} from "../utils/censorshipUtils";

export interface PreviewDialogProps {
	open: boolean;
	onClose: () => void;
	title: string;
	objectToDisplay: any;
}

export function PreviewDialog(props: PreviewDialogProps) {
	const { onClose, open } = props;

	const handleClose = () => {
		onClose();
	};

	const plainText = objectToPlainText(props.objectToDisplay);
	const markdown = objectToMarkdown(props.objectToDisplay);

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
					<CopyButtonBar
						compact
						plainTextContent={plainText}
						markdownContent={markdown}
						markdownTitle={props.title}
					/>
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
				<CopyButtonBar
					plainTextContent={plainText}
					markdownContent={markdown}
					markdownTitle={props.title}
				/>
				<Button onClick={handleClose} autoFocus variant="outlined" color="secondary">
					Close
				</Button>
			</DialogActions>
		</Dialog>
	);
}

// Utility functions for privacy and security are now in utils/censorshipUtils.ts

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
