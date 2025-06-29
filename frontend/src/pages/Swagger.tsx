import { Box } from "@mui/material";

export function Swagger() {
	/*
        return <InView as="div">
            <SwaggerUI url={apiUrl + "openapi-3.0.json"} />
        </InView>
        */

	return (
		<Box>
			<a href="https://petstore.swagger.io/">Swagger Petstore</a>
			<p>http://localhost:8090/openapi.json</p>
		</Box>
	);
}
