import KeyboardArrowDownIcon from "@mui/icons-material/KeyboardArrowDown";
import KeyboardArrowUpIcon from "@mui/icons-material/KeyboardArrowUp";
import {
	Box,
	Collapse,
	IconButton,
	Paper,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Typography,
} from "@mui/material";
import { Fragment, useState } from "react";
import type { SambaStatus } from "../../../store/sratApi";

export function SambaStatusMetrics({
	sambaStatus,
}: {
	sambaStatus: SambaStatus | undefined;
}) {
	const [openSessions, setOpenSessions] = useState<Record<string, boolean>>({});

	if (!sambaStatus) {
		return <Typography>Samba status data not available.</Typography>;
	}

	const sessions = Object.entries(sambaStatus.sessions || {});
	const tcons = Object.entries(sambaStatus.tcons || {});
	const tconsBySessionId = tcons.reduce<Record<string, typeof tcons>>(
		(acc, [key, tcon]) => {
			const sessionTcons = acc[tcon.session_id] || [];
			sessionTcons.push([key, tcon]);
			acc[tcon.session_id] = sessionTcons;
			return acc;
		},
		{},
	);

	return (
		<>
			<Typography variant="h6" gutterBottom>
				Samba Sessions
			</Typography>
			{sessions.length > 0 ? (
				<TableContainer component={Paper}>
					<Table aria-label="samba sessions table" size="small">
						<TableHead>
							<TableRow>
								<TableCell />
								<TableCell>Session ID</TableCell>
								<TableCell align="right">Channels</TableCell>
								<TableCell>Hostname</TableCell>
								<TableCell>Remote Machine</TableCell>
								<TableCell>Username</TableCell>
								<TableCell>Encryption</TableCell>
								<TableCell>Signing</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{sessions.map(([key, session]) => {
								const relatedTcons = tconsBySessionId[session.session_id] || [];
								const isOpen = !!openSessions[session.session_id];

								return (
									<Fragment key={key}>
										<TableRow sx={{ "& > *": { borderBottom: "unset" } }}>
											<TableCell>
												<IconButton
													aria-label={`expand session ${session.session_id}`}
													size="small"
													onClick={() => {
														setOpenSessions((prev) => ({
															...prev,
															[session.session_id]: !prev[session.session_id],
														}));
													}}
												>
													{isOpen ? <KeyboardArrowUpIcon /> : <KeyboardArrowDownIcon />}
												</IconButton>
											</TableCell>
											<TableCell component="th" scope="row">
												{session.session_id}
											</TableCell>
											<TableCell align="right">
												{Object.entries(session.channels || {}).map(([channelKey, channel]) => (
													<span key={channelKey}>{channel.channel_id}({channel.transport}) </span>
												))}
											</TableCell>
											<TableCell>{session.hostname}</TableCell>
											<TableCell>{session.remote_machine}</TableCell>
											<TableCell>{session.username}</TableCell>
											<TableCell>{session.encryption?.cipher} {session.encryption?.degree}</TableCell>
											<TableCell>{session.signing?.cipher} {session.signing?.degree}</TableCell>
										</TableRow>
										<TableRow>
											<TableCell style={{ paddingBottom: 0, paddingTop: 0 }} colSpan={8}>
												<Collapse in={isOpen} timeout="auto" unmountOnExit>
													<Box sx={{ margin: 1 }}>
														<Typography variant="subtitle2" gutterBottom component="div">
															Tcons
														</Typography>
														{relatedTcons.length > 0 ? (
															<Table size="small" aria-label={`samba tcons subtable ${session.session_id}`}>
																<TableHead>
																	<TableRow>
																		<TableCell>Tcon ID</TableCell>
																		<TableCell>Machine</TableCell>
																		<TableCell>Service</TableCell>
																		<TableCell>Encryption</TableCell>
																		<TableCell>Signing</TableCell>
																	</TableRow>
																</TableHead>
																<TableBody>
																	{relatedTcons.map(([tconKey, tcon]) => (
																		<TableRow key={tconKey}>
																			<TableCell component="th" scope="row">{tcon.tcon_id}</TableCell>
																			<TableCell>{tcon.machine}</TableCell>
																			<TableCell>{tcon.service}</TableCell>
																			<TableCell>{tcon.encryption?.cipher} {tcon.encryption?.degree}</TableCell>
																			<TableCell>{tcon.signing?.cipher} {tcon.signing?.degree}</TableCell>
																		</TableRow>
																	))}
																</TableBody>
															</Table>
														) : (
															<Typography variant="caption" color="text.secondary">
																No Tcons for this session.
															</Typography>
														)}
													</Box>
												</Collapse>
											</TableCell>
										</TableRow>
									</Fragment>
								);
							})}
						</TableBody>
					</Table>
				</TableContainer>
			) : (
				<Typography>No active Samba sessions.</Typography>
			)}
		</>
	);
}
