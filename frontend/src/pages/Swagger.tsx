import { Box } from "@mui/material";
import 'openapi-explorer';
import { apiUrl } from "../store/emptyApi";
import normalizeUrl from 'normalize-url';

// Allow the custom web component <openapi-explorer> in TSX
const OpenApiExplorer = 'openapi-explorer' as any;

export function Swagger() {
	return (
		<Box>
			<OpenApiExplorer spec-url={normalizeUrl(`${apiUrl}/openapi.yaml`)}>
				<div slot="overview">
					<h1>API Documentation</h1>
					<p>
						<a href={normalizeUrl(`${apiUrl}/openapi.json`)}>JSON</a> | <a href={normalizeUrl(`${apiUrl}/openapi.yaml`)}>YAML</a>
					</p>
				</div>
			</OpenApiExplorer>
		</Box>
	);
}
