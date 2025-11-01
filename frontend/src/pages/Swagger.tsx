import { Box } from "@mui/material";
import { apiUrl } from "../store/emptyApi";
import normalizeUrl from 'normalize-url';
import { useEffect, useState } from 'react';

// Import Prism and make it globally available for openapi-explorer
if (typeof window !== 'undefined' && !(globalThis as any).__TEST__) {
	import('prismjs').then((Prism) => {
		(window as any).Prism = Prism.default || Prism;
		// Load the openapi-explorer after Prism is available
		import('openapi-explorer');
	});
}

// Allow the custom web component <openapi-explorer> in TSX
const OpenApiExplorer = 'openapi-explorer' as any;

export function Swagger() {
	const [loaded, setLoaded] = useState(false);

	// Wait for custom element to be defined
	useEffect(() => {
		if ((globalThis as any).__TEST__) {
			// In tests, mark as loaded immediately to show the overview
			setLoaded(true);
		} else if (typeof window !== 'undefined' && window.customElements) {
			// Wait for the custom element to be registered
			window.customElements.whenDefined('openapi-explorer')
				.then(() => {
					console.log('openapi-explorer custom element is ready');
					setLoaded(true);
				})
				.catch((err) => {
					console.error('Error waiting for openapi-explorer:', err);
					setLoaded(true);
				});

			// Fallback timeout in case whenDefined doesn't resolve
			const timeout = setTimeout(() => {
				console.log('openapi-explorer timeout, marking as loaded anyway');
				setLoaded(true);
			}, 2000);

			return () => clearTimeout(timeout);
		} else {
			setLoaded(true);
		}
	}, []);

	return (
		<Box sx={{ height: '100%', width: '100%', display: 'flex', flexDirection: 'column' }}>
			{loaded ? (
				<OpenApiExplorer
					spec-url={normalizeUrl(`${apiUrl}/openapi.yaml`)}
					style={{ flex: 1, minHeight: 0 }}
				>
					<div slot="overview">
						<h1>API Documentation</h1>
						<p>
							<a href={normalizeUrl(`${apiUrl}/openapi.json`)}>JSON</a> | <a href={normalizeUrl(`${apiUrl}/openapi.yaml`)}>YAML</a>
						</p>
					</div>
				</OpenApiExplorer>
			) : (
				<Box sx={{ p: 2 }}>
					<h1>API Documentation</h1>
					<p>Loading OpenAPI explorer...</p>
				</Box>
			)}
		</Box>
	);
}