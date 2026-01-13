import { useColorScheme } from "@mui/material/styles";
import { InView } from "react-intersection-observer";
import SyntaxHighlighter from "react-syntax-highlighter";
import {
	a11yDark,
	a11yLight,
} from "react-syntax-highlighter/dist/esm/styles/hljs";
import { type SmbConf, useGetApiSambaConfigQuery } from "../store/sratApi";
import { Box } from "@mui/material";
import { CopyButtonBar } from "../components/CopyButtonBar";
import { censorPlainText } from "../utils/censorshipUtils";
import { useMemo } from "react";

export function SmbConf() {
	const { mode } = useColorScheme();
	const smbconfig = useGetApiSambaConfigQuery();

	// Censor sensitive data in the config
	const configData = (smbconfig.data as SmbConf)?.data || "";
	const censoredData = useMemo(() => censorPlainText(configData), [configData]);

	return (
		<InView
			as="div"
			onChange={(inView, entry) => {
				console.log("Inview:", inView)
				smbconfig.refetch();
			}}
		>
			<Box sx={{ position: 'relative' }}>
				{/* Floating copy button bar */}
				<Box
					sx={{
						position: 'sticky',
						top: 0,
						zIndex: 10,
						backgroundColor: 'background.paper',
						borderBottom: 1,
						borderColor: 'divider',
						p: 1,
						display: 'flex',
						justifyContent: 'flex-end',
						gap: 1
					}}
				>
					<CopyButtonBar
						plainTextContent={censoredData}
						markdownContent={`\`\`\`ini\n${censoredData}\n\`\`\``}
						markdownTitle="Samba Configuration"
					/>
				</Box>
				
				<SyntaxHighlighter
					customStyle={{ fontSize: "0.7rem" }}
					language="ini"
					style={mode === "light" ? a11yLight : a11yDark}
					wrapLines
					wrapLongLines
					showInlineLineNumbers
					showLineNumbers
					useInlineStyles
				>
					{smbconfig.isLoading ? "...Loading..." : censoredData}
				</SyntaxHighlighter>
			</Box>
		</InView >
	);
}
