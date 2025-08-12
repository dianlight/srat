import { Box } from "@mui/material";
import 'openapi-explorer';
import { apiUrl } from "../store/emptyApi";

// Allow the custom web component <openapi-explorer> in TSX
const OpenApiExplorer = 'openapi-explorer' as any;

export function Swagger() {
	return (
		<Box>
			<OpenApiExplorer spec-url={`${apiUrl}/openapi.json`}>
				<div slot="overview">
					<h1>The API</h1>
				</div>
			</OpenApiExplorer>
		</Box>
	);
}
