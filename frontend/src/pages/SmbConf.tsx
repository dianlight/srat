import { useColorScheme } from "@mui/material/styles";
import { InView } from "react-intersection-observer";
import SyntaxHighlighter from "react-syntax-highlighter";
import {
	a11yDark,
	a11yLight,
} from "react-syntax-highlighter/dist/esm/styles/hljs";
import { type SmbConf, useGetApiSambaConfigQuery } from "../store/sratApi";

export function SmbConf() {
	const { mode } = useColorScheme();
	const smbconfig = useGetApiSambaConfigQuery();

	return (
		<InView
			as="div"
			onChange={(inView, entry) => {
				console.log("Inview:", inView)
				smbconfig.refetch();
			}}

		>
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
				{smbconfig.isLoading ? "...Loading..." : (smbconfig.data as SmbConf)?.data || ""}
			</SyntaxHighlighter>
		</InView >
	);
}
