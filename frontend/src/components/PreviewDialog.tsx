import Button from "@mui/material/Button";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Paper from "@mui/material/Paper";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";

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

	return (
		<Dialog
			open={open}
			onClose={handleClose}
			aria-labelledby="alert-dialog-title"
			aria-describedby="alert-dialog-description"
		>
			<DialogTitle id="alert-dialog-title">Debug: {props.title}</DialogTitle>
			<DialogContent>
				<ObjectTable object={props.objectToDisplay} key={0} />
			</DialogContent>
			<DialogActions>
				<Button onClick={handleClose} autoFocus>
					Close
				</Button>
			</DialogActions>
		</Dialog>
	);
}

function ObjectField(props: { value: any; id?: string; nkey?: string }) {
	//console.log("ObjectField got", props.nkey, props.value, typeof props.value)
	if (props.value === undefined || props.value === null || props.value === "") {
		return <></>;
	} else if (
		typeof props.value === "string" ||
		typeof props.value === "number"
	) {
		return (
			<TableRow key={`${props.id}-${props.nkey}`}>
				<TableCell>{props.nkey}</TableCell>
				<TableCell align="right">{props.value}</TableCell>
			</TableRow>
		);
	} else if (typeof props.value === "boolean") {
		return (
			<TableRow key={`${props.id}-${props.nkey}`}>
				<TableCell>{props.nkey}</TableCell>
				<TableCell align="right">{props.value ? "Yes" : "No"}</TableCell>
			</TableRow>
		);
	} else if (Array.isArray(props.value)) {
		return props.value.map((item, index) => (
			<ObjectField
				value={item}
				key={`${props.id}.${index}`}
				id={`${props.id}.${index}`}
				nkey={props.nkey}
			/>
		));
	} else if (typeof props.value === "object") {
		return Object.getOwnPropertyNames(props.value).map((sel, index) => {
			//console.log("ObjectField", sel, Object.getOwnPropertyDescriptor(props.value, sel)?.value)
			return (
				<ObjectField
					value={Object.getOwnPropertyDescriptor(props.value, sel)?.value}
					key={`${props.id}.${index}`}
					id={`${props.id}.${index}`}
					nkey={(props.nkey !== undefined ? `${props.nkey}.` : "") + sel}
				/>
			);
		});
	} else {
		return (
			<TableRow key={`unk.${props.id}`}>
				<TableCell>Unknown type: {typeof props.value}</TableCell>
			</TableRow>
		);
	}
}

export function ObjectTable(props: {
	object: object | Array<any> | null | undefined;
}) {
	return (
		<TableContainer component={Paper}>
			<Table stickyHeader aria-label="Property table" size="small">
				<TableHead>
					<TableRow>
						<TableCell>Property</TableCell>
						<TableCell align="right">Value</TableCell>
					</TableRow>
				</TableHead>
				<TableBody>
					{props.object ? (
						<ObjectField value={props.object || {}} id={"0"} />
					) : (
						<></>
					)}
				</TableBody>
			</Table>
		</TableContainer>
	);
}
