import { Box } from "@mui/material";
import 'openapi-explorer';
import { apiUrl } from "../store/emptyApi";

export function Swagger() {
	/*
		return <InView as="div">
			<SwaggerUI url={apiUrl + "openapi-3.0.json"} />
		</InView>
		*/

	return (
		<Box>
			<>
				<openapi-explorer spec-url={`${apiUrl}/openapi.json`}>
					<div slot="overview">
						<h1>The API</h1>
					</div>
				</openapi-explorer>
				<a href="https://petstore.swagger.io/">Swagger Petstore</a>
				<p>{apiUrl}/openapi.json</p>
			</>
		</Box>
	);
}
