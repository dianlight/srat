import { Box } from "@mui/material";
import { useEffect } from "react";
import { apiUrl } from "../store/emptyApi";
import normalizeUrl from 'normalize-url';

// Allow the custom web component <openapi-explorer> in TSX
const OpenApiExplorer = 'openapi-explorer' as any;

export function Swagger() {
	useEffect(() => {
		// Lazily load the web component in real browsers; skip in tests to avoid happy-dom CSS engine issues
		if (typeof window !== 'undefined' && !(globalThis as any).__TEST__) {
			// @ts-ignore - module lacks published types, ambient module declared in global.d.ts
			import('openapi-explorer').catch(() => { /* ignore in non-browser envs */ });
		}
	}, []);

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
